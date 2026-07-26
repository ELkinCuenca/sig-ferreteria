package models

import "github.com/shopspring/decimal"

// UpdateProductRequest representa los campos editables
// de un producto. El código y el stock no se modifican.
type UpdateProductRequest struct {
	IDCategoria int64 `json:"id_categoria"`

	Nombre       string `json:"nombre"`
	Descripcion  string `json:"descripcion,omitempty"`
	UnidadMedida string `json:"unidad_medida"`
	Ubicacion    string `json:"ubicacion,omitempty"`

	PrecioCompra decimal.Decimal `json:"precio_compra"`
	PrecioVenta  decimal.Decimal `json:"precio_venta"`
	StockMinimo  decimal.Decimal `json:"stock_minimo"`
}

// UpdateProductStateRequest representa la activación
// o desactivación lógica de un producto.
type UpdateProductStateRequest struct {
	Estado string `json:"estado"`
}
