package models

// UpdateUserRequest contiene los datos administrativos
// editables de una cuenta.
type UpdateUserRequest struct {
	IDRol int64 `json:"id_rol"`

	Nombres   string `json:"nombres"`
	Apellidos string `json:"apellidos"`
	Correo    string `json:"correo"`
}
