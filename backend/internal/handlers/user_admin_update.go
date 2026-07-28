package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"sigefer.local/backend/internal/middleware"
	"sigefer.local/backend/internal/models"
	"sigefer.local/backend/internal/repository"
)

// Update procesa PATCH /api/v1/usuarios/{id}.
func (
	handler *UserAdminHandler,
) Update(
	writer http.ResponseWriter,
	request *http.Request,
) {
	principal, valid :=
		middleware.PrincipalFromContext(
			request.Context(),
		)

	if !valid {
		writeUserAdminError(
			writer,
			http.StatusUnauthorized,
			"sesión autenticada no disponible",
		)
		return
	}

	userID, valid :=
		parseUserAdminPathID(
			writer,
			request,
		)

	if !valid {
		return
	}

	var payload models.UpdateUserRequest

	if err := decodeUserAdminJSON(
		writer,
		request,
		&payload,
	); err != nil {
		writeUserAdminError(
			writer,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	normalizeUpdateUser(
		&payload,
	)

	if message :=
		validateUpdateUser(payload); message != "" {
		writeUserAdminError(
			writer,
			http.StatusBadRequest,
			message,
		)
		return
	}

	ctx, cancel := context.WithTimeout(
		request.Context(),
		20*time.Second,
	)
	defer cancel()

	user, err :=
		handler.repository.UpdateUser(
			ctx,
			userID,
			payload,
			principal.IDUsuario,
			clientIP(request),
		)

	switch {
	case errors.Is(
		err,
		repository.ErrUserAdminNotFound,
	):
		writeUserAdminError(
			writer,
			http.StatusNotFound,
			"usuario no encontrado",
		)

	case errors.Is(
		err,
		repository.ErrUserAdminRoleNotFound,
	):
		writeUserAdminError(
			writer,
			http.StatusBadRequest,
			"el rol no existe o está inactivo",
		)

	case errors.Is(
		err,
		repository.ErrUserAdminEmailDuplicate,
	):
		writeUserAdminError(
			writer,
			http.StatusConflict,
			"el correo ya se encuentra registrado",
		)

	case errors.Is(
		err,
		repository.ErrUserAdminLastAdministrator,
	):
		writeUserAdminError(
			writer,
			http.StatusConflict,
			"no se puede cambiar el rol del último administrador activo",
		)

	case err != nil:
		log.Printf(
			"error actualizando usuario %d: %v",
			userID,
			err,
		)

		writeUserAdminError(
			writer,
			http.StatusInternalServerError,
			"no fue posible actualizar el usuario",
		)

	default:
		writeJSON(writer, user)
	}
}

func normalizeUpdateUser(
	request *models.UpdateUserRequest,
) {
	request.Nombres =
		strings.TrimSpace(
			request.Nombres,
		)

	request.Apellidos =
		strings.TrimSpace(
			request.Apellidos,
		)

	request.Correo =
		strings.ToLower(
			strings.TrimSpace(
				request.Correo,
			),
		)
}

func validateUpdateUser(
	request models.UpdateUserRequest,
) string {
	if request.IDRol <= 0 {
		return "id_rol debe ser mayor que cero"
	}

	if containsInvalidUserText(
		request.Nombres,
		request.Apellidos,
		request.Correo,
	) {
		return "los datos contienen caracteres inválidos"
	}

	if length := utf8.RuneCountInString(
		request.Nombres,
	); length < 2 || length > 100 {
		return "nombres debe contener entre 2 y 100 caracteres"
	}

	if length := utf8.RuneCountInString(
		request.Apellidos,
	); length < 2 || length > 100 {
		return "apellidos debe contener entre 2 y 100 caracteres"
	}

	if request.Correo == "" ||
		utf8.RuneCountInString(
			request.Correo,
		) > 150 {
		return "correo es obligatorio y admite máximo 150 caracteres"
	}

	address, err :=
		mail.ParseAddress(
			request.Correo,
		)

	if err != nil ||
		!strings.EqualFold(
			address.Address,
			request.Correo,
		) {
		return "correo no es válido"
	}

	return ""
}
