package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/shopspring/decimal"

	"sigefer.local/backend/internal/models"
)

type editableProductSnapshot struct {
	IDProducto   int64
	IDCategoria  int64
	Nombre       string
	Descripcion  string
	UnidadMedida string
	Ubicacion    string
	PrecioCompra decimal.Decimal
	PrecioVenta  decimal.Decimal
	StockMinimo  decimal.Decimal
	Estado       string
}

// UpdateProduct actualiza la información comercial
// y ubicación de un producto sin modificar existencias.
func (repository *ProductRepository) UpdateProduct(
	ctx context.Context,
	code string,
	request models.UpdateProductRequest,
	userID int64,
	ipAddress string,
) (models.ProductDetail, error) {
	tx, err := repository.db.BeginTx(
		ctx,
		nil,
	)
	if err != nil {
		return models.ProductDetail{},
			fmt.Errorf(
				"no se pudo iniciar la edición del producto: %w",
				err,
			)
	}

	committed := false

	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	current, err := lockEditableProduct(
		ctx,
		tx,
		code,
	)
	if err != nil {
		return models.ProductDetail{}, err
	}

	activeCategory, err :=
		productCategoryIsActive(
			ctx,
			tx,
			request.IDCategoria,
		)
	if err != nil {
		return models.ProductDetail{}, err
	}

	if !activeCategory {
		return models.ProductDetail{},
			ErrProductCategoryNotFound
	}

	_, err = tx.ExecContext(
		ctx,
		`
			UPDATE PRODUCTO
			SET
				ID_CATEGORIA = :1,
				NOMBRE = :2,
				DESCRIPCION = :3,
				UNIDAD_MEDIDA = :4,
				PRECIO_COMPRA = :5,
				PRECIO_VENTA = :6,
				STOCK_MINIMO = :7,
				FECHA_ACTUALIZACION =
					CAST(
						SYSTIMESTAMP
						AT TIME ZONE '-05:00'
						AS TIMESTAMP
					)
			WHERE ID_PRODUCTO = :8
		`,
		request.IDCategoria,
		request.Nombre,
		nullableText(request.Descripcion),
		request.UnidadMedida,
		request.PrecioCompra.InexactFloat64(),
		request.PrecioVenta.InexactFloat64(),
		request.StockMinimo.InexactFloat64(),
		current.IDProducto,
	)
	if err != nil {
		return models.ProductDetail{},
			fmt.Errorf(
				"no se pudo actualizar el producto: %w",
				err,
			)
	}

	_, err = tx.ExecContext(
		ctx,
		`
			UPDATE INVENTARIO
			SET
				UBICACION = :1,
				FECHA_ACTUALIZACION =
					CAST(
						SYSTIMESTAMP
						AT TIME ZONE '-05:00'
						AS TIMESTAMP
					)
			WHERE ID_PRODUCTO = :2
		`,
		nullableText(request.Ubicacion),
		current.IDProducto,
	)
	if err != nil {
		return models.ProductDetail{},
			fmt.Errorf(
				"no se pudo actualizar la ubicación: %w",
				err,
			)
	}

	previousValues := map[string]any{
		"id_categoria":  current.IDCategoria,
		"nombre":        current.Nombre,
		"descripcion":   current.Descripcion,
		"unidad_medida": current.UnidadMedida,
		"precio_compra": current.PrecioCompra.StringFixed(2),
		"precio_venta":  current.PrecioVenta.StringFixed(2),
		"stock_minimo":  current.StockMinimo.StringFixed(3),
		"ubicacion":     current.Ubicacion,
		"estado":        current.Estado,
	}

	newValues := map[string]any{
		"id_categoria":  request.IDCategoria,
		"nombre":        request.Nombre,
		"descripcion":   request.Descripcion,
		"unidad_medida": request.UnidadMedida,
		"precio_compra": request.PrecioCompra.StringFixed(2),
		"precio_venta":  request.PrecioVenta.StringFixed(2),
		"stock_minimo":  request.StockMinimo.StringFixed(3),
		"ubicacion":     request.Ubicacion,
		"estado":        current.Estado,
	}

	if err := insertProductUpdateAudit(
		ctx,
		tx,
		current.IDProducto,
		userID,
		previousValues,
		newValues,
		ipAddress,
	); err != nil {
		return models.ProductDetail{}, err
	}

	if err := tx.Commit(); err != nil {
		return models.ProductDetail{},
			fmt.Errorf(
				"no se pudo confirmar la edición: %w",
				err,
			)
	}

	committed = true

	return repository.GetByCode(
		ctx,
		code,
	)
}

// UpdateProductState activa o desactiva lógicamente
// un producto sin eliminar sus registros históricos.
func (repository *ProductRepository) UpdateProductState(
	ctx context.Context,
	code string,
	newState string,
	userID int64,
	ipAddress string,
) (models.ProductDetail, error) {
	tx, err := repository.db.BeginTx(
		ctx,
		nil,
	)
	if err != nil {
		return models.ProductDetail{},
			fmt.Errorf(
				"no se pudo iniciar el cambio de estado: %w",
				err,
			)
	}

	committed := false

	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	var (
		productID    int64
		currentState string
	)

	err = tx.QueryRowContext(
		ctx,
		`
			SELECT
				ID_PRODUCTO,
				ESTADO
			FROM PRODUCTO
			WHERE CODIGO = :1
			FOR UPDATE
		`,
		code,
	).Scan(
		&productID,
		&currentState,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return models.ProductDetail{},
			ErrProductNotFound
	}

	if err != nil {
		return models.ProductDetail{},
			fmt.Errorf(
				"no se pudo bloquear el producto: %w",
				err,
			)
	}

	if currentState == newState {
		if err := tx.Rollback(); err != nil {
			return models.ProductDetail{},
				fmt.Errorf(
					"no se pudo finalizar el cambio sin modificaciones: %w",
					err,
				)
		}

		committed = true

		return repository.GetByCode(
			ctx,
			code,
		)
	}

	_, err = tx.ExecContext(
		ctx,
		`
			UPDATE PRODUCTO
			SET
				ESTADO = :1,
				FECHA_ACTUALIZACION =
					CAST(
						SYSTIMESTAMP
						AT TIME ZONE '-05:00'
						AS TIMESTAMP
					)
			WHERE ID_PRODUCTO = :2
		`,
		newState,
		productID,
	)
	if err != nil {
		return models.ProductDetail{},
			fmt.Errorf(
				"no se pudo cambiar el estado del producto: %w",
				err,
			)
	}

	previousValues := map[string]any{
		"estado": currentState,
	}

	newValues := map[string]any{
		"estado": newState,
	}

	if err := insertProductUpdateAudit(
		ctx,
		tx,
		productID,
		userID,
		previousValues,
		newValues,
		ipAddress,
	); err != nil {
		return models.ProductDetail{}, err
	}

	if err := tx.Commit(); err != nil {
		return models.ProductDetail{},
			fmt.Errorf(
				"no se pudo confirmar el cambio de estado: %w",
				err,
			)
	}

	committed = true

	return repository.GetByCode(
		ctx,
		code,
	)
}

func lockEditableProduct(
	ctx context.Context,
	tx *sql.Tx,
	code string,
) (editableProductSnapshot, error) {
	var (
		snapshot editableProductSnapshot

		description sql.NullString
		location    sql.NullString

		purchasePrice float64
		salePrice     float64
		minimumStock  float64
	)

	err := tx.QueryRowContext(
		ctx,
		`
			SELECT
				p.ID_PRODUCTO,
				p.ID_CATEGORIA,
				p.NOMBRE,
				p.DESCRIPCION,
				p.UNIDAD_MEDIDA,
				p.PRECIO_COMPRA,
				p.PRECIO_VENTA,
				p.STOCK_MINIMO,
				p.ESTADO,
				i.UBICACION
			FROM PRODUCTO p
			INNER JOIN INVENTARIO i
				ON i.ID_PRODUCTO =
					p.ID_PRODUCTO
			WHERE p.CODIGO = :1
			FOR UPDATE
		`,
		code,
	).Scan(
		&snapshot.IDProducto,
		&snapshot.IDCategoria,
		&snapshot.Nombre,
		&description,
		&snapshot.UnidadMedida,
		&purchasePrice,
		&salePrice,
		&minimumStock,
		&snapshot.Estado,
		&location,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return editableProductSnapshot{},
			ErrProductNotFound
	}

	if err != nil {
		return editableProductSnapshot{},
			fmt.Errorf(
				"no se pudo bloquear el producto: %w",
				err,
			)
	}

	if description.Valid {
		snapshot.Descripcion =
			description.String
	}

	if location.Valid {
		snapshot.Ubicacion =
			location.String
	}

	snapshot.PrecioCompra =
		decimal.NewFromFloat(
			purchasePrice,
		)

	snapshot.PrecioVenta =
		decimal.NewFromFloat(
			salePrice,
		)

	snapshot.StockMinimo =
		decimal.NewFromFloat(
			minimumStock,
		)

	return snapshot, nil
}

func insertProductUpdateAudit(
	ctx context.Context,
	tx *sql.Tx,
	productID int64,
	userID int64,
	previousValues any,
	newValues any,
	ipAddress string,
) error {
	previousJSON, err :=
		json.Marshal(previousValues)

	if err != nil {
		return fmt.Errorf(
			"no se pudo construir la auditoría anterior: %w",
			err,
		)
	}

	newJSON, err :=
		json.Marshal(newValues)

	if err != nil {
		return fmt.Errorf(
			"no se pudo construir la auditoría nueva: %w",
			err,
		)
	}

	_, err = tx.ExecContext(
		ctx,
		`
			INSERT INTO AUDITORIA (
				ID_USUARIO,
				TABLA_AFECTADA,
				OPERACION,
				ID_REGISTRO,
				VALORES_ANTERIORES,
				VALORES_NUEVOS,
				IP_ORIGEN,
				ORIGEN,
				FECHA_EVENTO
			)
			VALUES (
				:1,
				'PRODUCTO',
				'UPDATE',
				:2,
				:3,
				:4,
				:5,
				'API REST PRODUCTOS',
				CAST(
					SYSTIMESTAMP
					AT TIME ZONE '-05:00'
					AS TIMESTAMP
				)
			)
		`,
		userID,
		fmt.Sprintf("%d", productID),
		string(previousJSON),
		string(newJSON),
		nullableText(ipAddress),
	)
	if err != nil {
		return fmt.Errorf(
			"no se pudo auditar la actualización: %w",
			err,
		)
	}

	return nil
}
