package models

// Client representa un cliente activo disponible para ventas.
type Client struct {
	IDCliente          int64  `json:"id_cliente"`
	TipoIdentificacion string `json:"tipo_identificacion"`
	Identificacion     string `json:"identificacion"`
	NombreCompleto     string `json:"nombre_completo"`
	Telefono           string `json:"telefono,omitempty"`
	Correo             string `json:"correo,omitempty"`
	Direccion          string `json:"direccion,omitempty"`
	Estado             string `json:"estado"`
}
