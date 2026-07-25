package repository

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"sigefer.local/backend/internal/models"
)

var (
	ErrInvalidCredentials = errors.New(
		"credenciales inválidas",
	)

	ErrAccountBlocked = errors.New(
		"cuenta bloqueada",
	)

	ErrAccountInactive = errors.New(
		"cuenta inactiva",
	)

	ErrInvalidSession = errors.New(
		"sesión inválida",
	)
)

// AuthRepository administra usuarios y sesiones.
type AuthRepository struct {
	db               *sql.DB
	sessionHours     int
	maxLoginAttempts int
}

// NewAuthRepository crea el repositorio.
func NewAuthRepository(
	db *sql.DB,
	sessionHours int,
	maxLoginAttempts int,
) *AuthRepository {
	return &AuthRepository{
		db:               db,
		sessionHours:     sessionHours,
		maxLoginAttempts: maxLoginAttempts,
	}
}

type loginUserRecord struct {
	models.AuthUser

	PasswordHash   string
	State          string
	FailedAttempts int
}

// Login valida credenciales y crea una sesión.
func (repository *AuthRepository) Login(
	ctx context.Context,
	identifier string,
	password string,
	ipAddress string,
	userAgent string,
) (models.LoginResult, error) {
	transaction, err := repository.db.BeginTx(
		ctx,
		nil,
	)
	if err != nil {
		return models.LoginResult{}, fmt.Errorf(
			"no se pudo iniciar la autenticación: %w",
			err,
		)
	}

	committed := false

	defer func() {
		if !committed {
			_ = transaction.Rollback()
		}
	}()

	normalizedIdentifier := strings.ToLower(
		strings.TrimSpace(identifier),
	)

	var user loginUserRecord

	err = transaction.QueryRowContext(
		ctx,
		`
			SELECT
				u.ID_USUARIO,
				u.NOMBRE_USUARIO,
				u.NOMBRES,
				u.APELLIDOS,
				u.CORREO,
				r.NOMBRE,
				u.CLAVE_HASH,
				u.ESTADO,
				u.INTENTOS_FALLIDOS
			FROM USUARIO u
			INNER JOIN ROL r
				ON r.ID_ROL = u.ID_ROL
			WHERE (
				LOWER(u.NOMBRE_USUARIO) = :1
				OR LOWER(u.CORREO) = :2
			)
			  AND r.ESTADO = 'A'
			FOR UPDATE
		`,
		normalizedIdentifier,
		normalizedIdentifier,
	).Scan(
		&user.IDUsuario,
		&user.NombreUsuario,
		&user.Nombres,
		&user.Apellidos,
		&user.Correo,
		&user.Rol,
		&user.PasswordHash,
		&user.State,
		&user.FailedAttempts,
	)

	if errors.Is(err, sql.ErrNoRows) {
		if auditErr := insertAuthAudit(
			ctx,
			transaction,
			nil,
			"ERROR",
			`{"motivo":"credenciales_invalidas"}`,
			ipAddress,
			"API REST POST /api/v1/auth/login",
		); auditErr != nil {
			return models.LoginResult{}, auditErr
		}

		if err := transaction.Commit(); err != nil {
			return models.LoginResult{}, fmt.Errorf(
				"no se pudo auditar el acceso: %w",
				err,
			)
		}

		committed = true

		return models.LoginResult{},
			ErrInvalidCredentials
	}

	if err != nil {
		return models.LoginResult{}, fmt.Errorf(
			"no se pudo consultar el usuario: %w",
			err,
		)
	}

	switch user.State {
	case "BLOQUEADO":
		if err := repository.auditRejectedLogin(
			ctx,
			transaction,
			user.IDUsuario,
			"cuenta_bloqueada",
			ipAddress,
		); err != nil {
			return models.LoginResult{}, err
		}

		committed = true

		return models.LoginResult{},
			ErrAccountBlocked

	case "INACTIVO":
		if err := repository.auditRejectedLogin(
			ctx,
			transaction,
			user.IDUsuario,
			"cuenta_inactiva",
			ipAddress,
		); err != nil {
			return models.LoginResult{}, err
		}

		committed = true

		return models.LoginResult{},
			ErrAccountInactive
	}

	passwordErr := bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(password),
	)

	if errors.Is(
		passwordErr,
		bcrypt.ErrMismatchedHashAndPassword,
	) {
		attempts := user.FailedAttempts + 1
		newState := "ACTIVO"

		if attempts >=
			repository.maxLoginAttempts {
			newState = "BLOQUEADO"
		}

		_, err = transaction.ExecContext(
			ctx,
			`
				UPDATE USUARIO
				SET
					INTENTOS_FALLIDOS = :1,
					ULTIMO_INTENTO_FALLIDO =
						CAST(
							SYSTIMESTAMP
							AT TIME ZONE '-05:00'
							AS TIMESTAMP
						),
					ESTADO = :2,
					FECHA_BLOQUEO =
						CASE
							WHEN :3 = 'BLOQUEADO'
								THEN CAST(
									SYSTIMESTAMP
									AT TIME ZONE '-05:00'
									AS TIMESTAMP
								)
							ELSE FECHA_BLOQUEO
						END,
					FECHA_ACTUALIZACION =
						CAST(
							SYSTIMESTAMP
							AT TIME ZONE '-05:00'
							AS TIMESTAMP
						)
				WHERE ID_USUARIO = :4
			`,
			attempts,
			newState,
			newState,
			user.IDUsuario,
		)
		if err != nil {
			return models.LoginResult{}, fmt.Errorf(
				"no se pudo registrar el intento fallido: %w",
				err,
			)
		}

		details, _ := json.Marshal(
			map[string]any{
				"motivo":   "credenciales_invalidas",
				"intentos": attempts,
				"estado":   newState,
			},
		)

		userID := user.IDUsuario

		if err := insertAuthAudit(
			ctx,
			transaction,
			&userID,
			"ERROR",
			string(details),
			ipAddress,
			"API REST POST /api/v1/auth/login",
		); err != nil {
			return models.LoginResult{}, err
		}

		if err := transaction.Commit(); err != nil {
			return models.LoginResult{}, fmt.Errorf(
				"no se pudo confirmar el intento fallido: %w",
				err,
			)
		}

		committed = true

		if newState == "BLOQUEADO" {
			return models.LoginResult{},
				ErrAccountBlocked
		}

		return models.LoginResult{},
			ErrInvalidCredentials
	}

	if passwordErr != nil {
		return models.LoginResult{}, fmt.Errorf(
			"el hash de contraseña no es válido: %w",
			passwordErr,
		)
	}

	rawToken, tokenHash, err :=
		generateSessionToken()

	if err != nil {
		return models.LoginResult{}, err
	}

	_, err = transaction.ExecContext(
		ctx,
		`
			UPDATE USUARIO
			SET
				INTENTOS_FALLIDOS = 0,
				ULTIMO_INTENTO_FALLIDO = NULL,
				FECHA_BLOQUEO = NULL,
				ULTIMO_ACCESO =
					CAST(
						SYSTIMESTAMP
						AT TIME ZONE '-05:00'
						AS TIMESTAMP
					),
				FECHA_ACTUALIZACION =
					CAST(
						SYSTIMESTAMP
						AT TIME ZONE '-05:00'
						AS TIMESTAMP
					)
			WHERE ID_USUARIO = :1
		`,
		user.IDUsuario,
	)
	if err != nil {
		return models.LoginResult{}, fmt.Errorf(
			"no se pudo actualizar el acceso: %w",
			err,
		)
	}

	_, err = transaction.ExecContext(
		ctx,
		`
			INSERT INTO SESION_USUARIO (
				ID_USUARIO,
				TOKEN_HASH,
				FECHA_CREACION,
				FECHA_EXPIRACION,
				ULTIMA_ACTIVIDAD,
				IP_ORIGEN,
				AGENTE_USUARIO
			)
			VALUES (
				:1,
				:2,
				CAST(
					SYSTIMESTAMP
					AT TIME ZONE '-05:00'
					AS TIMESTAMP
				),
				CAST(
					SYSTIMESTAMP
					AT TIME ZONE '-05:00'
					AS TIMESTAMP
				) + NUMTODSINTERVAL(
					:3,
					'HOUR'
				),
				CAST(
					SYSTIMESTAMP
					AT TIME ZONE '-05:00'
					AS TIMESTAMP
				),
				:4,
				:5
			)
		`,
		user.IDUsuario,
		tokenHash,
		repository.sessionHours,
		nullableText(ipAddress),
		nullableText(userAgent),
	)
	if err != nil {
		return models.LoginResult{}, fmt.Errorf(
			"no se pudo crear la sesión: %w",
			err,
		)
	}

	var sessionID int64

	err = transaction.QueryRowContext(
		ctx,
		`
			SELECT ID_SESION
			FROM SESION_USUARIO
			WHERE TOKEN_HASH = :1
		`,
		tokenHash,
	).Scan(&sessionID)
	if err != nil {
		return models.LoginResult{}, fmt.Errorf(
			"no se pudo recuperar la sesión: %w",
			err,
		)
	}

	userID := user.IDUsuario

	if err := insertAuthAudit(
		ctx,
		transaction,
		&userID,
		"LOGIN",
		`{"resultado":"exitoso"}`,
		ipAddress,
		"API REST POST /api/v1/auth/login",
	); err != nil {
		return models.LoginResult{}, err
	}

	if err := transaction.Commit(); err != nil {
		return models.LoginResult{}, fmt.Errorf(
			"no se pudo confirmar la sesión: %w",
			err,
		)
	}

	committed = true

	return models.LoginResult{
		Token: rawToken,

		ExpiresInSeconds: int64(repository.sessionHours) * 3600,

		User: user.AuthUser,
	}, nil
}

// Authenticate valida un token Bearer.
func (repository *AuthRepository) Authenticate(
	ctx context.Context,
	rawToken string,
) (models.AuthPrincipal, error) {
	tokenHash := hashSessionToken(rawToken)

	var principal models.AuthPrincipal

	err := repository.db.QueryRowContext(
		ctx,
		`
			SELECT
				s.ID_SESION,
				u.ID_USUARIO,
				u.NOMBRE_USUARIO,
				u.NOMBRES,
				u.APELLIDOS,
				u.CORREO,
				r.NOMBRE
			FROM SESION_USUARIO s
			INNER JOIN USUARIO u
				ON u.ID_USUARIO = s.ID_USUARIO
			INNER JOIN ROL r
				ON r.ID_ROL = u.ID_ROL
			WHERE s.TOKEN_HASH = :1
			  AND s.FECHA_REVOCACION IS NULL
			  AND s.FECHA_EXPIRACION >
				CAST(
					SYSTIMESTAMP
					AT TIME ZONE '-05:00'
					AS TIMESTAMP
				)
			  AND u.ESTADO = 'ACTIVO'
			  AND r.ESTADO = 'A'
		`,
		tokenHash,
	).Scan(
		&principal.IDSesion,
		&principal.IDUsuario,
		&principal.NombreUsuario,
		&principal.Nombres,
		&principal.Apellidos,
		&principal.Correo,
		&principal.Rol,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return models.AuthPrincipal{},
			ErrInvalidSession
	}

	if err != nil {
		return models.AuthPrincipal{}, fmt.Errorf(
			"no se pudo validar la sesión: %w",
			err,
		)
	}

	_, err = repository.db.ExecContext(
		ctx,
		`
			UPDATE SESION_USUARIO
			SET ULTIMA_ACTIVIDAD =
				CAST(
					SYSTIMESTAMP
					AT TIME ZONE '-05:00'
					AS TIMESTAMP
				)
			WHERE ID_SESION = :1
		`,
		principal.IDSesion,
	)
	if err != nil {
		return models.AuthPrincipal{}, fmt.Errorf(
			"no se pudo actualizar la sesión: %w",
			err,
		)
	}

	return principal, nil
}

// Logout revoca una sesión activa.
func (repository *AuthRepository) Logout(
	ctx context.Context,
	principal models.AuthPrincipal,
	ipAddress string,
) error {
	transaction, err := repository.db.BeginTx(
		ctx,
		nil,
	)
	if err != nil {
		return fmt.Errorf(
			"no se pudo iniciar el cierre de sesión: %w",
			err,
		)
	}

	committed := false

	defer func() {
		if !committed {
			_ = transaction.Rollback()
		}
	}()

	result, err := transaction.ExecContext(
		ctx,
		`
			UPDATE SESION_USUARIO
			SET
				FECHA_REVOCACION =
					CAST(
						SYSTIMESTAMP
						AT TIME ZONE '-05:00'
						AS TIMESTAMP
					),
				ULTIMA_ACTIVIDAD =
					CAST(
						SYSTIMESTAMP
						AT TIME ZONE '-05:00'
						AS TIMESTAMP
					)
			WHERE ID_SESION = :1
			  AND ID_USUARIO = :2
			  AND FECHA_REVOCACION IS NULL
		`,
		principal.IDSesion,
		principal.IDUsuario,
	)
	if err != nil {
		return fmt.Errorf(
			"no se pudo revocar la sesión: %w",
			err,
		)
	}

	affectedRows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf(
			"no se pudo verificar la revocación: %w",
			err,
		)
	}

	if affectedRows == 0 {
		return ErrInvalidSession
	}

	userID := principal.IDUsuario

	if err := insertAuthAudit(
		ctx,
		transaction,
		&userID,
		"LOGOUT",
		`{"resultado":"exitoso"}`,
		ipAddress,
		"API REST POST /api/v1/auth/logout",
	); err != nil {
		return err
	}

	if err := transaction.Commit(); err != nil {
		return fmt.Errorf(
			"no se pudo confirmar el cierre de sesión: %w",
			err,
		)
	}

	committed = true

	return nil
}

func (
	repository *AuthRepository,
) auditRejectedLogin(
	ctx context.Context,
	transaction *sql.Tx,
	userID int64,
	reason string,
	ipAddress string,
) error {
	details, _ := json.Marshal(
		map[string]string{
			"motivo": reason,
		},
	)

	if err := insertAuthAudit(
		ctx,
		transaction,
		&userID,
		"ERROR",
		string(details),
		ipAddress,
		"API REST POST /api/v1/auth/login",
	); err != nil {
		return err
	}

	if err := transaction.Commit(); err != nil {
		return fmt.Errorf(
			"no se pudo auditar el acceso rechazado: %w",
			err,
		)
	}

	return nil
}

func insertAuthAudit(
	ctx context.Context,
	transaction *sql.Tx,
	userID *int64,
	operation string,
	details string,
	ipAddress string,
	origin string,
) error {
	var nullableUserID any

	if userID != nil {
		nullableUserID = *userID
	}

	_, err := transaction.ExecContext(
		ctx,
		`
			INSERT INTO AUDITORIA (
				ID_USUARIO,
				TABLA_AFECTADA,
				OPERACION,
				VALORES_NUEVOS,
				IP_ORIGEN,
				ORIGEN,
				FECHA_EVENTO
			)
			VALUES (
				:1,
				'USUARIO',
				:2,
				:3,
				:4,
				:5,
				CAST(
					SYSTIMESTAMP
					AT TIME ZONE '-05:00'
					AS TIMESTAMP
				)
			)
		`,
		nullableUserID,
		operation,
		details,
		nullableText(ipAddress),
		origin,
	)
	if err != nil {
		return fmt.Errorf(
			"no se pudo registrar la auditoría: %w",
			err,
		)
	}

	return nil
}

func generateSessionToken() (
	string,
	string,
	error,
) {
	randomBytes := make([]byte, 32)

	if _, err := rand.Read(randomBytes); err != nil {
		return "", "", fmt.Errorf(
			"no se pudo generar el token: %w",
			err,
		)
	}

	rawToken := base64.RawURLEncoding.EncodeToString(
		randomBytes,
	)

	return rawToken,
		hashSessionToken(rawToken),
		nil
}

func hashSessionToken(
	rawToken string,
) string {
	sum := sha256.Sum256(
		[]byte(rawToken),
	)

	return hex.EncodeToString(sum[:])
}

func nullableText(
	value string,
) any {
	value = strings.TrimSpace(value)

	if value == "" {
		return nil
	}

	return value
}
