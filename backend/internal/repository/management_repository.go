package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"sigefer.local/backend/internal/models"
)

// ErrSaleNotFound indica que una venta no existe.
var ErrSaleNotFound = errors.New("venta no encontrada")

// ManagementRepository administra consultas gerenciales.
type ManagementRepository struct {
	db *sql.DB
}

// NewManagementRepository crea el repositorio gerencial.
func NewManagementRepository(
	db *sql.DB,
) *ManagementRepository {
	return &ManagementRepository{
		db: db,
	}
}

// ListSales devuelve las ventas más recientes.
func (repository *ManagementRepository) ListSales(
	ctx context.Context,
	limit int,
) ([]models.SaleSummary, error) {
	const query = `
		SELECT
			ID_VENTA,
			NUMERO_VENTA,
			CLIENTE,
			FECHA_VENTA,
			SUBTOTAL,
			DESCUENTO,
			IMPUESTO,
			TOTAL,
			METODO_PAGO,
			ESTADO,
			TOTAL_ITEMS
		FROM (
			SELECT
				v.ID_VENTA,
				v.NUMERO_VENTA,

				CASE
					WHEN c.RAZON_SOCIAL IS NOT NULL
						THEN c.RAZON_SOCIAL
					ELSE TRIM(
						c.NOMBRES || ' ' || c.APELLIDOS
					)
				END AS CLIENTE,

				TO_CHAR(
					v.FECHA_VENTA,
					'YYYY-MM-DD"T"HH24:MI:SS'
				) AS FECHA_VENTA,

				v.SUBTOTAL,
				v.DESCUENTO,
				v.IMPUESTO,
				v.TOTAL,
				v.METODO_PAGO,
				v.ESTADO,
				COUNT(d.ID_DETALLE_VENTA) AS TOTAL_ITEMS

			FROM VENTA v

			INNER JOIN CLIENTE c
				ON c.ID_CLIENTE = v.ID_CLIENTE

			LEFT JOIN DETALLE_VENTA d
				ON d.ID_VENTA = v.ID_VENTA

			GROUP BY
				v.ID_VENTA,
				v.NUMERO_VENTA,
				c.RAZON_SOCIAL,
				c.NOMBRES,
				c.APELLIDOS,
				v.FECHA_VENTA,
				v.SUBTOTAL,
				v.DESCUENTO,
				v.IMPUESTO,
				v.TOTAL,
				v.METODO_PAGO,
				v.ESTADO

			ORDER BY v.ID_VENTA DESC
		)
		WHERE ROWNUM <= :1
	`

	rows, err := repository.db.QueryContext(
		ctx,
		query,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"no se pudo consultar las ventas: %w",
			err,
		)
	}
	defer rows.Close()

	sales := make([]models.SaleSummary, 0)

	for rows.Next() {
		var (
			sale     models.SaleSummary
			subtotal float64
			discount float64
			tax      float64
			total    float64
		)

		err := rows.Scan(
			&sale.IDVenta,
			&sale.NumeroVenta,
			&sale.Cliente,
			&sale.FechaVenta,
			&subtotal,
			&discount,
			&tax,
			&total,
			&sale.MetodoPago,
			&sale.Estado,
			&sale.TotalItems,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"no se pudo interpretar una venta: %w",
				err,
			)
		}

		sale.Subtotal = formatMoney(subtotal)
		sale.Descuento = formatMoney(discount)
		sale.Impuesto = formatMoney(tax)
		sale.Total = formatMoney(total)

		sales = append(sales, sale)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"error recorriendo las ventas: %w",
			err,
		)
	}

	return sales, nil
}

// FindSaleByNumber devuelve una venta con sus detalles.
func (repository *ManagementRepository) FindSaleByNumber(
	ctx context.Context,
	saleNumber string,
) (models.SaleDetail, error) {
	const headerQuery = `
		SELECT
			v.ID_VENTA,
			v.NUMERO_VENTA,

			CASE
				WHEN c.RAZON_SOCIAL IS NOT NULL
					THEN c.RAZON_SOCIAL
				ELSE TRIM(
					c.NOMBRES || ' ' || c.APELLIDOS
				)
			END AS CLIENTE,

			c.IDENTIFICACION,

			TO_CHAR(
				v.FECHA_VENTA,
				'YYYY-MM-DD"T"HH24:MI:SS'
			),

			v.SUBTOTAL,
			v.DESCUENTO,
			v.IMPUESTO,
			v.TOTAL,
			v.METODO_PAGO,
			v.ESTADO,
			NVL(v.OBSERVACION, '')

		FROM VENTA v

		INNER JOIN CLIENTE c
			ON c.ID_CLIENTE = v.ID_CLIENTE

		WHERE v.NUMERO_VENTA = :1
	`

	var (
		sale     models.SaleDetail
		subtotal float64
		discount float64
		tax      float64
		total    float64
	)

	err := repository.db.QueryRowContext(
		ctx,
		headerQuery,
		saleNumber,
	).Scan(
		&sale.IDVenta,
		&sale.NumeroVenta,
		&sale.Cliente,
		&sale.IdentificacionCliente,
		&sale.FechaVenta,
		&subtotal,
		&discount,
		&tax,
		&total,
		&sale.MetodoPago,
		&sale.Estado,
		&sale.Observacion,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return models.SaleDetail{}, ErrSaleNotFound
	}

	if err != nil {
		return models.SaleDetail{}, fmt.Errorf(
			"no se pudo consultar la venta: %w",
			err,
		)
	}

	sale.Status = "ok"
	sale.Subtotal = formatMoney(subtotal)
	sale.Descuento = formatMoney(discount)
	sale.Impuesto = formatMoney(tax)
	sale.Total = formatMoney(total)

	const detailQuery = `
		SELECT
			p.CODIGO,
			p.NOMBRE,
			d.CANTIDAD,
			d.PRECIO_UNITARIO,
			d.DESCUENTO,
			d.SUBTOTAL_LINEA

		FROM DETALLE_VENTA d

		INNER JOIN PRODUCTO p
			ON p.ID_PRODUCTO = d.ID_PRODUCTO

		WHERE d.ID_VENTA = :1

		ORDER BY d.ID_DETALLE_VENTA
	`

	rows, err := repository.db.QueryContext(
		ctx,
		detailQuery,
		sale.IDVenta,
	)
	if err != nil {
		return models.SaleDetail{}, fmt.Errorf(
			"no se pudo consultar el detalle: %w",
			err,
		)
	}
	defer rows.Close()

	sale.Items = make([]models.SaleDetailItem, 0)

	for rows.Next() {
		var (
			item      models.SaleDetailItem
			quantity  float64
			price     float64
			discount  float64
			lineTotal float64
		)

		err := rows.Scan(
			&item.CodigoProducto,
			&item.NombreProducto,
			&quantity,
			&price,
			&discount,
			&lineTotal,
		)
		if err != nil {
			return models.SaleDetail{}, fmt.Errorf(
				"no se pudo interpretar el detalle: %w",
				err,
			)
		}

		item.Cantidad = formatQuantity(quantity)
		item.PrecioUnitario = formatMoney(price)
		item.Descuento = formatMoney(discount)
		item.SubtotalLinea = formatMoney(lineTotal)

		sale.Items = append(sale.Items, item)
	}

	if err := rows.Err(); err != nil {
		return models.SaleDetail{}, fmt.Errorf(
			"error recorriendo el detalle: %w",
			err,
		)
	}

	return sale, nil
}

// ListStockAlerts devuelve las alertas de inventario.
func (repository *ManagementRepository) ListStockAlerts(
	ctx context.Context,
	status string,
) ([]models.StockAlert, error) {
	query := `
		SELECT
			a.ID_ALERTA,
			p.CODIGO,
			p.NOMBRE,
			a.TIPO_ALERTA,
			a.STOCK_DETECTADO,
			a.STOCK_MINIMO,
			a.ESTADO,
			a.MENSAJE,

			TO_CHAR(
				a.FECHA_GENERACION,
				'YYYY-MM-DD"T"HH24:MI:SS'
			),

			TO_CHAR(
				a.FECHA_ATENCION,
				'YYYY-MM-DD"T"HH24:MI:SS'
			)

		FROM ALERTA_STOCK a

		INNER JOIN PRODUCTO p
			ON p.ID_PRODUCTO = a.ID_PRODUCTO
	`

	args := make([]any, 0, 1)

	if status != "" {
		query += `
			WHERE a.ESTADO = :1
		`

		args = append(args, status)
	}

	query += `
		ORDER BY
			CASE
				WHEN a.ESTADO = 'PENDIENTE' THEN 1
				WHEN a.ESTADO = 'ATENDIDA' THEN 2
				ELSE 3
			END,
			a.FECHA_GENERACION DESC
	`

	rows, err := repository.db.QueryContext(
		ctx,
		query,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"no se pudo consultar alertas: %w",
			err,
		)
	}
	defer rows.Close()

	alerts := make([]models.StockAlert, 0)

	for rows.Next() {
		var (
			alert         models.StockAlert
			detectedStock float64
			minimumStock  float64
			message       sql.NullString
			attendedAt    sql.NullString
		)

		err := rows.Scan(
			&alert.IDAlerta,
			&alert.CodigoProducto,
			&alert.Producto,
			&alert.TipoAlerta,
			&detectedStock,
			&minimumStock,
			&alert.Estado,
			&message,
			&alert.FechaGeneracion,
			&attendedAt,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"no se pudo interpretar una alerta: %w",
				err,
			)
		}

		alert.StockDetectado =
			formatQuantity(detectedStock)

		alert.StockMinimo =
			formatQuantity(minimumStock)

		if message.Valid {
			alert.Mensaje = message.String
		}

		if attendedAt.Valid {
			alert.FechaAtencion = attendedAt.String
		}

		alerts = append(alerts, alert)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"error recorriendo las alertas: %w",
			err,
		)
	}

	return alerts, nil
}

// Dashboard devuelve los indicadores principales del negocio.
func (repository *ManagementRepository) Dashboard(
	ctx context.Context,
) (models.DashboardSummary, error) {
	const query = `
		SELECT
			TO_CHAR(
				SYSTIMESTAMP AT TIME ZONE '-05:00',
				'YYYY-MM-DD"T"HH24:MI:SS TZH:TZM'
			),

			(
				SELECT COUNT(*)
				FROM VENTA
				WHERE ESTADO = 'COMPLETADA'
				  AND CAST(FECHA_VENTA AS DATE) - (5 / 24)
				      >= TRUNC(SYSDATE - (5 / 24))
				  AND CAST(FECHA_VENTA AS DATE) - (5 / 24)
				      < TRUNC(SYSDATE - (5 / 24)) + 1
			),

			NVL(
				(
					SELECT SUM(TOTAL)
					FROM VENTA
					WHERE ESTADO = 'COMPLETADA'
					  AND CAST(FECHA_VENTA AS DATE) - (5 / 24)
					      >= TRUNC(SYSDATE - (5 / 24))
					  AND CAST(FECHA_VENTA AS DATE) - (5 / 24)
					      < TRUNC(SYSDATE - (5 / 24)) + 1
				),
				0
			),

			NVL(
				(
					SELECT SUM(d.CANTIDAD)
					FROM DETALLE_VENTA d
					INNER JOIN VENTA v
						ON v.ID_VENTA = d.ID_VENTA
					WHERE v.ESTADO = 'COMPLETADA'
					  AND CAST(v.FECHA_VENTA AS DATE) - (5 / 24)
					      >= TRUNC(SYSDATE - (5 / 24))
					  AND CAST(v.FECHA_VENTA AS DATE) - (5 / 24)
					      < TRUNC(SYSDATE - (5 / 24)) + 1
				),
				0
			),

			(
				SELECT COUNT(*)
				FROM VW_PRODUCTOS_REPOSICION
			),

			(
				SELECT COUNT(*)
				FROM ALERTA_STOCK
				WHERE ESTADO = 'PENDIENTE'
			),

			NVL(
				(
					SELECT SUM(
						i.STOCK_ACTUAL * p.PRECIO_COMPRA
					)
					FROM INVENTARIO i
					INNER JOIN PRODUCTO p
						ON p.ID_PRODUCTO = i.ID_PRODUCTO
					WHERE p.ESTADO = 'A'
				),
				0
			),

			NVL(
				(
					SELECT SUM(
						i.STOCK_ACTUAL * p.PRECIO_VENTA
					)
					FROM INVENTARIO i
					INNER JOIN PRODUCTO p
						ON p.ID_PRODUCTO = i.ID_PRODUCTO
					WHERE p.ESTADO = 'A'
				),
				0
			),

			NVL(
				(
					SELECT SUM(
						i.STOCK_ACTUAL *
						(
							p.PRECIO_VENTA -
							p.PRECIO_COMPRA
						)
					)
					FROM INVENTARIO i
					INNER JOIN PRODUCTO p
						ON p.ID_PRODUCTO = i.ID_PRODUCTO
					WHERE p.ESTADO = 'A'
				),
				0
			),

			NVL(
				(
					SELECT SUM(
						COSTO_REPOSICION_ESTIMADO
					)
					FROM VW_PRODUCTOS_REPOSICION
				),
				0
			)

		FROM DUAL
	`

	var (
		dashboard          models.DashboardSummary
		incomeToday        float64
		unitsToday         float64
		inventoryCost      float64
		inventorySaleValue float64
		potentialMargin    float64
		replenishmentCost  float64
	)

	err := repository.db.QueryRowContext(
		ctx,
		query,
	).Scan(
		&dashboard.FechaGeneracion,
		&dashboard.VentasHoy,
		&incomeToday,
		&unitsToday,
		&dashboard.ProductosReposicion,
		&dashboard.AlertasPendientes,
		&inventoryCost,
		&inventorySaleValue,
		&potentialMargin,
		&replenishmentCost,
	)
	if err != nil {
		return models.DashboardSummary{}, fmt.Errorf(
			"no se pudo generar el dashboard: %w",
			err,
		)
	}

	dashboard.Status = "ok"
	dashboard.IngresosHoy = formatMoney(incomeToday)
	dashboard.UnidadesVendidasHoy =
		formatQuantity(unitsToday)
	dashboard.ValorInventarioCosto =
		formatMoney(inventoryCost)
	dashboard.ValorInventarioVenta =
		formatMoney(inventorySaleValue)
	dashboard.MargenPotencial =
		formatMoney(potentialMargin)
	dashboard.CostoReposicionEstimado =
		formatMoney(replenishmentCost)

	return dashboard, nil
}

func formatMoney(value float64) string {
	return fmt.Sprintf("%.2f", value)
}

func formatQuantity(value float64) string {
	return fmt.Sprintf("%.3f", value)
}
