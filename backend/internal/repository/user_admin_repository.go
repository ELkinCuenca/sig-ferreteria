package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"sigefer.local/backend/internal/models"
)

var (
	ErrUserAdminNotFound = errors.New(
		"usuario no encontrado",
	)

	ErrUserAdminDuplicate = errors.New(
		"el nombre de usuario o correo ya está registrado",
	)

	ErrUserAdminRoleNotFound = errors.New(
		"el rol no existe o está inactivo",
	)
)

// UserAdminRepository administra cuentas desde el panel.
type UserAdminRepository struct {
	db *sql.DB
}

// NewUserAdminRepository crea el repositorio.
func NewUserAdminRepository(
	db *sql.DB,
) *UserAdminRepository {
	return &UserAdminRepository{
		db: db,
	}
}

// ListRoles devuelve los roles activos.
func (repository *UserAdminRepository) ListRoles(
	ctx context.Context,
) ([]models.RoleSummary, error) {
	rows, err := repository.db.QueryContext(
		ctx,
		`
			SELECT
				ID_ROL,
				NOMBRE,
				NVL(DESCRIPCION, ''),
				ESTADO
			FROM ROL
			WHERE ESTADO = 'A'
			ORDER BY ID_ROL
		`,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"no se pudieron consultar los roles: %w",
			err,
		)
	}
	defer rows.Close()

	roles := make(
		[]models.RoleSummary,
		0,
	)

	for rows.Next() {
		var role models.RoleSummary

		if err := rows.Scan(
			&role.IDRol,
			&role.Nombre,
			&role.Descripcion,
			&role.Estado,
		); err != nil {
			return nil, fmt.Errorf(
				"no se pudo interpretar un rol: %w",
				err,
			)
		}

		roles = append(
			roles,
			role,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"error recorriendo roles: %w",
			err,
		)
	}

	return roles, nil
}

// ListUsers devuelve todos los usuarios sin hashes.
func (repository *UserAdminRepository) ListUsers(
	ctx context.Context,
) ([]models.UserAdmin, error) {
	rows, err := repository.db.QueryContext(
		ctx,
		`
			SELECT
				u.ID_USUARIO,
				u.ID_ROL,
				u.NOMBRE_USUARIO,
				u.NOMBRES,
				u.APELLIDOS,
				u.CORREO,
				r.NOMBRE,
				u.ESTADO,
				u.INTENTOS_FALLIDOS,
				TO_CHAR(
					u.ULTIMO_ACCESO,
					'YYYY-MM-DD"T"HH24:MI:SS'
				),
				TO_CHAR(
					u.ULTIMO_INTENTO_FALLIDO,
					'YYYY-MM-DD"T"HH24:MI:SS'
				),
				TO_CHAR(
					u.FECHA_BLOQUEO,
					'YYYY-MM-DD"T"HH24:MI:SS'
				),
				TO_CHAR(
					u.FECHA_CREACION,
					'YYYY-MM-DD"T"HH24:MI:SS'
				),
				TO_CHAR(
					u.FECHA_ACTUALIZACION,
					'YYYY-MM-DD"T"HH24:MI:SS'
				)
			FROM USUARIO u
			INNER JOIN ROL r
				ON r.ID_ROL = u.ID_ROL
			ORDER BY u.ID_USUARIO
		`,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"no se pudieron consultar los usuarios: %w",
			err,
		)
	}
	defer rows.Close()

	users := make(
		[]models.UserAdmin,
		0,
	)

	for rows.Next() {
		user, err := scanUserAdmin(rows)
		if err != nil {
			return nil, fmt.Errorf(
				"no se pudo interpretar un usuario: %w",
				err,
			)
		}

		users = append(
			users,
			user,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"error recorriendo usuarios: %w",
			err,
		)
	}

	return users, nil
}

// GetUserByID devuelve un usuario específico.
func (repository *UserAdminRepository) GetUserByID(
	ctx context.Context,
	userID int64,
) (models.UserAdmin, error) {
	row := repository.db.QueryRowContext(
		ctx,
		`
			SELECT
				u.ID_USUARIO,
				u.ID_ROL,
				u.NOMBRE_USUARIO,
				u.NOMBRES,
				u.APELLIDOS,
				u.CORREO,
				r.NOMBRE,
				u.ESTADO,
				u.INTENTOS_FALLIDOS,
				TO_CHAR(
					u.ULTIMO_ACCESO,
					'YYYY-MM-DD"T"HH24:MI:SS'
				),
				TO_CHAR(
					u.ULTIMO_INTENTO_FALLIDO,
					'YYYY-MM-DD"T"HH24:MI:SS'
				),
				TO_CHAR(
					u.FECHA_BLOQUEO,
					'YYYY-MM-DD"T"HH24:MI:SS'
				),
				TO_CHAR(
					u.FECHA_CREACION,
					'YYYY-MM-DD"T"HH24:MI:SS'
				),
				TO_CHAR(
					u.FECHA_ACTUALIZACION,
					'YYYY-MM-DD"T"HH24:MI:SS'
				)
			FROM USUARIO u
			INNER JOIN ROL r
				ON r.ID_ROL = u.ID_ROL
			WHERE u.ID_USUARIO = :1
		`,
		userID,
	)

	user, err := scanUserAdmin(row)

	if errors.Is(err, sql.ErrNoRows) {
		return models.UserAdmin{},
			ErrUserAdminNotFound
	}

	if err != nil {
		return models.UserAdmin{},
			fmt.Errorf(
				"no se pudo consultar el usuario: %w",
				err,
			)
	}

	return user, nil
}

// CreateUser crea un usuario activo y registra auditoría.
func (repository *UserAdminRepository) CreateUser(
	ctx context.Context,
	request models.CreateUserRequest,
	passwordHash string,
	actorUserID int64,
	ipAddress string,
) (models.UserAdmin, error) {
	tx, err := repository.db.BeginTx(
		ctx,
		nil,
	)
	if err != nil {
		return models.UserAdmin{},
			fmt.Errorf(
				"no se pudo iniciar la creación: %w",
				err,
			)
	}

	committed := false

	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	var roleName string

	err = tx.QueryRowContext(
		ctx,
		`
			SELECT NOMBRE
			FROM ROL
			WHERE ID_ROL = :1
			  AND ESTADO = 'A'
		`,
		request.IDRol,
	).Scan(&roleName)

	if errors.Is(err, sql.ErrNoRows) {
		return models.UserAdmin{},
			ErrUserAdminRoleNotFound
	}

	if err != nil {
		return models.UserAdmin{},
			fmt.Errorf(
				"no se pudo validar el rol: %w",
				err,
			)
	}

	var duplicateCount int

	err = tx.QueryRowContext(
		ctx,
		`
			SELECT COUNT(*)
			FROM USUARIO
			WHERE LOWER(NOMBRE_USUARIO) = :1
			   OR LOWER(CORREO) = :2
		`,
		request.NombreUsuario,
		request.Correo,
	).Scan(&duplicateCount)

	if err != nil {
		return models.UserAdmin{},
			fmt.Errorf(
				"no se pudo validar el usuario: %w",
				err,
			)
	}

	if duplicateCount > 0 {
		return models.UserAdmin{},
			ErrUserAdminDuplicate
	}

	var newUserID int64

	_, err = tx.ExecContext(
		ctx,
		`
			INSERT INTO USUARIO (
				ID_ROL,
				NOMBRE_USUARIO,
				NOMBRES,
				APELLIDOS,
				CORREO,
				CLAVE_HASH,
				ESTADO,
				INTENTOS_FALLIDOS,
				FECHA_CREACION,
				FECHA_ACTUALIZACION
			)
			VALUES (
				:1,
				:2,
				:3,
				:4,
				:5,
				:6,
				'ACTIVO',
				0,
				CAST(
					SYSTIMESTAMP
					AT TIME ZONE '-05:00'
					AS TIMESTAMP
				),
				CAST(
					SYSTIMESTAMP
					AT TIME ZONE '-05:00'
					AS TIMESTAMP
				)
			)
			RETURNING ID_USUARIO
			INTO :7
		`,
		request.IDRol,
		request.NombreUsuario,
		request.Nombres,
		request.Apellidos,
		request.Correo,
		passwordHash,
		sql.Out{
			Dest: &newUserID,
		},
	)
	if err != nil {
		if strings.Contains(
			err.Error(),
			"ORA-00001",
		) {
			return models.UserAdmin{},
				ErrUserAdminDuplicate
		}

		return models.UserAdmin{},
			fmt.Errorf(
				"no se pudo insertar el usuario: %w",
				err,
			)
	}

	if err := insertUserCreationAudit(
		ctx,
		tx,
		actorUserID,
		newUserID,
		request,
		roleName,
		ipAddress,
	); err != nil {
		return models.UserAdmin{}, err
	}

	if err := tx.Commit(); err != nil {
		return models.UserAdmin{},
			fmt.Errorf(
				"no se pudo confirmar la creación: %w",
				err,
			)
	}

	committed = true

	return repository.GetUserByID(
		ctx,
		newUserID,
	)
}

type userAdminScanner interface {
	Scan(dest ...any) error
}

func scanUserAdmin(
	scanner userAdminScanner,
) (models.UserAdmin, error) {
	var (
		user models.UserAdmin

		lastAccess        sql.NullString
		lastFailedAttempt sql.NullString
		blockDate         sql.NullString
	)

	err := scanner.Scan(
		&user.IDUsuario,
		&user.IDRol,
		&user.NombreUsuario,
		&user.Nombres,
		&user.Apellidos,
		&user.Correo,
		&user.Rol,
		&user.Estado,
		&user.IntentosFallidos,
		&lastAccess,
		&lastFailedAttempt,
		&blockDate,
		&user.FechaCreacion,
		&user.FechaActualizacion,
	)
	if err != nil {
		return models.UserAdmin{}, err
	}

	if lastAccess.Valid {
		user.UltimoAcceso =
			lastAccess.String
	}

	if lastFailedAttempt.Valid {
		user.UltimoIntentoFallido =
			lastFailedAttempt.String
	}

	if blockDate.Valid {
		user.FechaBloqueo =
			blockDate.String
	}

	return user, nil
}

func insertUserCreationAudit(
	ctx context.Context,
	tx *sql.Tx,
	actorUserID int64,
	newUserID int64,
	request models.CreateUserRequest,
	roleName string,
	ipAddress string,
) error {
	values, err := json.Marshal(
		map[string]any{
			"id_rol": request.IDRol,

			"rol": roleName,

			"nombre_usuario": request.NombreUsuario,

			"nombres": request.Nombres,

			"apellidos": request.Apellidos,

			"correo": request.Correo,

			"estado": "ACTIVO",
		},
	)
	if err != nil {
		return fmt.Errorf(
			"no se pudo construir la auditoría: %w",
			err,
		)
	}

	_, err = tx.ExecContext(
		ctx,
		`
			INSERT INTO AUDITORIA (
				ID_USUARIO,
				TABLA_AFECTADA,
				OPERACION,
				ID_REGISTRO,
				VALORES_NUEVOS,
				IP_ORIGEN,
				ORIGEN,
				FECHA_EVENTO
			)
			VALUES (
				:1,
				'USUARIO',
				'INSERT',
				:2,
				:3,
				:4,
				'API REST USUARIOS',
				CAST(
					SYSTIMESTAMP
					AT TIME ZONE '-05:00'
					AS TIMESTAMP
				)
			)
		`,
		actorUserID,
		fmt.Sprintf(
			"%d",
			newUserID,
		),
		string(values),
		nullableText(ipAddress),
	)
	if err != nil {
		return fmt.Errorf(
			"no se pudo auditar el usuario: %w",
			err,
		)
	}

	return nil
}
