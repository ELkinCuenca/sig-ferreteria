package repository

import (
	"context"
	"database/sql"
	"fmt"

	"sigefer.local/backend/internal/models"
)

// ProductRepository administra el acceso a productos en Oracle.
type ProductRepository struct {
	db *sql.DB
}

// NewProductRepository crea el repositorio de productos.
func NewProductRepository(db *sql.DB) *ProductRepository {
	return &ProductRepository{
		db: db,
	}
}

// List devuelve los productos activos con su inventario.
// Cuando lowStockOnly es true, devuelve únicamente productos
// cuyo stock disponible es menor o igual al stock mínimo.
func (repository *ProductRepository) List(
	ctx context.Context,
	lowStockOnly bool,
) ([]models.Product, error) {
	query := `
		SELECT
			p.ID_PRODUCTO,
			p.CODIGO,
			p.NOMBRE,
			c.NOMBRE AS CATEGORIA,
			p.UNIDAD_MEDIDA,
			p.PRECIO_COMPRA,
			p.PRECIO_VENTA,
			ROUND(
				p.PRECIO_VENTA - p.PRECIO_COMPRA,
				2
			) AS MARGEN_UNITARIO,
			i.STOCK_ACTUAL,
			i.STOCK_RESERVADO,
			i.STOCK_DISPONIBLE,
			p.STOCK_MINIMO,
			CASE
				WHEN i.STOCK_DISPONIBLE <= p.STOCK_MINIMO
					THEN 'STOCK BAJO'
				ELSE 'NORMAL'
			END AS ESTADO_STOCK
		FROM PRODUCTO p
		INNER JOIN CATEGORIA c
			ON c.ID_CATEGORIA = p.ID_CATEGORIA
		INNER JOIN INVENTARIO i
			ON i.ID_PRODUCTO = p.ID_PRODUCTO
		WHERE p.ESTADO = 'A'
		  AND c.ESTADO = 'A'
	`

	if lowStockOnly {
		query += `
		  AND i.STOCK_DISPONIBLE <= p.STOCK_MINIMO
		`
	}

	query += `
		ORDER BY p.NOMBRE
	`

	rows, err := repository.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf(
			"no se pudo consultar productos: %w",
			err,
		)
	}
	defer rows.Close()

	products := make([]models.Product, 0)

	for rows.Next() {
		var product models.Product

		err := rows.Scan(
			&product.IDProducto,
			&product.Codigo,
			&product.Nombre,
			&product.Categoria,
			&product.UnidadMedida,
			&product.PrecioCompra,
			&product.PrecioVenta,
			&product.MargenUnitario,
			&product.StockActual,
			&product.StockReservado,
			&product.StockDisponible,
			&product.StockMinimo,
			&product.EstadoStock,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"no se pudo interpretar un producto: %w",
				err,
			)
		}

		products = append(products, product)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"error recorriendo productos: %w",
			err,
		)
	}

	return products, nil
}
