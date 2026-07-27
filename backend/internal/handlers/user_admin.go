package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"

	"sigefer.local/backend/internal/middleware"
	"sigefer.local/backend/internal/models"
	"sigefer.local/backend/internal/repository"
)

const (
	minimumUserPasswordLength = 12
	maximumUserPasswordBytes  = 72
)

var usernamePattern = regexp.MustCompile(
	`^[a-z0-9][a-z0-9._-]{2,49}$`,
)

// UserAdminHandler administra usuarios.
type UserAdminHandler struct {
	repository *repository.UserAdminRepository
}

// NewUserAdminHandler crea el handler.
func NewUserAdminHandler(
	userRepository *repository.UserAdminRepository,
) *UserAdminHandler {
	return &UserAdminHandler{
		repository: userRepository,
	}
}

type RoleListResponse struct {
	Status string               `json:"status"`
	Total  int                  `json:"total"`
	Roles  []models.RoleSummary `json:"roles"`
}

type UserAdminListResponse struct {
	Status   string             `json:"status"`
	Total    int                `json:"total"`
	Usuarios []models.UserAdmin `json:"usuarios"`
}

// ListRoles procesa GET /api/v1/roles.
func (handler *UserAdminHandler) ListRoles(
	writer http.ResponseWriter,
	request *http.Request,
) {
	ctx, cancel := context.WithTimeout(
		request.Context(),
		10*time.Second,
	)
	defer cancel()

	roles, err :=
		handler.repository.ListRoles(ctx)

	if err != nil {
		log.Printf(
			"error consultando roles: %v",
			err,
		)

		writeUserAdminError(
			writer,
			http.StatusInternalServerError,
			"no fue posible consultar los roles",
		)
		return
	}

	writeJSON(
		writer,
		RoleListResponse{
			Status: "ok",
			Total:  len(roles),
			Roles:  roles,
		},
	)
}

// List procesa GET /api/v1/usuarios.
func (handler *UserAdminHandler) List(
	writer http.ResponseWriter,
	request *http.Request,
) {
	ctx, cancel := context.WithTimeout(
		request.Context(),
		10*time.Second,
	)
	defer cancel()

	users, err :=
		handler.repository.ListUsers(ctx)

	if err != nil {
		log.Printf(
			"error consultando usuarios: %v",
			err,
		)

		writeUserAdminError(
			writer,
			http.StatusInternalServerError,
			"no fue posible consultar los usuarios",
		)
		return
	}

	writeJSON(
		writer,
		UserAdminListResponse{
			Status:   "ok",
			Total:    len(users),
			Usuarios: users,
		},
	)
}

// Get procesa GET /api/v1/usuarios/{id}.
func (handler *UserAdminHandler) Get(
	writer http.ResponseWriter,
	request *http.Request,
) {
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
		return
	}

	ctx, cancel := context.WithTimeout(
		request.Context(),
		10*time.Second,
	)
	defer cancel()

	user, err :=
		handler.repository.GetUserByID(
			ctx,
			userID,
		)

	if errors.Is(
		err,
		repository.ErrUserAdminNotFound,
	) {
		writeUserAdminError(
			writer,
			http.StatusNotFound,
			"usuario no encontrado",
		)
		return
	}

	if err != nil {
		log.Printf(
			"error consultando usuario %d: %v",
			userID,
			err,
		)

		writeUserAdminError(
			writer,
			http.StatusInternalServerError,
			"no fue posible consultar el usuario",
		)
		return
	}

	writeJSON(writer, user)
}

// Create procesa POST /api/v1/usuarios.
func (handler *UserAdminHandler) Create(
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

	var payload models.CreateUserRequest

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

	normalizeCreateUser(&payload)

	if message :=
		validateCreateUser(payload); message != "" {
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
			"error generando hash de usuario: %v",
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
		handler.repository.CreateUser(
			ctx,
			payload,
			string(passwordHash),
			principal.IDUsuario,
			clientIP(request),
		)

	switch {
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
		repository.ErrUserAdminDuplicate,
	):
		writeUserAdminError(
			writer,
			http.StatusConflict,
			"el nombre de usuario o correo ya está registrado",
		)

	case err != nil:
		log.Printf(
			"error creando usuario %s: %v",
			payload.NombreUsuario,
			err,
		)

		writeUserAdminError(
			writer,
			http.StatusInternalServerError,
			"no fue posible crear el usuario",
		)

	default:
		writer.Header().Set(
			"Location",
			"/api/v1/usuarios/"+
				strconv.FormatInt(
					user.IDUsuario,
					10,
				),
		)

		writer.WriteHeader(
			http.StatusCreated,
		)

		writeJSON(writer, user)
	}
}

func normalizeCreateUser(
	request *models.CreateUserRequest,
) {
	request.NombreUsuario =
		strings.ToLower(
			strings.TrimSpace(
				request.NombreUsuario,
			),
		)

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

func validateCreateUser(
	request models.CreateUserRequest,
) string {
	if request.IDRol <= 0 {
		return "id_rol debe ser mayor que cero"
	}

	if !usernamePattern.MatchString(
		request.NombreUsuario,
	) {
		return "nombre_usuario debe contener entre 3 y 50 caracteres y solo admite letras minúsculas, números, punto, guion o guion bajo"
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

	return validateUserPassword(
		request.Contrasena,
	)
}

func validateUserPassword(
	password string,
) string {
	passwordRunes := []rune(password)
	passwordBytes := []byte(password)

	if len(passwordRunes) <
		minimumUserPasswordLength {
		return "contrasena debe contener al menos 12 caracteres"
	}

	if len(passwordBytes) >
		maximumUserPasswordBytes {
		return "contrasena no puede superar 72 bytes"
	}

	var (
		hasUpper  bool
		hasLower  bool
		hasNumber bool
		hasSymbol bool
	)

	for _, character := range passwordRunes {
		switch {
		case unicode.IsUpper(character):
			hasUpper = true

		case unicode.IsLower(character):
			hasLower = true

		case unicode.IsDigit(character):
			hasNumber = true

		case unicode.IsPunct(character) ||
			unicode.IsSymbol(character):
			hasSymbol = true
		}
	}

	if !hasUpper ||
		!hasLower ||
		!hasNumber ||
		!hasSymbol {
		return "contrasena debe incluir mayúscula, minúscula, número y símbolo"
	}

	return ""
}

func containsInvalidUserText(
	values ...string,
) bool {
	for _, value := range values {
		if strings.ContainsRune(
			value,
			'\uFFFD',
		) {
			return true
		}
	}

	return false
}

func decodeUserAdminJSON(
	writer http.ResponseWriter,
	request *http.Request,
	destination any,
) error {
	request.Body = http.MaxBytesReader(
		writer,
		request.Body,
		64*1024,
	)

	decoder := json.NewDecoder(
		request.Body,
	)

	decoder.DisallowUnknownFields()

	if err := decoder.Decode(
		destination,
	); err != nil {
		return errors.New(
			"cuerpo JSON inválido",
		)
	}

	if err := decoder.Decode(
		&struct{}{},
	); !errors.Is(err, io.EOF) {
		return errors.New(
			"el cuerpo debe contener un único objeto JSON",
		)
	}

	return nil
}

func writeUserAdminError(
	writer http.ResponseWriter,
	statusCode int,
	message string,
) {
	writer.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)

	writer.WriteHeader(statusCode)

	writeJSON(
		writer,
		ErrorResponse{
			Status:  "error",
			Message: message,
		},
	)
}
