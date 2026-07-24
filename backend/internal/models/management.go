package models

// SaleSummary representa una venta dentro del listado general.
type SaleSummary struct {
	IDVenta     int64  `json:"id_venta"`
	NumeroVenta string `json:"numero_venta"`
	Cliente     string `json:"cliente"`
	FechaVenta  string `json:"fecha_venta"`
	Subtotal    string `json:"subtotal"`
	Descuento   string `json:"descuento"`
	Impuesto    string `json:"impuesto"`
	Total       string `json:"total"`
	MetodoPago  string `json:"metodo_pago"`
	Estado      string `json:"estado"`
	TotalItems  int64  `json:"total_items"`
}

// SaleListResponse representa el listado de ventas.
type SaleListResponse struct {
	Status string        `json:"status"`
	Total  int           `json:"total"`
	Ventas []SaleSummary `json:"ventas"`
}

// SaleDetailItem representa un producto de una venta consultada.
type SaleDetailItem struct {
	CodigoProducto string `json:"codigo_producto"`
	NombreProducto string `json:"nombre_producto"`
	Cantidad       string `json:"cantidad"`
	PrecioUnitario string `json:"precio_unitario"`
	Descuento      string `json:"descuento"`
	SubtotalLinea  string `json:"subtotal_linea"`
}

// SaleDetail representa la información completa de una venta.
type SaleDetail struct {
	Status                string           `json:"status"`
	IDVenta               int64            `json:"id_venta"`
	NumeroVenta           string           `json:"numero_venta"`
	Cliente               string           `json:"cliente"`
	IdentificacionCliente string           `json:"identificacion_cliente"`
	FechaVenta            string           `json:"fecha_venta"`
	Subtotal              string           `json:"subtotal"`
	Descuento             string           `json:"descuento"`
	Impuesto              string           `json:"impuesto"`
	Total                 string           `json:"total"`
	MetodoPago            string           `json:"metodo_pago"`
	Estado                string           `json:"estado"`
	Observacion           string           `json:"observacion"`
	Items                 []SaleDetailItem `json:"items"`
}

// StockAlert representa una alerta de inventario.
type StockAlert struct {
	IDAlerta            int64  `json:"id_alerta"`
	CodigoProducto      string `json:"codigo_producto"`
	Producto            string `json:"producto"`
	TipoAlerta          string `json:"tipo_alerta"`
	StockDetectado      string `json:"stock_detectado"`
	StockMinimo         string `json:"stock_minimo"`
	Estado              string `json:"estado"`
	Mensaje             string `json:"mensaje"`
	FechaGeneracion     string `json:"fecha_generacion"`
	FechaAtencion       string `json:"fecha_atencion,omitempty"`
	ObservacionAtencion string `json:"observacion_atencion,omitempty"`
	IDUsuarioAtencion   *int64 `json:"id_usuario_atencion,omitempty"`
}

// StockAlertListResponse representa el listado de alertas.
type StockAlertListResponse struct {
	Status  string       `json:"status"`
	Total   int          `json:"total"`
	Estado  string       `json:"filtro_estado,omitempty"`
	Alertas []StockAlert `json:"alertas"`
}

// DashboardSummary representa los indicadores principales del SIG.
type DashboardSummary struct {
	Status                  string `json:"status"`
	FechaGeneracion         string `json:"fecha_generacion"`
	VentasHoy               int64  `json:"ventas_hoy"`
	IngresosHoy             string `json:"ingresos_hoy"`
	UnidadesVendidasHoy     string `json:"unidades_vendidas_hoy"`
	ProductosReposicion     int64  `json:"productos_reposicion"`
	AlertasPendientes       int64  `json:"alertas_pendientes"`
	ValorInventarioCosto    string `json:"valor_inventario_costo"`
	ValorInventarioVenta    string `json:"valor_inventario_venta"`
	MargenPotencial         string `json:"margen_potencial"`
	CostoReposicionEstimado string `json:"costo_reposicion_estimado"`
}

// UpdateStockAlertRequest representa el cambio de estado de una alerta.
type UpdateStockAlertRequest struct {
	Estado      string `json:"estado"`
	Observacion string `json:"observacion"`
	IDUsuario   *int64 `json:"id_usuario,omitempty"`
}

// StockAlertUpdateResult representa una alerta después de ser procesada.
type StockAlertUpdateResult struct {
	Status              string `json:"status"`
	IDAlerta            int64  `json:"id_alerta"`
	CodigoProducto      string `json:"codigo_producto"`
	Producto            string `json:"producto"`
	TipoAlerta          string `json:"tipo_alerta"`
	Estado              string `json:"estado"`
	StockDetectado      string `json:"stock_detectado"`
	StockMinimo         string `json:"stock_minimo"`
	Mensaje             string `json:"mensaje,omitempty"`
	ObservacionAtencion string `json:"observacion_atencion"`
	FechaGeneracion     string `json:"fecha_generacion"`
	FechaAtencion       string `json:"fecha_atencion"`
	IDUsuarioAtencion   *int64 `json:"id_usuario_atencion,omitempty"`
}
