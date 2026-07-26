package models

import "github.com/shopspring/decimal"

// BPMProvider representa un proveedor disponible
// para una solicitud de reposición.
type BPMProvider struct {
	IDProveedor    int64  `json:"id_proveedor"`
	RUC            string `json:"ruc"`
	RazonSocial    string `json:"razon_social"`
	NombreContacto string `json:"nombre_contacto,omitempty"`
	Telefono       string `json:"telefono,omitempty"`
	Correo         string `json:"correo,omitempty"`
}

// CreateReplenishmentRequest representa la creación
// de un borrador de reposición.
type CreateReplenishmentRequest struct {
	CodigoProducto        string          `json:"codigo_producto"`
	IDProveedor           int64           `json:"id_proveedor"`
	IDAlerta              *int64          `json:"id_alerta,omitempty"`
	CantidadSolicitada    decimal.Decimal `json:"cantidad_solicitada"`
	CostoUnitarioEstimado decimal.Decimal `json:"costo_unitario_estimado"`
	Observacion           string          `json:"observacion,omitempty"`
}

// ReplenishmentTransitionRequest contiene información
// adicional para una transición BPM.
type ReplenishmentTransitionRequest struct {
	Observacion string `json:"observacion,omitempty"`
}

// Replenishment representa una instancia consolidada
// del proceso BPM de reposición.
type Replenishment struct {
	IDSolicitud     int64  `json:"id_solicitud"`
	NumeroSolicitud string `json:"numero_solicitud"`
	CodigoProducto  string `json:"codigo_producto"`
	Producto        string `json:"producto"`
	UnidadMedida    string `json:"unidad_medida"`
	IDProveedor     int64  `json:"id_proveedor"`
	RUCProveedor    string `json:"ruc_proveedor"`
	Proveedor       string `json:"proveedor"`
	IDAlerta        *int64 `json:"id_alerta,omitempty"`
	TipoAlerta      string `json:"tipo_alerta,omitempty"`
	EstadoAlerta    string `json:"estado_alerta,omitempty"`

	CantidadSolicitada    string `json:"cantidad_solicitada"`
	CantidadRecibida      string `json:"cantidad_recibida"`
	CostoUnitarioEstimado string `json:"costo_unitario_estimado"`
	CostoTotalEstimado    string `json:"costo_total_estimado"`

	Estado          string `json:"estado"`
	StockActual     string `json:"stock_actual"`
	StockReservado  string `json:"stock_reservado"`
	StockDisponible string `json:"stock_disponible"`
	StockMinimo     string `json:"stock_minimo"`

	IDUsuarioSolicitante int64  `json:"id_usuario_solicitante"`
	UsuarioSolicitante   string `json:"usuario_solicitante"`

	IDUsuarioAprobador *int64 `json:"id_usuario_aprobador,omitempty"`
	UsuarioAprobador   string `json:"usuario_aprobador,omitempty"`

	IDUsuarioReceptor *int64 `json:"id_usuario_receptor,omitempty"`
	UsuarioReceptor   string `json:"usuario_receptor,omitempty"`

	FechaSolicitud  string `json:"fecha_solicitud,omitempty"`
	FechaAprobacion string `json:"fecha_aprobacion,omitempty"`
	FechaRechazo    string `json:"fecha_rechazo,omitempty"`
	FechaPedido     string `json:"fecha_pedido,omitempty"`
	FechaRecepcion  string `json:"fecha_recepcion,omitempty"`
	FechaCierre     string `json:"fecha_cierre,omitempty"`

	Observacion   string `json:"observacion,omitempty"`
	MotivoRechazo string `json:"motivo_rechazo,omitempty"`

	FechaCreacion      string `json:"fecha_creacion"`
	FechaActualizacion string `json:"fecha_actualizacion"`
}

// ReplenishmentHistory representa una transición
// almacenada en el historial BPM.
type ReplenishmentHistory struct {
	IDHistorial    int64  `json:"id_historial"`
	IDUsuario      int64  `json:"id_usuario"`
	Usuario        string `json:"usuario"`
	EstadoAnterior string `json:"estado_anterior,omitempty"`
	EstadoNuevo    string `json:"estado_nuevo"`
	Accion         string `json:"accion"`
	Observacion    string `json:"observacion,omitempty"`
	FechaEvento    string `json:"fecha_evento"`
}

// ReplenishmentDetail agrega el historial al proceso.
type ReplenishmentDetail struct {
	Replenishment
	Historial []ReplenishmentHistory `json:"historial"`
}

// RejectReplenishmentRequest representa el rechazo
// de una solicitud de reposición.
type RejectReplenishmentRequest struct {
	MotivoRechazo string `json:"motivo_rechazo"`
}

// ReceiveReplenishmentRequest representa la entrada
// física de productos al inventario.
type ReceiveReplenishmentRequest struct {
	CantidadRecibida decimal.Decimal `json:"cantidad_recibida"`
	Observacion      string          `json:"observacion,omitempty"`
}
