package models

// LoginRequest representa las credenciales recibidas.
type LoginRequest struct {
	Usuario    string `json:"usuario"`
	Contrasena string `json:"contrasena"`
}

// AuthUser representa la identidad pública de un usuario.
type AuthUser struct {
	IDUsuario     int64  `json:"id_usuario"`
	NombreUsuario string `json:"nombre_usuario"`
	Nombres       string `json:"nombres"`
	Apellidos     string `json:"apellidos"`
	Correo        string `json:"correo"`
	Rol           string `json:"rol"`
}

// AuthPrincipal representa una sesión autenticada.
type AuthPrincipal struct {
	IDSesion int64 `json:"-"`
	AuthUser
}

// LoginResult representa el resultado interno del acceso.
type LoginResult struct {
	Token            string
	ExpiresInSeconds int64
	User             AuthUser
}
