package models

import "github.com/shopspring/decimal"

// Category representa una categoría activa del catálogo.
type Category struct {
	IDCategoria int64  `json:"id_categoria"`
	Nombre      string `json:"nombre"`
	Descripcion string `json:"descripcion,omitempty"`
	Estado      string `json:"estado"`
}

// CreateProductRequest representa la creación de un producto
// junto con su inventario inicial.
type CreateProductRequest struct {
	IDCategoria int64 `json:"id_categoria"`

	Codigo       string `json:"codigo"`
	Nombre       string `json:"nombre"`
	Descripcion  string `json:"descripcion,omitempty"`
	UnidadMedida string `json:"unidad_medida"`
	Ubicacion    string `json:"ubicacion,omitempty"`

	PrecioCompra decimal.Decimal `json:"precio_compra"`
	PrecioVenta  decimal.Decimal `json:"precio_venta"`
	StockMinimo  decimal.Decimal `json:"stock_minimo"`
	StockInicial decimal.Decimal `json:"stock_inicial"`
}

// ProductDetail representa la información completa
// comercial y de inventario de un producto.
type ProductDetail struct {
	IDProducto  int64 `json:"id_producto"`
	IDCategoria int64 `json:"id_categoria"`

	Codigo      string `json:"codigo"`
	Nombre      string `json:"nombre"`
	Descripcion string `json:"descripcion,omitempty"`

	Categoria    string `json:"categoria"`
	UnidadMedida string `json:"unidad_medida"`
	Estado       string `json:"estado"`
	Ubicacion    string `json:"ubicacion,omitempty"`

	PrecioCompra   string `json:"precio_compra"`
	PrecioVenta    string `json:"precio_venta"`
	MargenUnitario string `json:"margen_unitario"`

	StockActual     string `json:"stock_actual"`
	StockReservado  string `json:"stock_reservado"`
	StockDisponible string `json:"stock_disponible"`
	StockMinimo     string `json:"stock_minimo"`

	EstadoStock string `json:"estado_stock"`

	FechaCreacion      string `json:"fecha_creacion"`
	FechaActualizacion string `json:"fecha_actualizacion"`
}
