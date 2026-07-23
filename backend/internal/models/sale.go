package models

import "github.com/shopspring/decimal"

// SaleItemRequest representa un producto solicitado en una venta.
type SaleItemRequest struct {
	CodigoProducto string          `json:"codigo_producto"`
	Cantidad       decimal.Decimal `json:"cantidad"`
	Descuento      decimal.Decimal `json:"descuento"`
}

// SaleCreateRequest representa la solicitud de creación de una venta.
type SaleCreateRequest struct {
	IdentificacionCliente string            `json:"identificacion_cliente"`
	IDUsuario             *int64            `json:"id_usuario,omitempty"`
	MetodoPago            string            `json:"metodo_pago"`
	DescuentoGeneral      decimal.Decimal   `json:"descuento_general"`
	Observacion           string            `json:"observacion,omitempty"`
	Items                 []SaleItemRequest `json:"items"`
}

// SaleItemResult representa una línea confirmada de la venta.
type SaleItemResult struct {
	CodigoProducto  string `json:"codigo_producto"`
	NombreProducto  string `json:"nombre_producto"`
	Cantidad        string `json:"cantidad"`
	PrecioUnitario  string `json:"precio_unitario"`
	Descuento       string `json:"descuento"`
	SubtotalLinea   string `json:"subtotal_linea"`
	StockAnterior   string `json:"stock_anterior"`
	StockNuevo      string `json:"stock_nuevo"`
	StockDisponible string `json:"stock_disponible"`
}

// StockAlertResult representa una alerta generada por la venta.
type StockAlertResult struct {
	CodigoProducto string `json:"codigo_producto"`
	TipoAlerta     string `json:"tipo_alerta"`
	StockDetectado string `json:"stock_detectado"`
	StockMinimo    string `json:"stock_minimo"`
}

// SaleCreateResponse representa una venta confirmada.
type SaleCreateResponse struct {
	Status       string             `json:"status"`
	IDVenta      int64              `json:"id_venta"`
	NumeroVenta  string             `json:"numero_venta"`
	Subtotal     string             `json:"subtotal"`
	Descuento    string             `json:"descuento"`
	Impuesto     string             `json:"impuesto"`
	Total        string             `json:"total"`
	MetodoPago   string             `json:"metodo_pago"`
	Estado       string             `json:"estado"`
	Items        []SaleItemResult   `json:"items"`
	AlertasStock []StockAlertResult `json:"alertas_stock"`
}
