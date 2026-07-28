package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"sigefer.local/backend/internal/middleware"
	"sigefer.local/backend/internal/models"
	"sigefer.local/backend/internal/repository"
)

// UpdateState procesa
// PATCH /api/v1/usuarios/{id}/estado.
func (
	handler *UserAdminHandler,
) UpdateState(
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

	var payload models.UpdateUserStateRequest

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

	payload.Estado =
		strings.ToUpper(
			strings.TrimSpace(
				payload.Estado,
			),
		)

	if payload.Estado != "ACTIVO" &&
		payload.Estado != "INACTIVO" {
		writeUserAdminError(
			writer,
			http.StatusBadRequest,
			"estado debe ser ACTIVO o INACTIVO",
		)
		return
	}

	ctx, cancel := context.WithTimeout(
		request.Context(),
		20*time.Second,
	)
	defer cancel()

	user, err :=
		handler.repository.UpdateUserState(
			ctx,
			userID,
			payload.Estado,
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
		repository.ErrUserAdminSelfDeactivation,
	):
		writeUserAdminError(
			writer,
			http.StatusConflict,
			"no puedes desactivar tu propia cuenta",
		)

	case errors.Is(
		err,
		repository.ErrUserAdminLastAdministrator,
	):
		writeUserAdminError(
			writer,
			http.StatusConflict,
			"no se puede desactivar el último administrador activo",
		)

	case err != nil:
		log.Printf(
			"error actualizando estado del usuario %d: %v",
			userID,
			err,
		)

		writeUserAdminError(
			writer,
			http.StatusInternalServerError,
			"no fue posible actualizar el estado",
		)

	default:
		writeJSON(writer, user)
	}
}

// Unlock procesa
// PATCH /api/v1/usuarios/{id}/desbloquear.
func (
	handler *UserAdminHandler,
) Unlock(
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

	ctx, cancel := context.WithTimeout(
		request.Context(),
		20*time.Second,
	)
	defer cancel()

	user, err :=
		handler.repository.UnlockUser(
			ctx,
			userID,
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
		repository.ErrUserAdminNotBlocked,
	):
		writeUserAdminError(
			writer,
			http.StatusConflict,
			"el usuario no se encuentra bloqueado",
		)

	case err != nil:
		log.Printf(
			"error desbloqueando usuario %d: %v",
			userID,
			err,
		)

		writeUserAdminError(
			writer,
			http.StatusInternalServerError,
			"no fue posible desbloquear el usuario",
		)

	default:
		writeJSON(writer, user)
	}
}

// ResetPassword procesa
// PATCH /api/v1/usuarios/{id}/contrasena.
func (
	handler *UserAdminHandler,
) ResetPassword(
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

	var payload models.ResetUserPasswordRequest

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

	if message :=
		validateUserPassword(
			payload.Contrasena,
		); message != "" {
		writeUserAdminError(
			writer,
			http.StatusBadRequest,
			message,
		)
		return
	}

	passwordHash, err :=
		bcrypt.GenerateFromPassword(
			[]byte(payload.Contrasena),
			bcrypt.DefaultCost,
		)

	payload.Contrasena = ""

	if err != nil {
		log.Printf(
			"error generando hash de contraseña: %v",
			err,
		)

		writeUserAdminError(
			writer,
			http.StatusInternalServerError,
			"no fue posible proteger la contraseña",
		)
		return
	}

	ctx, cancel := context.WithTimeout(
		request.Context(),
		20*time.Second,
	)
	defer cancel()

	user, err :=
		handler.repository.ResetUserPassword(
			ctx,
			userID,
			string(passwordHash),
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

	case err != nil:
		log.Printf(
			"error restableciendo contraseña del usuario %d: %v",
			userID,
			err,
		)

		writeUserAdminError(
			writer,
			http.StatusInternalServerError,
			"no fue posible restablecer la contraseña",
		)

	default:
		writeJSON(writer, user)
	}
}

func parseUserAdminPathID(
	writer http.ResponseWriter,
	request *http.Request,
) (int64, bool) {
	userID, err := strconv.ParseInt(
		request.PathValue("id"),
		10,
		64,
	)

	if err != nil || userID <= 0 {
		writeUserAdminError(
			writer,
			http.StatusBadRequest,
			"id de usuario inválido",
		)

		return 0, false
	}

	return userID, true
}
