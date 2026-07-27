package models

// RoleSummary representa un rol asignable.
type RoleSummary struct {
	IDRol       int64  `json:"id_rol"`
	Nombre      string `json:"nombre"`
	Descripcion string `json:"descripcion,omitempty"`
	Estado      string `json:"estado"`
}

// UserAdmin representa un usuario sin información sensible.
type UserAdmin struct {
	IDUsuario int64 `json:"id_usuario"`
	IDRol     int64 `json:"id_rol"`

	NombreUsuario string `json:"nombre_usuario"`
	Nombres       string `json:"nombres"`
	Apellidos     string `json:"apellidos"`
	Correo        string `json:"correo"`

	Rol    string `json:"rol"`
	Estado string `json:"estado"`

	IntentosFallidos int `json:"intentos_fallidos"`

	UltimoAcceso         string `json:"ultimo_acceso,omitempty"`
	UltimoIntentoFallido string `json:"ultimo_intento_fallido,omitempty"`
	FechaBloqueo         string `json:"fecha_bloqueo,omitempty"`
	FechaCreacion        string `json:"fecha_creacion"`
	FechaActualizacion   string `json:"fecha_actualizacion"`
}

// CreateUserRequest contiene los datos de una nueva cuenta.
type CreateUserRequest struct {
	IDRol int64 `json:"id_rol"`

	NombreUsuario string `json:"nombre_usuario"`
	Nombres       string `json:"nombres"`
	Apellidos     string `json:"apellidos"`
	Correo        string `json:"correo"`
	Contrasena    string `json:"contrasena"`
}
