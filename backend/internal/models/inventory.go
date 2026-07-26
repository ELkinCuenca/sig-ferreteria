package models

import "github.com/shopspring/decimal"

// InventoryAdjustmentRequest representa un ajuste manual
// positivo o negativo de existencias.
type InventoryAdjustmentRequest struct {
	CodigoProducto string          `json:"codigo_producto"`
	TipoAjuste     string          `json:"tipo_ajuste"`
	Cantidad       decimal.Decimal `json:"cantidad"`
	Motivo         string          `json:"motivo"`
}

// InventoryAdjustmentResult representa el resultado
// transaccional del ajuste.
type InventoryAdjustmentResult struct {
	Status       string `json:"status"`
	IDMovimiento int64  `json:"id_movimiento"`

	CodigoProducto string `json:"codigo_producto"`
	Producto       string `json:"producto"`
	TipoMovimiento string `json:"tipo_movimiento"`

	Cantidad        string `json:"cantidad"`
	StockAnterior   string `json:"stock_anterior"`
	StockNuevo      string `json:"stock_nuevo"`
	StockReservado  string `json:"stock_reservado"`
	StockDisponible string `json:"stock_disponible"`
	StockMinimo     string `json:"stock_minimo"`
	EstadoStock     string `json:"estado_stock"`
	ResultadoAlerta string `json:"resultado_alerta"`
	Motivo          string `json:"motivo"`
	FechaMovimiento string `json:"fecha_movimiento"`
}

// InventoryMovement representa un movimiento histórico.
type InventoryMovement struct {
	IDMovimiento int64 `json:"id_movimiento"`
	IDProducto   int64 `json:"id_producto"`

	CodigoProducto string `json:"codigo_producto"`
	Producto       string `json:"producto"`
	TipoMovimiento string `json:"tipo_movimiento"`

	Cantidad      string `json:"cantidad"`
	StockAnterior string `json:"stock_anterior"`
	StockNuevo    string `json:"stock_nuevo"`

	Motivo string `json:"motivo,omitempty"`

	IDUsuario *int64 `json:"id_usuario,omitempty"`
	Usuario   string `json:"usuario"`

	IDVenta               *int64 `json:"id_venta,omitempty"`
	IDSolicitudReposicion *int64 `json:"id_solicitud_reposicion,omitempty"`
	Referencia            string `json:"referencia,omitempty"`
	FechaMovimiento       string `json:"fecha_movimiento"`
}
