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
	ErrInventoryProductInactive = errors.New(
		"el producto está inactivo",
	)

	ErrInventoryReservedConflict = errors.New(
		"el stock resultante sería menor que el stock reservado",
	)

	ErrInventoryAdjustmentType = errors.New(
		"tipo de ajuste no permitido",
	)
)

type lockedInventoryProduct struct {
	IDProducto   int64
	IDInventario int64

	Codigo string
	Nombre string
	Estado string

	StockActual    decimal.Decimal
	StockReservado decimal.Decimal
	StockMinimo    decimal.Decimal
}

// AdjustInventory registra un ajuste positivo o negativo
// dentro de una única transacción Oracle.
func (repository *ProductRepository) AdjustInventory(
	ctx context.Context,
	request models.InventoryAdjustmentRequest,
	userID int64,
	ipAddress string,
) (models.InventoryAdjustmentResult, error) {
	tx, err := repository.db.BeginTx(
		ctx,
		nil,
	)
	if err != nil {
		return models.InventoryAdjustmentResult{},
			fmt.Errorf(
				"no se pudo iniciar el ajuste: %w",
				err,
			)
	}

	committed := false

	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	product, err := lockInventoryProduct(
		ctx,
		tx,
		request.CodigoProducto,
	)
	if err != nil {
		return models.InventoryAdjustmentResult{},
			err
	}

	if product.Estado != "A" {
		return models.InventoryAdjustmentResult{},
			ErrInventoryProductInactive
	}

	movementType := ""
	newCurrentStock := product.StockActual

	switch request.TipoAjuste {
	case "POSITIVO":
		movementType = "AJUSTE_POSITIVO"
		newCurrentStock =
			product.StockActual.Add(
				request.Cantidad,
			)

	case "NEGATIVO":
		movementType = "AJUSTE_NEGATIVO"
		newCurrentStock =
			product.StockActual.Sub(
				request.Cantidad,
			)

	default:
		return models.InventoryAdjustmentResult{},
			ErrInventoryAdjustmentType
	}

	if newCurrentStock.LessThan(
		product.StockReservado,
	) {
		return models.InventoryAdjustmentResult{},
			ErrInventoryReservedConflict
	}

	newAvailableStock :=
		newCurrentStock.Sub(
			product.StockReservado,
		)

	updateResult, err := tx.ExecContext(
		ctx,
		`
			UPDATE INVENTARIO
			SET
				STOCK_ACTUAL = :1,
				FECHA_ULTIMO_MOVIMIENTO =
					CAST(
						SYSTIMESTAMP
						AT TIME ZONE '-05:00'
						AS TIMESTAMP
					),
				FECHA_ACTUALIZACION =
					CAST(
						SYSTIMESTAMP
						AT TIME ZONE '-05:00'
						AS TIMESTAMP
					)
			WHERE ID_INVENTARIO = :2
		`,
		newCurrentStock.InexactFloat64(),
		product.IDInventario,
	)
	if err != nil {
		return models.InventoryAdjustmentResult{},
			fmt.Errorf(
				"no se pudo actualizar el inventario: %w",
				err,
			)
	}

	affectedRows, err :=
		updateResult.RowsAffected()

	if err != nil {
		return models.InventoryAdjustmentResult{},
			fmt.Errorf(
				"no se pudo verificar el inventario: %w",
				err,
			)
	}

	if affectedRows != 1 {
		return models.InventoryAdjustmentResult{},
			fmt.Errorf(
				"se esperaba actualizar un inventario",
			)
	}

	movementID, err := insertInventoryAdjustmentMovement(
		ctx,
		tx,
		product.IDProducto,
		userID,
		movementType,
		request.Cantidad,
		product.StockActual,
		newCurrentStock,
		request.Motivo,
	)
	if err != nil {
		return models.InventoryAdjustmentResult{},
			err
	}

	alertResult, err := synchronizeInventoryAlert(
		ctx,
		tx,
		product,
		movementID,
		newAvailableStock,
		userID,
	)
	if err != nil {
		return models.InventoryAdjustmentResult{},
			err
	}

	if err := insertInventoryAdjustmentAudit(
		ctx,
		tx,
		product,
		movementID,
		movementType,
		request,
		newCurrentStock,
		newAvailableStock,
		userID,
		ipAddress,
	); err != nil {
		return models.InventoryAdjustmentResult{},
			err
	}

	movementDate, err :=
		findInventoryMovementDate(
			ctx,
			tx,
			movementID,
		)

	if err != nil {
		return models.InventoryAdjustmentResult{},
			err
	}

	if err := tx.Commit(); err != nil {
		return models.InventoryAdjustmentResult{},
			fmt.Errorf(
				"no se pudo confirmar el ajuste: %w",
				err,
			)
	}

	committed = true

	stateStock := "NORMAL"

	if newAvailableStock.IsZero() {
		stateStock = "AGOTADO"
	} else if newAvailableStock.LessThanOrEqual(
		product.StockMinimo,
	) {
		stateStock = "STOCK_BAJO"
	}

	return models.InventoryAdjustmentResult{
		Status:       "ok",
		IDMovimiento: movementID,

		CodigoProducto: product.Codigo,
		Producto:       product.Nombre,
		TipoMovimiento: movementType,

		Cantidad: request.Cantidad.StringFixed(3),

		StockAnterior: product.StockActual.StringFixed(3),

		StockNuevo: newCurrentStock.StringFixed(3),

		StockReservado: product.StockReservado.StringFixed(3),

		StockDisponible: newAvailableStock.StringFixed(3),

		StockMinimo: product.StockMinimo.StringFixed(3),

		EstadoStock:     stateStock,
		ResultadoAlerta: alertResult,
		Motivo:          request.Motivo,
		FechaMovimiento: movementDate,
	}, nil
}

// ListInventoryMovements devuelve el historial
// con filtros opcionales.
func (repository *ProductRepository) ListInventoryMovements(
	ctx context.Context,
	limit int,
	code string,
	movementType string,
) ([]models.InventoryMovement, error) {
	query := `
		SELECT *
		FROM (
			SELECT
				m.ID_MOVIMIENTO,
				m.ID_PRODUCTO,
				p.CODIGO,
				p.NOMBRE,
				m.TIPO_MOVIMIENTO,
				m.CANTIDAD,
				m.STOCK_ANTERIOR,
				m.STOCK_NUEVO,
				m.MOTIVO,
				m.ID_USUARIO,
				NVL(
					u.NOMBRE_USUARIO,
					'SISTEMA'
				) AS USUARIO,
				m.ID_VENTA,
				m.ID_SOLICITUD_REPOSICION,
				CASE
					WHEN m.ID_VENTA IS NOT NULL
						THEN v.NUMERO_VENTA
					WHEN m.ID_SOLICITUD_REPOSICION
						IS NOT NULL
						THEN sr.NUMERO_SOLICITUD
					ELSE NULL
				END AS REFERENCIA,
				TO_CHAR(
					m.FECHA_MOVIMIENTO,
					'YYYY-MM-DD"T"HH24:MI:SS'
				) AS FECHA_MOVIMIENTO
			FROM MOVIMIENTO_INVENTARIO m
			INNER JOIN PRODUCTO p
				ON p.ID_PRODUCTO =
					m.ID_PRODUCTO
			LEFT JOIN USUARIO u
				ON u.ID_USUARIO =
					m.ID_USUARIO
			LEFT JOIN VENTA v
				ON v.ID_VENTA =
					m.ID_VENTA
			LEFT JOIN SOLICITUD_REPOSICION sr
				ON sr.ID_SOLICITUD =
					m.ID_SOLICITUD_REPOSICION
			WHERE 1 = 1
	`

	args := make([]any, 0, 3)

	if code != "" {
		args = append(args, code)

		query += fmt.Sprintf(
			"\n AND p.CODIGO = :%d",
			len(args),
		)
	}

	if movementType != "" {
		args = append(
			args,
			movementType,
		)

		query += fmt.Sprintf(
			"\n AND m.TIPO_MOVIMIENTO = :%d",
			len(args),
		)
	}

	query += `
			ORDER BY m.ID_MOVIMIENTO DESC
		)
	`

	args = append(args, limit)

	query += fmt.Sprintf(
		"\n WHERE ROWNUM <= :%d",
		len(args),
	)

	rows, err := repository.db.QueryContext(
		ctx,
		query,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"no se pudieron consultar los movimientos: %w",
			err,
		)
	}
	defer rows.Close()

	movements := make(
		[]models.InventoryMovement,
		0,
	)

	for rows.Next() {
		var (
			movement models.InventoryMovement

			quantity      float64
			previousStock float64
			newStock      float64

			reason    sql.NullString
			userID    sql.NullInt64
			saleID    sql.NullInt64
			requestID sql.NullInt64
			reference sql.NullString
		)

		err := rows.Scan(
			&movement.IDMovimiento,
			&movement.IDProducto,
			&movement.CodigoProducto,
			&movement.Producto,
			&movement.TipoMovimiento,
			&quantity,
			&previousStock,
			&newStock,
			&reason,
			&userID,
			&movement.Usuario,
			&saleID,
			&requestID,
			&reference,
			&movement.FechaMovimiento,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"no se pudo interpretar un movimiento: %w",
				err,
			)
		}

		movement.Cantidad =
			decimal.NewFromFloat(
				quantity,
			).StringFixed(3)

		movement.StockAnterior =
			decimal.NewFromFloat(
				previousStock,
			).StringFixed(3)

		movement.StockNuevo =
			decimal.NewFromFloat(
				newStock,
			).StringFixed(3)

		if reason.Valid {
			movement.Motivo =
				reason.String
		}

		if userID.Valid {
			value := userID.Int64
			movement.IDUsuario = &value
		}

		if saleID.Valid {
			value := saleID.Int64
			movement.IDVenta = &value
		}

		if requestID.Valid {
			value := requestID.Int64
			movement.IDSolicitudReposicion =
				&value
		}

		if reference.Valid {
			movement.Referencia =
				reference.String
		}

		movements = append(
			movements,
			movement,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"error recorriendo movimientos: %w",
			err,
		)
	}

	return movements, nil
}

func lockInventoryProduct(
	ctx context.Context,
	tx *sql.Tx,
	code string,
) (lockedInventoryProduct, error) {
	var (
		product lockedInventoryProduct

		currentStock  float64
		reservedStock float64
		minimumStock  float64
	)

	err := tx.QueryRowContext(
		ctx,
		`
			SELECT
				p.ID_PRODUCTO,
				i.ID_INVENTARIO,
				p.CODIGO,
				p.NOMBRE,
				p.ESTADO,
				i.STOCK_ACTUAL,
				i.STOCK_RESERVADO,
				p.STOCK_MINIMO
			FROM PRODUCTO p
			INNER JOIN INVENTARIO i
				ON i.ID_PRODUCTO =
					p.ID_PRODUCTO
			WHERE p.CODIGO = :1
			FOR UPDATE
		`,
		code,
	).Scan(
		&product.IDProducto,
		&product.IDInventario,
		&product.Codigo,
		&product.Nombre,
		&product.Estado,
		&currentStock,
		&reservedStock,
		&minimumStock,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return lockedInventoryProduct{},
			ErrProductNotFound
	}

	if err != nil {
		return lockedInventoryProduct{},
			fmt.Errorf(
				"no se pudo bloquear el inventario: %w",
				err,
			)
	}

	product.StockActual =
		decimal.NewFromFloat(
			currentStock,
		)

	product.StockReservado =
		decimal.NewFromFloat(
			reservedStock,
		)

	product.StockMinimo =
		decimal.NewFromFloat(
			minimumStock,
		)

	return product, nil
}

func insertInventoryAdjustmentMovement(
	ctx context.Context,
	tx *sql.Tx,
	productID int64,
	userID int64,
	movementType string,
	quantity decimal.Decimal,
	previousStock decimal.Decimal,
	newStock decimal.Decimal,
	reason string,
) (int64, error) {
	var movementID int64

	_, err := tx.ExecContext(
		ctx,
		`
			INSERT INTO MOVIMIENTO_INVENTARIO (
				ID_PRODUCTO,
				ID_USUARIO,
				TIPO_MOVIMIENTO,
				CANTIDAD,
				STOCK_ANTERIOR,
				STOCK_NUEVO,
				MOTIVO,
				FECHA_MOVIMIENTO
			)
			VALUES (
				:1,
				:2,
				:3,
				:4,
				:5,
				:6,
				:7,
				CAST(
					SYSTIMESTAMP
					AT TIME ZONE '-05:00'
					AS TIMESTAMP
				)
			)
			RETURNING ID_MOVIMIENTO
			INTO :8
		`,
		productID,
		userID,
		movementType,
		quantity.InexactFloat64(),
		previousStock.InexactFloat64(),
		newStock.InexactFloat64(),
		reason,
		sql.Out{
			Dest: &movementID,
		},
	)
	if err != nil {
		return 0, fmt.Errorf(
			"no se pudo registrar el movimiento: %w",
			err,
		)
	}

	return movementID, nil
}

func synchronizeInventoryAlert(
	ctx context.Context,
	tx *sql.Tx,
	product lockedInventoryProduct,
	movementID int64,
	newAvailableStock decimal.Decimal,
	userID int64,
) (string, error) {
	if newAvailableStock.LessThanOrEqual(
		product.StockMinimo,
	) {
		alertType := "STOCK_BAJO"

		if newAvailableStock.IsZero() {
			alertType = "SIN_STOCK"
		}

		message := fmt.Sprintf(
			"El producto %s tiene stock disponible %s y mínimo %s",
			product.Codigo,
			newAvailableStock.StringFixed(3),
			product.StockMinimo.StringFixed(3),
		)

		_, err := tx.ExecContext(
			ctx,
			`
				MERGE INTO ALERTA_STOCK destino
				USING (
					SELECT
						:1 AS ID_PRODUCTO,
						:2 AS ID_MOVIMIENTO,
						:3 AS TIPO_ALERTA,
						:4 AS STOCK_DETECTADO,
						:5 AS STOCK_MINIMO,
						:6 AS MENSAJE
					FROM DUAL
				) fuente
				ON (
					destino.ID_PRODUCTO =
						fuente.ID_PRODUCTO
					AND destino.ESTADO =
						'PENDIENTE'
				)
				WHEN MATCHED THEN
					UPDATE SET
						destino.ID_MOVIMIENTO =
							fuente.ID_MOVIMIENTO,
						destino.TIPO_ALERTA =
							fuente.TIPO_ALERTA,
						destino.STOCK_DETECTADO =
							fuente.STOCK_DETECTADO,
						destino.STOCK_MINIMO =
							fuente.STOCK_MINIMO,
						destino.MENSAJE =
							fuente.MENSAJE,
						destino.FECHA_GENERACION =
							CAST(
								SYSTIMESTAMP
								AT TIME ZONE '-05:00'
								AS TIMESTAMP
							),
						destino.FECHA_ACTUALIZACION =
							CAST(
								SYSTIMESTAMP
								AT TIME ZONE '-05:00'
								AS TIMESTAMP
							)
				WHEN NOT MATCHED THEN
					INSERT (
						ID_PRODUCTO,
						ID_MOVIMIENTO,
						TIPO_ALERTA,
						STOCK_DETECTADO,
						STOCK_MINIMO,
						ESTADO,
						MENSAJE,
						FECHA_GENERACION,
						FECHA_ACTUALIZACION
					)
					VALUES (
						fuente.ID_PRODUCTO,
						fuente.ID_MOVIMIENTO,
						fuente.TIPO_ALERTA,
						fuente.STOCK_DETECTADO,
						fuente.STOCK_MINIMO,
						'PENDIENTE',
						fuente.MENSAJE,
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
			product.IDProducto,
			movementID,
			alertType,
			newAvailableStock.InexactFloat64(),
			product.StockMinimo.InexactFloat64(),
			message,
		)
		if err != nil {
			return "", fmt.Errorf(
				"no se pudo crear o actualizar la alerta: %w",
				err,
			)
		}

		return "PENDIENTE", nil
	}

	result, err := tx.ExecContext(
		ctx,
		`
			UPDATE ALERTA_STOCK
			SET
				ESTADO = 'ATENDIDA',
				ID_USUARIO_ATENCION = :1,
				FECHA_ATENCION =
					CAST(
						SYSTIMESTAMP
						AT TIME ZONE '-05:00'
						AS TIMESTAMP
					),
				OBSERVACION_ATENCION =
					'Stock normalizado mediante ajuste manual',
				FECHA_ACTUALIZACION =
					CAST(
						SYSTIMESTAMP
						AT TIME ZONE '-05:00'
						AS TIMESTAMP
					)
			WHERE ID_PRODUCTO = :2
			  AND ESTADO = 'PENDIENTE'
		`,
		userID,
		product.IDProducto,
	)
	if err != nil {
		return "", fmt.Errorf(
			"no se pudieron atender las alertas: %w",
			err,
		)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return "", fmt.Errorf(
			"no se pudo verificar la actualización de alertas: %w",
			err,
		)
	}

	if rows > 0 {
		return "ATENDIDA", nil
	}

	return "SIN_CAMBIOS", nil
}

func insertInventoryAdjustmentAudit(
	ctx context.Context,
	tx *sql.Tx,
	product lockedInventoryProduct,
	movementID int64,
	movementType string,
	request models.InventoryAdjustmentRequest,
	newCurrentStock decimal.Decimal,
	newAvailableStock decimal.Decimal,
	userID int64,
	ipAddress string,
) error {
	previousValues, err := json.Marshal(
		map[string]any{
			"codigo_producto": product.Codigo,

			"stock_actual": product.StockActual.StringFixed(3),

			"stock_reservado": product.StockReservado.StringFixed(3),

			"stock_disponible": product.StockActual.
				Sub(
					product.StockReservado,
				).
				StringFixed(3),
		},
	)
	if err != nil {
		return fmt.Errorf(
			"no se pudo construir la auditoría anterior: %w",
			err,
		)
	}

	newValues, err := json.Marshal(
		map[string]any{
			"codigo_producto": product.Codigo,

			"id_movimiento": movementID,

			"tipo_movimiento": movementType,

			"cantidad": request.Cantidad.StringFixed(3),

			"stock_actual": newCurrentStock.StringFixed(3),

			"stock_reservado": product.StockReservado.StringFixed(3),

			"stock_disponible": newAvailableStock.StringFixed(3),

			"motivo": request.Motivo,
		},
	)
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
				'INVENTARIO',
				'UPDATE',
				:2,
				:3,
				:4,
				:5,
				'API REST INVENTARIO',
				CAST(
					SYSTIMESTAMP
					AT TIME ZONE '-05:00'
					AS TIMESTAMP
				)
			)
		`,
		userID,
		fmt.Sprintf(
			"%d",
			product.IDInventario,
		),
		string(previousValues),
		string(newValues),
		nullableText(ipAddress),
	)
	if err != nil {
		return fmt.Errorf(
			"no se pudo auditar el ajuste: %w",
			err,
		)
	}

	return nil
}

func findInventoryMovementDate(
	ctx context.Context,
	tx *sql.Tx,
	movementID int64,
) (string, error) {
	var movementDate string

	err := tx.QueryRowContext(
		ctx,
		`
			SELECT TO_CHAR(
				FECHA_MOVIMIENTO,
				'YYYY-MM-DD"T"HH24:MI:SS'
			)
			FROM MOVIMIENTO_INVENTARIO
			WHERE ID_MOVIMIENTO = :1
		`,
		movementID,
	).Scan(&movementDate)

	if err != nil {
		return "", fmt.Errorf(
			"no se pudo recuperar la fecha del movimiento: %w",
			err,
		)
	}

	return strings.TrimSpace(
		movementDate,
	), nil
}
