package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"sigefer.local/backend/internal/models"
)

var ErrUserAdminEmailDuplicate = errors.New(
	"el correo ya se encuentra registrado",
)

type userAdminUpdateSnapshot struct {
	IDRol int64

	Rol       string
	Nombres   string
	Apellidos string
	Correo    string
	Estado    string
}

// UpdateUser modifica los datos administrativos de
// una cuenta dentro de una transacción.
func (
	repository *UserAdminRepository,
) UpdateUser(
	ctx context.Context,
	targetUserID int64,
	request models.UpdateUserRequest,
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
				"no se pudo iniciar la edición del usuario: %w",
				err,
			)
	}

	committed := false

	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	previous, err := lockUserAdminUpdate(
		ctx,
		tx,
		targetUserID,
	)
	if err != nil {
		return models.UserAdmin{}, err
	}

	roleChanged :=
		request.IDRol != previous.IDRol

	dataChanged :=
		roleChanged ||
			request.Nombres != previous.Nombres ||
			request.Apellidos != previous.Apellidos ||
			request.Correo != previous.Correo

	if !dataChanged {
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

	if roleChanged &&
		previous.Rol == "ADMINISTRADOR" &&
		previous.Estado == "ACTIVO" {
		if err := protectLastActiveAdministrator(
			ctx,
			tx,
			previous.IDRol,
		); err != nil {
			return models.UserAdmin{}, err
		}
	}

	var newRoleName string

	err = tx.QueryRowContext(
		ctx,
		`
			SELECT NOMBRE
			FROM ROL
			WHERE ID_ROL = :1
			  AND ESTADO = 'A'
			FOR UPDATE
		`,
		request.IDRol,
	).Scan(&newRoleName)

	if errors.Is(err, sql.ErrNoRows) {
		return models.UserAdmin{},
			ErrUserAdminRoleNotFound
	}

	if err != nil {
		return models.UserAdmin{},
			fmt.Errorf(
				"no se pudo validar el nuevo rol: %w",
				err,
			)
	}

	var duplicateEmailCount int

	err = tx.QueryRowContext(
		ctx,
		`
			SELECT COUNT(*)
			FROM USUARIO
			WHERE LOWER(CORREO) = :1
			  AND ID_USUARIO <> :2
		`,
		request.Correo,
		targetUserID,
	).Scan(&duplicateEmailCount)

	if err != nil {
		return models.UserAdmin{},
			fmt.Errorf(
				"no se pudo validar el correo: %w",
				err,
			)
	}

	if duplicateEmailCount > 0 {
		return models.UserAdmin{},
			ErrUserAdminEmailDuplicate
	}

	result, err := tx.ExecContext(
		ctx,
		`
			UPDATE USUARIO
			SET
				ID_ROL = :1,
				NOMBRES = :2,
				APELLIDOS = :3,
				CORREO = :4,
				FECHA_ACTUALIZACION =
					CAST(
						SYSTIMESTAMP
						AT TIME ZONE '-05:00'
						AS TIMESTAMP
					)
			WHERE ID_USUARIO = :5
		`,
		request.IDRol,
		request.Nombres,
		request.Apellidos,
		request.Correo,
		targetUserID,
	)
	if err != nil {
		if strings.Contains(
			err.Error(),
			"ORA-00001",
		) {
			return models.UserAdmin{},
				ErrUserAdminEmailDuplicate
		}

		return models.UserAdmin{},
			fmt.Errorf(
				"no se pudo actualizar el usuario: %w",
				err,
			)
	}

	affectedRows, err :=
		result.RowsAffected()

	if err != nil {
		return models.UserAdmin{},
			fmt.Errorf(
				"no se pudo verificar la actualización: %w",
				err,
			)
	}

	if affectedRows != 1 {
		return models.UserAdmin{},
			ErrUserAdminNotFound
	}

	revokedSessions := int64(0)

	if roleChanged {
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

	if err := insertUserAdminActionAudit(
		ctx,
		tx,
		actorUserID,
		targetUserID,
		map[string]any{
			"id_rol": previous.IDRol,

			"rol": previous.Rol,

			"nombres": previous.Nombres,

			"apellidos": previous.Apellidos,

			"correo": previous.Correo,

			"estado": previous.Estado,
		},
		map[string]any{
			"id_rol": request.IDRol,

			"rol": newRoleName,

			"nombres": request.Nombres,

			"apellidos": request.Apellidos,

			"correo": request.Correo,

			"estado": previous.Estado,

			"rol_modificado": roleChanged,

			"sesiones_revocadas": revokedSessions,
		},
		ipAddress,
		"API REST PATCH /api/v1/usuarios/{id}",
	); err != nil {
		return models.UserAdmin{}, err
	}

	if err := tx.Commit(); err != nil {
		return models.UserAdmin{},
			fmt.Errorf(
				"no se pudo confirmar la edición: %w",
				err,
			)
	}

	committed = true

	return repository.GetUserByID(
		ctx,
		targetUserID,
	)
}

func lockUserAdminUpdate(
	ctx context.Context,
	tx *sql.Tx,
	userID int64,
) (userAdminUpdateSnapshot, error) {
	var snapshot userAdminUpdateSnapshot

	err := tx.QueryRowContext(
		ctx,
		`
			SELECT
				u.ID_ROL,
				r.NOMBRE,
				u.NOMBRES,
				u.APELLIDOS,
				u.CORREO,
				u.ESTADO
			FROM USUARIO u
			INNER JOIN ROL r
				ON r.ID_ROL = u.ID_ROL
			WHERE u.ID_USUARIO = :1
			FOR UPDATE OF
				u.ID_ROL,
				u.NOMBRES,
				u.APELLIDOS,
				u.CORREO
		`,
		userID,
	).Scan(
		&snapshot.IDRol,
		&snapshot.Rol,
		&snapshot.Nombres,
		&snapshot.Apellidos,
		&snapshot.Correo,
		&snapshot.Estado,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return userAdminUpdateSnapshot{},
			ErrUserAdminNotFound
	}

	if err != nil {
		return userAdminUpdateSnapshot{},
			fmt.Errorf(
				"no se pudo bloquear el usuario: %w",
				err,
			)
	}

	return snapshot, nil
}
