package models

// Product representa un producto con su información comercial
// y el estado actual de inventario.
type Product struct {
	IDProducto      int64   `json:"id_producto"`
	Codigo          string  `json:"codigo"`
	Nombre          string  `json:"nombre"`
	Categoria       string  `json:"categoria"`
	UnidadMedida    string  `json:"unidad_medida"`
	PrecioCompra    float64 `json:"precio_compra"`
	PrecioVenta     float64 `json:"precio_venta"`
	MargenUnitario  float64 `json:"margen_unitario"`
	StockActual     float64 `json:"stock_actual"`
	StockReservado  float64 `json:"stock_reservado"`
	StockDisponible float64 `json:"stock_disponible"`
	StockMinimo     float64 `json:"stock_minimo"`
	EstadoStock     string  `json:"estado_stock"`
}
