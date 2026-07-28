package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"sigefer.local/backend/internal/models"
)

var (
	ErrUserAdminSelfDeactivation = errors.New(
		"el administrador no puede desactivar su propia cuenta",
	)

	ErrUserAdminLastAdministrator = errors.New(
		"no se puede desactivar el último administrador activo",
	)

	ErrUserAdminNotBlocked = errors.New(
		"el usuario no se encuentra bloqueado",
	)
)

type userAdminActionSnapshot struct {
	IDRol            int64
	Rol              string
	Estado           string
	IntentosFallidos int
}

// UpdateUserState activa o desactiva una cuenta.
func (
	repository *UserAdminRepository,
) UpdateUserState(
	ctx context.Context,
	targetUserID int64,
	newState string,
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
				"no se pudo iniciar el cambio de estado: %w",
				err,
			)
	}

	committed := false

	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	previous, err := lockUserAdminAction(
		ctx,
		tx,
		targetUserID,
	)
	if err != nil {
		return models.UserAdmin{}, err
	}

	if targetUserID == actorUserID &&
		newState == "INACTIVO" {
		return models.UserAdmin{},
			ErrUserAdminSelfDeactivation
	}

	if previous.Estado == newState {
		if err := tx.Commit(); err != nil {
			return models.UserAdmin{},
				fmt.Errorf(
					"no se pudo finalizar la consulta: %w",
					err,
				)
		}

		committed = true

		return repository.GetUserByID(
			ctx,
			targetUserID,
		)
	}

	if previous.Rol == "ADMINISTRADOR" &&
		newState == "INACTIVO" {
		if err := protectLastActiveAdministrator(
			ctx,
			tx,
			previous.IDRol,
		); err != nil {
			return models.UserAdmin{}, err
		}
	}

	result, err := tx.ExecContext(
		ctx,
		`
			UPDATE USUARIO
			SET
				ESTADO = :1,
				INTENTOS_FALLIDOS =
					CASE
						WHEN :2 = 'ACTIVO'
							THEN 0
						ELSE INTENTOS_FALLIDOS
					END,
				ULTIMO_INTENTO_FALLIDO =
					CASE
						WHEN :3 = 'ACTIVO'
							THEN NULL
						ELSE ULTIMO_INTENTO_FALLIDO
					END,
				FECHA_BLOQUEO =
					CASE
						WHEN :4 = 'ACTIVO'
							THEN NULL
						ELSE FECHA_BLOQUEO
					END,
				FECHA_ACTUALIZACION =
					CAST(
						SYSTIMESTAMP
						AT TIME ZONE '-05:00'
						AS TIMESTAMP
					)
			WHERE ID_USUARIO = :5
		`,
		newState,
		newState,
		newState,
		newState,
		targetUserID,
	)
	if err != nil {
		return models.UserAdmin{},
			fmt.Errorf(
				"no se pudo actualizar el estado: %w",
				err,
			)
	}

	affectedRows, err :=
		result.RowsAffected()

	if err != nil {
		return models.UserAdmin{},
			fmt.Errorf(
				"no se pudo verificar el estado: %w",
				err,
			)
	}

	if affectedRows != 1 {
		return models.UserAdmin{},
			ErrUserAdminNotFound
	}

	revokedSessions := int64(0)

	if newState == "INACTIVO" {
		revokedSessions, err =
			revokeUserAdminSessions(
				ctx,
				tx,
				targetUserID,
			)

		if err != nil {
			return models.UserAdmin{}, err
		}
	}

	newFailedAttempts :=
		previous.IntentosFallidos

	if newState == "ACTIVO" {
		newFailedAttempts = 0
	}

	if err := insertUserAdminActionAudit(
		ctx,
		tx,
		actorUserID,
		targetUserID,
		map[string]any{
			"estado": previous.Estado,

			"intentos_fallidos": previous.IntentosFallidos,
		},
		map[string]any{
			"estado": newState,

			"intentos_fallidos": newFailedAttempts,

			"sesiones_revocadas": revokedSessions,
		},
		ipAddress,
		"API REST PATCH /api/v1/usuarios/{id}/estado",
	); err != nil {
		return models.UserAdmin{}, err
	}

	if err := tx.Commit(); err != nil {
		return models.UserAdmin{},
			fmt.Errorf(
				"no se pudo confirmar el cambio de estado: %w",
				err,
			)
	}

	committed = true

	return repository.GetUserByID(
		ctx,
		targetUserID,
	)
}

// UnlockUser reactiva una cuenta bloqueada,
// reinicia intentos y revoca sesiones anteriores.
func (
	repository *UserAdminRepository,
) UnlockUser(
	ctx context.Context,
	targetUserID int64,
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
				"no se pudo iniciar el desbloqueo: %w",
				err,
			)
	}

	committed := false

	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	previous, err := lockUserAdminAction(
		ctx,
		tx,
		targetUserID,
	)
	if err != nil {
		return models.UserAdmin{}, err
	}

	if previous.Estado != "BLOQUEADO" {
		return models.UserAdmin{},
			ErrUserAdminNotBlocked
	}

	revokedSessions, err :=
		revokeUserAdminSessions(
			ctx,
			tx,
			targetUserID,
		)

	if err != nil {
		return models.UserAdmin{}, err
	}

	result, err := tx.ExecContext(
		ctx,
		`
			UPDATE USUARIO
			SET
				ESTADO = 'ACTIVO',
				INTENTOS_FALLIDOS = 0,
				ULTIMO_INTENTO_FALLIDO = NULL,
				FECHA_BLOQUEO = NULL,
				FECHA_ACTUALIZACION =
					CAST(
						SYSTIMESTAMP
						AT TIME ZONE '-05:00'
						AS TIMESTAMP
					)
			WHERE ID_USUARIO = :1
		`,
		targetUserID,
	)
	if err != nil {
		return models.UserAdmin{},
			fmt.Errorf(
				"no se pudo desbloquear el usuario: %w",
				err,
			)
	}

	affectedRows, err :=
		result.RowsAffected()

	if err != nil {
		return models.UserAdmin{},
			fmt.Errorf(
				"no se pudo verificar el desbloqueo: %w",
				err,
			)
	}

	if affectedRows != 1 {
		return models.UserAdmin{},
			ErrUserAdminNotFound
	}

	if err := insertUserAdminActionAudit(
		ctx,
		tx,
		actorUserID,
		targetUserID,
		map[string]any{
			"estado": previous.Estado,

			"intentos_fallidos": previous.IntentosFallidos,
		},
		map[string]any{
			"estado": "ACTIVO",

			"intentos_fallidos": 0,

			"sesiones_revocadas": revokedSessions,

			"accion": "DESBLOQUEO",
		},
		ipAddress,
		"API REST PATCH /api/v1/usuarios/{id}/desbloquear",
	); err != nil {
		return models.UserAdmin{}, err
	}

	if err := tx.Commit(); err != nil {
		return models.UserAdmin{},
			fmt.Errorf(
				"no se pudo confirmar el desbloqueo: %w",
				err,
			)
	}

	committed = true

	return repository.GetUserByID(
		ctx,
		targetUserID,
	)
}

// ResetUserPassword reemplaza el hash,
// limpia bloqueos y revoca sesiones activas.
func (
	repository *UserAdminRepository,
) ResetUserPassword(
	ctx context.Context,
	targetUserID int64,
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
				"no se pudo iniciar el cambio de contraseña: %w",
				err,
			)
	}

	committed := false

	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	previous, err := lockUserAdminAction(
		ctx,
		tx,
		targetUserID,
	)
	if err != nil {
		return models.UserAdmin{}, err
	}

	newState := previous.Estado

	if previous.Estado == "BLOQUEADO" {
		newState = "ACTIVO"
	}

	revokedSessions, err :=
		revokeUserAdminSessions(
			ctx,
			tx,
			targetUserID,
		)

	if err != nil {
		return models.UserAdmin{}, err
	}

	result, err := tx.ExecContext(
		ctx,
		`
			UPDATE USUARIO
			SET
				CLAVE_HASH = :1,
				ESTADO = :2,
				INTENTOS_FALLIDOS = 0,
				ULTIMO_INTENTO_FALLIDO = NULL,
				FECHA_BLOQUEO = NULL,
				FECHA_ACTUALIZACION =
					CAST(
						SYSTIMESTAMP
						AT TIME ZONE '-05:00'
						AS TIMESTAMP
					)
			WHERE ID_USUARIO = :3
		`,
		passwordHash,
		newState,
		targetUserID,
	)
	if err != nil {
		return models.UserAdmin{},
			fmt.Errorf(
				"no se pudo actualizar la contraseña: %w",
				err,
			)
	}

	affectedRows, err :=
		result.RowsAffected()

	if err != nil {
		return models.UserAdmin{},
			fmt.Errorf(
				"no se pudo verificar el cambio de contraseña: %w",
				err,
			)
	}

	if affectedRows != 1 {
		return models.UserAdmin{},
			ErrUserAdminNotFound
	}

	if err := insertUserAdminActionAudit(
		ctx,
		tx,
		actorUserID,
		targetUserID,
		map[string]any{
			"estado": previous.Estado,

			"intentos_fallidos": previous.IntentosFallidos,

			"contrasena": "PROTEGIDA",
		},
		map[string]any{
			"estado": newState,

			"intentos_fallidos": 0,

			"contrasena": "RESTABLECIDA",

			"sesiones_revocadas": revokedSessions,
		},
		ipAddress,
		"API REST PATCH /api/v1/usuarios/{id}/contrasena",
	); err != nil {
		return models.UserAdmin{}, err
	}

	if err := tx.Commit(); err != nil {
		return models.UserAdmin{},
			fmt.Errorf(
				"no se pudo confirmar el cambio de contraseña: %w",
				err,
			)
	}

	committed = true

	return repository.GetUserByID(
		ctx,
		targetUserID,
	)
}

func lockUserAdminAction(
	ctx context.Context,
	tx *sql.Tx,
	userID int64,
) (userAdminActionSnapshot, error) {
	var snapshot userAdminActionSnapshot

	err := tx.QueryRowContext(
		ctx,
		`
			SELECT
				u.ID_ROL,
				r.NOMBRE,
				u.ESTADO,
				u.INTENTOS_FALLIDOS
			FROM USUARIO u
			INNER JOIN ROL r
				ON r.ID_ROL = u.ID_ROL
			WHERE u.ID_USUARIO = :1
			FOR UPDATE OF
				u.ESTADO,
				u.INTENTOS_FALLIDOS
		`,
		userID,
	).Scan(
		&snapshot.IDRol,
		&snapshot.Rol,
		&snapshot.Estado,
		&snapshot.IntentosFallidos,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return userAdminActionSnapshot{},
			ErrUserAdminNotFound
	}

	if err != nil {
		return userAdminActionSnapshot{},
			fmt.Errorf(
				"no se pudo bloquear el usuario: %w",
				err,
			)
	}

	return snapshot, nil
}

func protectLastActiveAdministrator(
	ctx context.Context,
	tx *sql.Tx,
	adminRoleID int64,
) error {
	var lockedRoleID int64

	err := tx.QueryRowContext(
		ctx,
		`
			SELECT ID_ROL
			FROM ROL
			WHERE ID_ROL = :1
			FOR UPDATE
		`,
		adminRoleID,
	).Scan(&lockedRoleID)

	if err != nil {
		return fmt.Errorf(
			"no se pudo bloquear el rol administrador: %w",
			err,
		)
	}

	var activeAdministrators int

	err = tx.QueryRowContext(
		ctx,
		`
			SELECT COUNT(*)
			FROM USUARIO
			WHERE ID_ROL = :1
			  AND ESTADO = 'ACTIVO'
		`,
		lockedRoleID,
	).Scan(&activeAdministrators)

	if err != nil {
		return fmt.Errorf(
			"no se pudieron verificar los administradores: %w",
			err,
		)
	}

	if activeAdministrators <= 1 {
		return ErrUserAdminLastAdministrator
	}

	return nil
}

func revokeUserAdminSessions(
	ctx context.Context,
	tx *sql.Tx,
	userID int64,
) (int64, error) {
	result, err := tx.ExecContext(
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
			WHERE ID_USUARIO = :1
			  AND FECHA_REVOCACION IS NULL
		`,
		userID,
	)
	if err != nil {
		return 0, fmt.Errorf(
			"no se pudieron revocar las sesiones: %w",
			err,
		)
	}

	affectedRows, err :=
		result.RowsAffected()

	if err != nil {
		return 0, fmt.Errorf(
			"no se pudieron verificar las sesiones revocadas: %w",
			err,
		)
	}

	return affectedRows, nil
}

func insertUserAdminActionAudit(
	ctx context.Context,
	tx *sql.Tx,
	actorUserID int64,
	targetUserID int64,
	previousValues map[string]any,
	newValues map[string]any,
	ipAddress string,
	origin string,
) error {
	previousJSON, err :=
		json.Marshal(previousValues)

	if err != nil {
		return fmt.Errorf(
			"no se pudo construir la auditoría anterior: %w",
			err,
		)
	}

	newJSON, err :=
		json.Marshal(newValues)

	if err != nil {
		return fmt.Errorf(
			"no se pudo construir la auditoría nueva: %w",
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
				VALORES_ANTERIORES,
				VALORES_NUEVOS,
				IP_ORIGEN,
				ORIGEN,
				FECHA_EVENTO
			)
			VALUES (
				:1,
				'USUARIO',
				'UPDATE',
				:2,
				:3,
				:4,
				:5,
				:6,
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
			targetUserID,
		),
		string(previousJSON),
		string(newJSON),
		nullableText(ipAddress),
		origin,
	)
	if err != nil {
		return fmt.Errorf(
			"no se pudo auditar la operación del usuario: %w",
			err,
		)
	}

	return nil
}
