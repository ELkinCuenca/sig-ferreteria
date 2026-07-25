package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"sigefer.local/backend/internal/middleware"
	"sigefer.local/backend/internal/models"
	"sigefer.local/backend/internal/repository"
)

// AuthHandler procesa autenticación y sesiones.
type AuthHandler struct {
	repository *repository.AuthRepository
}

// NewAuthHandler crea el handler.
func NewAuthHandler(
	authRepository *repository.AuthRepository,
) *AuthHandler {
	return &AuthHandler{
		repository: authRepository,
	}
}

type loginResponse struct {
	Status           string          `json:"status"`
	Token            string          `json:"token"`
	TokenType        string          `json:"tipo_token"`
	ExpiresInSeconds int64           `json:"expira_en_segundos"`
	User             models.AuthUser `json:"usuario"`
}

type profileResponse struct {
	Status string          `json:"status"`
	User   models.AuthUser `json:"usuario"`
}

type logoutResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// Login procesa POST /api/v1/auth/login.
func (handler *AuthHandler) Login(
	writer http.ResponseWriter,
	request *http.Request,
) {
	writer.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)

	var loginRequest models.LoginRequest

	if err := decodeLoginRequest(
		request,
		&loginRequest,
	); err != nil {
		writer.WriteHeader(
			http.StatusBadRequest,
		)

		writeJSON(
			writer,
			ErrorResponse{
				Status:  "error",
				Message: err.Error(),
			},
		)
		return
	}

	loginRequest.Usuario = strings.TrimSpace(
		loginRequest.Usuario,
	)

	if loginRequest.Usuario == "" ||
		utf8.RuneCountInString(
			loginRequest.Usuario,
		) > 150 {
		writer.WriteHeader(
			http.StatusBadRequest,
		)

		writeJSON(
			writer,
			ErrorResponse{
				Status: "error",
				Message: "usuario es obligatorio " +
					"y admite máximo 150 caracteres",
			},
		)
		return
	}

	passwordLength := len(
		[]byte(loginRequest.Contrasena),
	)

	if passwordLength == 0 ||
		passwordLength > 72 {
		writer.WriteHeader(
			http.StatusBadRequest,
		)

		writeJSON(
			writer,
			ErrorResponse{
				Status: "error",
				Message: "contrasena es obligatoria " +
					"y admite máximo 72 bytes",
			},
		)
		return
	}

	ipAddress, userAgent :=
		requestMetadata(request)

	ctx, cancel := context.WithTimeout(
		request.Context(),
		15*time.Second,
	)
	defer cancel()

	result, err := handler.repository.Login(
		ctx,
		loginRequest.Usuario,
		loginRequest.Contrasena,
		ipAddress,
		userAgent,
	)

	// Reducir el tiempo que la contraseña permanece
	// referenciada por el objeto de solicitud.
	loginRequest.Contrasena = ""

	switch {
	case errors.Is(
		err,
		repository.ErrInvalidCredentials,
	):
		writer.WriteHeader(
			http.StatusUnauthorized,
		)

		writeJSON(
			writer,
			ErrorResponse{
				Status:  "error",
				Message: "usuario o contraseña incorrectos",
			},
		)
		return

	case errors.Is(
		err,
		repository.ErrAccountBlocked,
	):
		writer.WriteHeader(http.StatusLocked)

		writeJSON(
			writer,
			ErrorResponse{
				Status:  "error",
				Message: "la cuenta está bloqueada",
			},
		)
		return

	case errors.Is(
		err,
		repository.ErrAccountInactive,
	):
		writer.WriteHeader(http.StatusForbidden)

		writeJSON(
			writer,
			ErrorResponse{
				Status:  "error",
				Message: "la cuenta está inactiva",
			},
		)
		return

	case err != nil:
		log.Printf(
			"error durante el inicio de sesión: %v",
			err,
		)

		writer.WriteHeader(
			http.StatusInternalServerError,
		)

		writeJSON(
			writer,
			ErrorResponse{
				Status:  "error",
				Message: "no fue posible iniciar sesión",
			},
		)
		return
	}

	writeJSON(
		writer,
		loginResponse{
			Status:           "ok",
			Token:            result.Token,
			TokenType:        "Bearer",
			ExpiresInSeconds: result.ExpiresInSeconds,
			User:             result.User,
		},
	)
}

// Profile procesa GET /api/v1/auth/perfil.
func (handler *AuthHandler) Profile(
	writer http.ResponseWriter,
	request *http.Request,
) {
	writer.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)

	principal, valid :=
		middleware.PrincipalFromContext(
			request.Context(),
		)

	if !valid {
		writer.WriteHeader(
			http.StatusUnauthorized,
		)

		writeJSON(
			writer,
			ErrorResponse{
				Status:  "error",
				Message: "sesión no disponible",
			},
		)
		return
	}

	writeJSON(
		writer,
		profileResponse{
			Status: "ok",
			User:   principal.AuthUser,
		},
	)
}

// Logout procesa POST /api/v1/auth/logout.
func (handler *AuthHandler) Logout(
	writer http.ResponseWriter,
	request *http.Request,
) {
	writer.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)

	principal, valid :=
		middleware.PrincipalFromContext(
			request.Context(),
		)

	if !valid {
		writer.WriteHeader(
			http.StatusUnauthorized,
		)

		writeJSON(
			writer,
			ErrorResponse{
				Status:  "error",
				Message: "sesión no disponible",
			},
		)
		return
	}

	ipAddress, _ := requestMetadata(request)

	ctx, cancel := context.WithTimeout(
		request.Context(),
		10*time.Second,
	)
	defer cancel()

	err := handler.repository.Logout(
		ctx,
		principal,
		ipAddress,
	)
	if err != nil {
		log.Printf(
			"error cerrando la sesión: %v",
			err,
		)

		writer.WriteHeader(
			http.StatusInternalServerError,
		)

		writeJSON(
			writer,
			ErrorResponse{
				Status:  "error",
				Message: "no fue posible cerrar la sesión",
			},
		)
		return
	}

	writeJSON(
		writer,
		logoutResponse{
			Status:  "ok",
			Message: "sesión cerrada correctamente",
		},
	)
}

func decodeLoginRequest(
	request *http.Request,
	destination *models.LoginRequest,
) error {
	body := io.LimitReader(
		request.Body,
		16*1024,
	)

	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		return errors.New(
			"el cuerpo JSON no es válido",
		)
	}

	var trailing any

	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New(
			"el cuerpo debe contener un solo objeto JSON",
		)
	}

	return nil
}

func requestMetadata(
	request *http.Request,
) (string, string) {
	ipAddress := strings.TrimSpace(
		request.Header.Get("X-Forwarded-For"),
	)

	if separator := strings.Index(
		ipAddress,
		",",
	); separator >= 0 {
		ipAddress = strings.TrimSpace(
			ipAddress[:separator],
		)
	}

	if ipAddress == "" {
		ipAddress = strings.TrimSpace(
			request.Header.Get("X-Real-IP"),
		)
	}

	if ipAddress == "" {
		host, _, err := net.SplitHostPort(
			request.RemoteAddr,
		)

		if err == nil {
			ipAddress = host
		} else {
			ipAddress = request.RemoteAddr
		}
	}

	return truncateRunes(ipAddress, 50),
		truncateRunes(
			request.UserAgent(),
			500,
		)
}

func truncateRunes(
	value string,
	maximum int,
) string {
	runes := []rune(value)

	if len(runes) <= maximum {
		return value
	}

	return string(runes[:maximum])
}
