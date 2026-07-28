package models

// UpdateUserStateRequest cambia el estado lógico
// de una cuenta administrativa.
type UpdateUserStateRequest struct {
	Estado string `json:"estado"`
}

// ResetUserPasswordRequest contiene la nueva
// contraseña establecida por un administrador.
type ResetUserPasswordRequest struct {
	Contrasena string `json:"contrasena"`
}
