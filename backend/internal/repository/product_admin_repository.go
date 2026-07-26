package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"sigefer.local/backend/internal/models"
)

var (
	ErrProductNotFound = errors.New(
		"producto no encontrado",
	)

	ErrProductCodeExists = errors.New(
		"el código de producto ya existe",
	)

	ErrProductCategoryNotFound = errors.New(
		"categoría no encontrada o inactiva",
	)
)

// ListCategories devuelve las categorías activas.
func (repository *ProductRepository) ListCategories(
	ctx context.Context,
) ([]models.Category, error) {
	rows, err := repository.db.QueryContext(
		ctx,
		`
			SELECT
				ID_CATEGORIA,
				NOMBRE,
				DESCRIPCION,
				ESTADO
			FROM CATEGORIA
			WHERE ESTADO = 'A'
			ORDER BY NOMBRE
		`,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"no se pudieron consultar las categorías: %w",
			err,
		)
	}
	defer rows.Close()

	categories := make(
		[]models.Category,
		0,
	)

	for rows.Next() {
		var (
			category    models.Category
			description sql.NullString
		)

		if err := rows.Scan(
			&category.IDCategoria,
			&category.Nombre,
			&description,
			&category.Estado,
		); err != nil {
			return nil, fmt.Errorf(
				"no se pudo interpretar una categoría: %w",
				err,
			)
		}

		if description.Valid {
			category.Descripcion =
				description.String
		}

		categories = append(
			categories,
			category,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"error recorriendo categorías: %w",
			err,
		)
	}

	return categories, nil
}

// GetByCode devuelve un producto activo o inactivo.
func (repository *ProductRepository) GetByCode(
	ctx context.Context,
	code string,
) (models.ProductDetail, error) {
	row := repository.db.QueryRowContext(
		ctx,
		`
			SELECT
				p.ID_PRODUCTO,
				p.ID_CATEGORIA,
				p.CODIGO,
				p.NOMBRE,
				p.DESCRIPCION,
				c.NOMBRE AS CATEGORIA,
				p.UNIDAD_MEDIDA,
				p.ESTADO,
				i.UBICACION,
				p.PRECIO_COMPRA,
				p.PRECIO_VENTA,
				ROUND(
					p.PRECIO_VENTA -
					p.PRECIO_COMPRA,
					2
				),
				i.STOCK_ACTUAL,
				i.STOCK_RESERVADO,
				i.STOCK_DISPONIBLE,
				p.STOCK_MINIMO,
				CASE
					WHEN i.STOCK_DISPONIBLE <= 0
						THEN 'AGOTADO'
					WHEN i.STOCK_DISPONIBLE <=
						p.STOCK_MINIMO
						THEN 'STOCK_BAJO'
					ELSE 'NORMAL'
				END,
				TO_CHAR(
					p.FECHA_CREACION,
					'YYYY-MM-DD"T"HH24:MI:SS'
				),
				TO_CHAR(
					p.FECHA_ACTUALIZACION,
					'YYYY-MM-DD"T"HH24:MI:SS'
				)
			FROM PRODUCTO p
			INNER JOIN CATEGORIA c
				ON c.ID_CATEGORIA =
					p.ID_CATEGORIA
			INNER JOIN INVENTARIO i
				ON i.ID_PRODUCTO =
					p.ID_PRODUCTO
			WHERE p.CODIGO = :1
		`,
		code,
	)

	product, err :=
		scanProductDetail(row)

	if errors.Is(err, sql.ErrNoRows) {
		return models.ProductDetail{},
			ErrProductNotFound
	}

	if err != nil {
		return models.ProductDetail{},
			fmt.Errorf(
				"no se pudo consultar el producto: %w",
				err,
			)
	}

	return product, nil
}

// CreateProduct crea PRODUCTO e INVENTARIO dentro
// de una única transacción.
func (repository *ProductRepository) CreateProduct(
	ctx context.Context,
	request models.CreateProductRequest,
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
				"no se pudo iniciar la transacción: %w",
				err,
			)
	}

	committed := false

	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

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

	var codeCount int

	err = tx.QueryRowContext(
		ctx,
		`
			SELECT COUNT(*)
			FROM PRODUCTO
			WHERE CODIGO = :1
		`,
		request.Codigo,
	).Scan(&codeCount)

	if err != nil {
		return models.ProductDetail{},
			fmt.Errorf(
				"no se pudo validar el código: %w",
				err,
			)
	}

	if codeCount > 0 {
		return models.ProductDetail{},
			ErrProductCodeExists
	}

	_, err = tx.ExecContext(
		ctx,
		`
			INSERT INTO PRODUCTO (
				ID_CATEGORIA,
				CODIGO,
				NOMBRE,
				DESCRIPCION,
				UNIDAD_MEDIDA,
				PRECIO_COMPRA,
				PRECIO_VENTA,
				STOCK_MINIMO,
				ESTADO,
				FECHA_CREACION,
				FECHA_ACTUALIZACION
			)
			VALUES (
				:1,
				:2,
				:3,
				:4,
				:5,
				:6,
				:7,
				:8,
				'A',
				CAST(
					SYSTIMESTAMP
					AT TIME ZONE '-05:00'
					AS TIMESTAMP
				),
				CAST(
					SYSTIMESTAMP
					AT TIME ZONE '-05:00'
					AS TIMESTAMP
				)
			)
		`,
		request.IDCategoria,
		request.Codigo,
		request.Nombre,
		nullableText(request.Descripcion),
		request.UnidadMedida,
		request.PrecioCompra.InexactFloat64(),
		request.PrecioVenta.InexactFloat64(),
		request.StockMinimo.InexactFloat64(),
	)
	if err != nil {
		if strings.Contains(
			err.Error(),
			"ORA-00001",
		) {
			return models.ProductDetail{},
				ErrProductCodeExists
		}

		return models.ProductDetail{},
			fmt.Errorf(
				"no se pudo insertar el producto: %w",
				err,
			)
	}

	var productID int64

	err = tx.QueryRowContext(
		ctx,
		`
			SELECT ID_PRODUCTO
			FROM PRODUCTO
			WHERE CODIGO = :1
		`,
		request.Codigo,
	).Scan(&productID)

	if err != nil {
		return models.ProductDetail{},
			fmt.Errorf(
				"no se pudo recuperar el producto: %w",
				err,
			)
	}

	_, err = tx.ExecContext(
		ctx,
		`
			INSERT INTO INVENTARIO (
				ID_PRODUCTO,
				STOCK_ACTUAL,
				STOCK_RESERVADO,
				UBICACION,
				FECHA_ULTIMO_MOVIMIENTO,
				FECHA_ACTUALIZACION
			)
			VALUES (
				:1,
				:2,
				0,
				:3,
				CASE
					WHEN :2 > 0
					THEN CAST(
						SYSTIMESTAMP
						AT TIME ZONE '-05:00'
						AS TIMESTAMP
					)
					ELSE NULL
				END,
				CAST(
					SYSTIMESTAMP
					AT TIME ZONE '-05:00'
					AS TIMESTAMP
				)
			)
		`,
		productID,
		request.StockInicial.InexactFloat64(),
		nullableText(request.Ubicacion),
	)
	if err != nil {
		return models.ProductDetail{},
			fmt.Errorf(
				"no se pudo crear el inventario: %w",
				err,
			)
	}

	if err := insertProductCreationAudit(
		ctx,
		tx,
		productID,
		userID,
		request,
		ipAddress,
	); err != nil {
		return models.ProductDetail{}, err
	}

	if err := tx.Commit(); err != nil {
		return models.ProductDetail{},
			fmt.Errorf(
				"no se pudo confirmar el producto: %w",
				err,
			)
	}

	committed = true

	return repository.GetByCode(
		ctx,
		request.Codigo,
	)
}

func productCategoryIsActive(
	ctx context.Context,
	tx *sql.Tx,
	categoryID int64,
) (bool, error) {
	var total int

	err := tx.QueryRowContext(
		ctx,
		`
			SELECT COUNT(*)
			FROM CATEGORIA
			WHERE ID_CATEGORIA = :1
			  AND ESTADO = 'A'
		`,
		categoryID,
	).Scan(&total)

	if err != nil {
		return false, fmt.Errorf(
			"no se pudo validar la categoría: %w",
			err,
		)
	}

	return total == 1, nil
}

type productDetailScanner interface {
	Scan(dest ...any) error
}

func scanProductDetail(
	scanner productDetailScanner,
) (models.ProductDetail, error) {
	var (
		product     models.ProductDetail
		description sql.NullString
		location    sql.NullString

		purchasePrice  float64
		salePrice      float64
		margin         float64
		currentStock   float64
		reservedStock  float64
		availableStock float64
		minimumStock   float64
	)

	err := scanner.Scan(
		&product.IDProducto,
		&product.IDCategoria,
		&product.Codigo,
		&product.Nombre,
		&description,
		&product.Categoria,
		&product.UnidadMedida,
		&product.Estado,
		&location,
		&purchasePrice,
		&salePrice,
		&margin,
		&currentStock,
		&reservedStock,
		&availableStock,
		&minimumStock,
		&product.EstadoStock,
		&product.FechaCreacion,
		&product.FechaActualizacion,
	)
	if err != nil {
		return models.ProductDetail{}, err
	}

	if description.Valid {
		product.Descripcion =
			description.String
	}

	if location.Valid {
		product.Ubicacion =
			location.String
	}

	product.PrecioCompra =
		decimal.NewFromFloat(
			purchasePrice,
		).StringFixed(2)

	product.PrecioVenta =
		decimal.NewFromFloat(
			salePrice,
		).StringFixed(2)

	product.MargenUnitario =
		decimal.NewFromFloat(
			margin,
		).StringFixed(2)

	product.StockActual =
		decimal.NewFromFloat(
			currentStock,
		).StringFixed(3)

	product.StockReservado =
		decimal.NewFromFloat(
			reservedStock,
		).StringFixed(3)

	product.StockDisponible =
		decimal.NewFromFloat(
			availableStock,
		).StringFixed(3)

	product.StockMinimo =
		decimal.NewFromFloat(
			minimumStock,
		).StringFixed(3)

	return product, nil
}

func insertProductCreationAudit(
	ctx context.Context,
	tx *sql.Tx,
	productID int64,
	userID int64,
	request models.CreateProductRequest,
	ipAddress string,
) error {
	values, err := json.Marshal(
		map[string]any{
			"id_categoria": request.IDCategoria,

			"codigo": request.Codigo,

			"nombre": request.Nombre,

			"descripcion": request.Descripcion,

			"unidad_medida": request.UnidadMedida,

			"precio_compra": request.PrecioCompra.StringFixed(2),

			"precio_venta": request.PrecioVenta.StringFixed(2),

			"stock_minimo": request.StockMinimo.StringFixed(3),

			"stock_inicial": request.StockInicial.StringFixed(3),

			"ubicacion": request.Ubicacion,

			"estado": "A",
		},
	)
	if err != nil {
		return fmt.Errorf(
			"no se pudo construir la auditoría: %w",
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
				VALORES_NUEVOS,
				IP_ORIGEN,
				ORIGEN,
				FECHA_EVENTO
			)
			VALUES (
				:1,
				'PRODUCTO',
				'INSERT',
				:2,
				:3,
				:4,
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
		string(values),
		nullableText(ipAddress),
	)
	if err != nil {
		return fmt.Errorf(
			"no se pudo auditar el producto: %w",
			err,
		)
	}

	return nil
}
