package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"sigefer.local/backend/internal/models"
)

var (
	ErrBPMProductNotFound = errors.New(
		"producto no encontrado",
	)

	ErrBPMProviderNotFound = errors.New(
		"proveedor no encontrado",
	)

	ErrBPMAlertNotFound = errors.New(
		"alerta no encontrada",
	)

	ErrBPMAlertUnavailable = errors.New(
		"alerta no disponible",
	)

	ErrBPMActiveRequestExists = errors.New(
		"ya existe una reposición activa",
	)

	ErrBPMRequestNotFound = errors.New(
		"solicitud de reposición no encontrada",
	)

	ErrBPMInvalidTransition = errors.New(
		"transición BPM no permitida",
	)
)

// BPMRepository administra el proceso de reposición.
type BPMRepository struct {
	db *sql.DB
}

// NewBPMRepository crea el repositorio BPM.
func NewBPMRepository(
	db *sql.DB,
) *BPMRepository {
	return &BPMRepository{
		db: db,
	}
}

// ListProviders devuelve los proveedores activos.
func (repository *BPMRepository) ListProviders(
	ctx context.Context,
) ([]models.BPMProvider, error) {
	rows, err := repository.db.QueryContext(
		ctx,
		`
			SELECT
				ID_PROVEEDOR,
				RUC,
				RAZON_SOCIAL,
				NOMBRE_CONTACTO,
				TELEFONO,
				CORREO
			FROM PROVEEDOR
			WHERE ESTADO = 'A'
			ORDER BY RAZON_SOCIAL
		`,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"no se pudieron consultar los proveedores: %w",
			err,
		)
	}
	defer rows.Close()

	providers := make(
		[]models.BPMProvider,
		0,
	)

	for rows.Next() {
		var (
			provider models.BPMProvider
			contact  sql.NullString
			phone    sql.NullString
			email    sql.NullString
		)

		if err := rows.Scan(
			&provider.IDProveedor,
			&provider.RUC,
			&provider.RazonSocial,
			&contact,
			&phone,
			&email,
		); err != nil {
			return nil, fmt.Errorf(
				"no se pudo leer un proveedor: %w",
				err,
			)
		}

		if contact.Valid {
			provider.NombreContacto =
				contact.String
		}

		if phone.Valid {
			provider.Telefono =
				phone.String
		}

		if email.Valid {
			provider.Correo =
				email.String
		}

		providers = append(
			providers,
			provider,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"error recorriendo proveedores: %w",
			err,
		)
	}

	return providers, nil
}

// ListReplenishments devuelve solicitudes filtradas.
func (
	repository *BPMRepository,
) ListReplenishments(
	ctx context.Context,
	state string,
	limit int,
) ([]models.Replenishment, error) {
	query := `
		SELECT *
		FROM (
			SELECT
				ID_SOLICITUD,
				NUMERO_SOLICITUD,
				CODIGO_PRODUCTO,
				PRODUCTO,
				UNIDAD_MEDIDA,
				ID_PROVEEDOR,
				RUC_PROVEEDOR,
				PROVEEDOR,
				ID_ALERTA,
				TIPO_ALERTA,
				ESTADO_ALERTA,
				CANTIDAD_SOLICITADA,
				CANTIDAD_RECIBIDA,
				COSTO_UNITARIO_ESTIMADO,
				COSTO_TOTAL_ESTIMADO,
				ESTADO,
				STOCK_ACTUAL,
				STOCK_RESERVADO,
				STOCK_DISPONIBLE,
				STOCK_MINIMO,
				ID_USUARIO_SOLICITANTE,
				USUARIO_SOLICITANTE,
				ID_USUARIO_APROBADOR,
				USUARIO_APROBADOR,
				ID_USUARIO_RECEPTOR,
				USUARIO_RECEPTOR,
				TO_CHAR(
					FECHA_SOLICITUD,
					'YYYY-MM-DD"T"HH24:MI:SS'
				),
				TO_CHAR(
					FECHA_APROBACION,
					'YYYY-MM-DD"T"HH24:MI:SS'
				),
				TO_CHAR(
					FECHA_RECHAZO,
					'YYYY-MM-DD"T"HH24:MI:SS'
				),
				TO_CHAR(
					FECHA_PEDIDO,
					'YYYY-MM-DD"T"HH24:MI:SS'
				),
				TO_CHAR(
					FECHA_RECEPCION,
					'YYYY-MM-DD"T"HH24:MI:SS'
				),
				TO_CHAR(
					FECHA_CIERRE,
					'YYYY-MM-DD"T"HH24:MI:SS'
				),
				OBSERVACION,
				MOTIVO_RECHAZO,
				TO_CHAR(
					FECHA_CREACION,
					'YYYY-MM-DD"T"HH24:MI:SS'
				),
				TO_CHAR(
					FECHA_ACTUALIZACION,
					'YYYY-MM-DD"T"HH24:MI:SS'
				)
			FROM VW_BPM_REPOSICION
	`

	arguments := make([]any, 0, 2)

	if state != "" {
		query += `
			WHERE ESTADO = :1
		`

		arguments = append(
			arguments,
			state,
		)
	}

	query += `
			ORDER BY
				FECHA_CREACION DESC,
				ID_SOLICITUD DESC
		)
	`

	if state != "" {
		query += `
			WHERE ROWNUM <= :2
		`
	} else {
		query += `
			WHERE ROWNUM <= :1
		`
	}

	arguments = append(
		arguments,
		limit,
	)

	rows, err := repository.db.QueryContext(
		ctx,
		query,
		arguments...,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"no se pudieron consultar las reposiciones: %w",
			err,
		)
	}
	defer rows.Close()

	replenishments := make(
		[]models.Replenishment,
		0,
	)

	for rows.Next() {
		replenishment, err :=
			scanReplenishment(rows)

		if err != nil {
			return nil, err
		}

		replenishments = append(
			replenishments,
			replenishment,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"error recorriendo reposiciones: %w",
			err,
		)
	}

	return replenishments, nil
}

// GetReplenishment devuelve el detalle y su historial.
func (
	repository *BPMRepository,
) GetReplenishment(
	ctx context.Context,
	number string,
) (models.ReplenishmentDetail, error) {
	row := repository.db.QueryRowContext(
		ctx,
		`
			SELECT
				ID_SOLICITUD,
				NUMERO_SOLICITUD,
				CODIGO_PRODUCTO,
				PRODUCTO,
				UNIDAD_MEDIDA,
				ID_PROVEEDOR,
				RUC_PROVEEDOR,
				PROVEEDOR,
				ID_ALERTA,
				TIPO_ALERTA,
				ESTADO_ALERTA,
				CANTIDAD_SOLICITADA,
				CANTIDAD_RECIBIDA,
				COSTO_UNITARIO_ESTIMADO,
				COSTO_TOTAL_ESTIMADO,
				ESTADO,
				STOCK_ACTUAL,
				STOCK_RESERVADO,
				STOCK_DISPONIBLE,
				STOCK_MINIMO,
				ID_USUARIO_SOLICITANTE,
				USUARIO_SOLICITANTE,
				ID_USUARIO_APROBADOR,
				USUARIO_APROBADOR,
				ID_USUARIO_RECEPTOR,
				USUARIO_RECEPTOR,
				TO_CHAR(
					FECHA_SOLICITUD,
					'YYYY-MM-DD"T"HH24:MI:SS'
				),
				TO_CHAR(
					FECHA_APROBACION,
					'YYYY-MM-DD"T"HH24:MI:SS'
				),
				TO_CHAR(
					FECHA_RECHAZO,
					'YYYY-MM-DD"T"HH24:MI:SS'
				),
				TO_CHAR(
					FECHA_PEDIDO,
					'YYYY-MM-DD"T"HH24:MI:SS'
				),
				TO_CHAR(
					FECHA_RECEPCION,
					'YYYY-MM-DD"T"HH24:MI:SS'
				),
				TO_CHAR(
					FECHA_CIERRE,
					'YYYY-MM-DD"T"HH24:MI:SS'
				),
				OBSERVACION,
				MOTIVO_RECHAZO,
				TO_CHAR(
					FECHA_CREACION,
					'YYYY-MM-DD"T"HH24:MI:SS'
				),
				TO_CHAR(
					FECHA_ACTUALIZACION,
					'YYYY-MM-DD"T"HH24:MI:SS'
				)
			FROM VW_BPM_REPOSICION
			WHERE NUMERO_SOLICITUD = :1
		`,
		number,
	)

	replenishment, err :=
		scanReplenishment(row)

	if errors.Is(err, sql.ErrNoRows) {
		return models.ReplenishmentDetail{},
			ErrBPMRequestNotFound
	}

	if err != nil {
		return models.ReplenishmentDetail{},
			err
	}

	history, err := repository.listHistory(
		ctx,
		replenishment.IDSolicitud,
	)
	if err != nil {
		return models.ReplenishmentDetail{},
			err
	}

	return models.ReplenishmentDetail{
		Replenishment: replenishment,
		Historial:     history,
	}, nil
}

// CreateDraft crea un proceso en estado BORRADOR.
func (repository *BPMRepository) CreateDraft(
	ctx context.Context,
	request models.CreateReplenishmentRequest,
	userID int64,
	ipAddress string,
) (models.ReplenishmentDetail, error) {
	tx, err := repository.db.BeginTx(ctx, nil)
	if err != nil {
		return models.ReplenishmentDetail{},
			fmt.Errorf(
				"no se pudo iniciar la transacción BPM: %w",
				err,
			)
	}

	committed := false

	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	var productID int64

	err = tx.QueryRowContext(
		ctx,
		`
			SELECT p.ID_PRODUCTO
			FROM PRODUCTO p
			INNER JOIN INVENTARIO i
				ON i.ID_PRODUCTO = p.ID_PRODUCTO
			WHERE p.CODIGO = :1
			  AND p.ESTADO = 'A'
		`,
		request.CodigoProducto,
	).Scan(&productID)

	if errors.Is(err, sql.ErrNoRows) {
		return models.ReplenishmentDetail{},
			ErrBPMProductNotFound
	}

	if err != nil {
		return models.ReplenishmentDetail{},
			fmt.Errorf(
				"no se pudo consultar el producto: %w",
				err,
			)
	}

	var providerCount int

	err = tx.QueryRowContext(
		ctx,
		`
			SELECT COUNT(*)
			FROM PROVEEDOR
			WHERE ID_PROVEEDOR = :1
			  AND ESTADO = 'A'
		`,
		request.IDProveedor,
	).Scan(&providerCount)

	if err != nil {
		return models.ReplenishmentDetail{},
			fmt.Errorf(
				"no se pudo validar el proveedor: %w",
				err,
			)
	}

	if providerCount != 1 {
		return models.ReplenishmentDetail{},
			ErrBPMProviderNotFound
	}

	if request.IDAlerta != nil {
		var alertCount int

		err = tx.QueryRowContext(
			ctx,
			`
				SELECT COUNT(*)
				FROM ALERTA_STOCK
				WHERE ID_ALERTA = :1
				  AND ID_PRODUCTO = :2
				  AND ESTADO = 'PENDIENTE'
			`,
			*request.IDAlerta,
			productID,
		).Scan(&alertCount)

		if err != nil {
			return models.ReplenishmentDetail{},
				fmt.Errorf(
					"no se pudo validar la alerta: %w",
					err,
				)
		}

		if alertCount != 1 {
			return models.ReplenishmentDetail{},
				ErrBPMAlertNotFound
		}
	}

	var activeCount int

	err = tx.QueryRowContext(
		ctx,
		`
			SELECT COUNT(*)
			FROM SOLICITUD_REPOSICION
			WHERE ID_PRODUCTO = :1
			  AND ESTADO IN (
				  'BORRADOR',
				  'SOLICITADA',
				  'APROBADA',
				  'EN_PEDIDO',
				  'RECIBIDA'
			  )
		`,
		productID,
	).Scan(&activeCount)

	if err != nil {
		return models.ReplenishmentDetail{},
			fmt.Errorf(
				"no se pudo validar el proceso activo: %w",
				err,
			)
	}

	if activeCount > 0 {
		return models.ReplenishmentDetail{},
			ErrBPMActiveRequestExists
	}

	var requestNumber string

	err = tx.QueryRowContext(
		ctx,
		`
			SELECT
				'REP-'
				|| TO_CHAR(
					CAST(
						SYSTIMESTAMP
						AT TIME ZONE '-05:00'
						AS TIMESTAMP
					),
					'YYYY'
				)
				|| '-'
				|| LPAD(
					SEQ_NUM_REPOSICION.NEXTVAL,
					6,
					'0'
				)
			FROM DUAL
		`,
	).Scan(&requestNumber)

	if err != nil {
		return models.ReplenishmentDetail{},
			fmt.Errorf(
				"no se pudo generar el número BPM: %w",
				err,
			)
	}

	var alertValue any

	if request.IDAlerta != nil {
		alertValue = *request.IDAlerta
	}

	_, err = tx.ExecContext(
		ctx,
		`
			INSERT INTO SOLICITUD_REPOSICION (
				NUMERO_SOLICITUD,
				ID_PRODUCTO,
				ID_PROVEEDOR,
				ID_ALERTA,
				CANTIDAD_SOLICITADA,
				CANTIDAD_RECIBIDA,
				COSTO_UNITARIO_ESTIMADO,
				ESTADO,
				ID_USUARIO_SOLICITANTE,
				OBSERVACION,
				FECHA_CREACION,
				FECHA_ACTUALIZACION
			)
			VALUES (
				:1,
				:2,
				:3,
				:4,
				:5,
				0,
				:6,
				'BORRADOR',
				:7,
				:8,
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
		`,
		requestNumber,
		productID,
		request.IDProveedor,
		alertValue,
		request.CantidadSolicitada.InexactFloat64(),
		request.CostoUnitarioEstimado.InexactFloat64(),
		userID,
		nullableText(request.Observacion),
	)
	if err != nil {
		if strings.Contains(
			err.Error(),
			"UQ_REP_PRODUCTO_ACTIVA",
		) || strings.Contains(
			err.Error(),
			"ORA-00001",
		) {
			return models.ReplenishmentDetail{},
				ErrBPMActiveRequestExists
		}

		return models.ReplenishmentDetail{},
			fmt.Errorf(
				"no se pudo crear la reposición: %w",
				err,
			)
	}

	var requestID int64

	err = tx.QueryRowContext(
		ctx,
		`
			SELECT ID_SOLICITUD
			FROM SOLICITUD_REPOSICION
			WHERE NUMERO_SOLICITUD = :1
		`,
		requestNumber,
	).Scan(&requestID)

	if err != nil {
		return models.ReplenishmentDetail{},
			fmt.Errorf(
				"no se pudo recuperar la reposición: %w",
				err,
			)
	}

	if err := insertBPMHistory(
		ctx,
		tx,
		requestID,
		userID,
		"",
		"BORRADOR",
		"CREAR",
		request.Observacion,
	); err != nil {
		return models.ReplenishmentDetail{},
			err
	}

	if err := insertBPMAudit(
		ctx,
		tx,
		userID,
		requestID,
		"INSERT",
		"BORRADOR",
		requestNumber,
		ipAddress,
	); err != nil {
		return models.ReplenishmentDetail{},
			err
	}

	if err := tx.Commit(); err != nil {
		return models.ReplenishmentDetail{},
			fmt.Errorf(
				"no se pudo confirmar la reposición: %w",
				err,
			)
	}

	committed = true

	return repository.GetReplenishment(
		ctx,
		requestNumber,
	)
}

// Send cambia BORRADOR a SOLICITADA.
func (repository *BPMRepository) Send(
	ctx context.Context,
	number string,
	userID int64,
	observation string,
	ipAddress string,
) (models.ReplenishmentDetail, error) {
	tx, err := repository.db.BeginTx(ctx, nil)
	if err != nil {
		return models.ReplenishmentDetail{},
			fmt.Errorf(
				"no se pudo iniciar la transición BPM: %w",
				err,
			)
	}

	committed := false

	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	var (
		requestID int64
		state     string
		alertID   sql.NullInt64
	)

	err = tx.QueryRowContext(
		ctx,
		`
			SELECT
				ID_SOLICITUD,
				ESTADO,
				ID_ALERTA
			FROM SOLICITUD_REPOSICION
			WHERE NUMERO_SOLICITUD = :1
			FOR UPDATE
		`,
		number,
	).Scan(
		&requestID,
		&state,
		&alertID,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return models.ReplenishmentDetail{},
			ErrBPMRequestNotFound
	}

	if err != nil {
		return models.ReplenishmentDetail{},
			fmt.Errorf(
				"no se pudo bloquear la solicitud: %w",
				err,
			)
	}

	if state != "BORRADOR" {
		return models.ReplenishmentDetail{},
			ErrBPMInvalidTransition
	}

	_, err = tx.ExecContext(
		ctx,
		`
			UPDATE SOLICITUD_REPOSICION
			SET
				ESTADO = 'SOLICITADA',
				FECHA_SOLICITUD =
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
			WHERE ID_SOLICITUD = :1
		`,
		requestID,
	)
	if err != nil {
		return models.ReplenishmentDetail{},
			fmt.Errorf(
				"no se pudo enviar la solicitud: %w",
				err,
			)
	}

	if alertID.Valid {
		result, err := tx.ExecContext(
			ctx,
			`
				UPDATE ALERTA_STOCK
				SET
					ESTADO = 'ATENDIDA',
					ID_USUARIO_ATENCION = :1,
					FECHA_ATENCION =
						CAST(
							SYSTIMESTAMP
							AT TIME ZONE '-05:00'
							AS TIMESTAMP
						),
					OBSERVACION_ATENCION =
						'Solicitud de reposición '
						|| :2
						|| ' generada.',
					FECHA_ACTUALIZACION =
						CAST(
							SYSTIMESTAMP
							AT TIME ZONE '-05:00'
							AS TIMESTAMP
						)
				WHERE ID_ALERTA = :3
				  AND ESTADO = 'PENDIENTE'
			`,
			userID,
			number,
			alertID.Int64,
		)
		if err != nil {
			return models.ReplenishmentDetail{},
				fmt.Errorf(
					"no se pudo atender la alerta: %w",
					err,
				)
		}

		rows, err := result.RowsAffected()
		if err != nil {
			return models.ReplenishmentDetail{},
				fmt.Errorf(
					"no se pudo verificar la alerta: %w",
					err,
				)
		}

		if rows != 1 {
			return models.ReplenishmentDetail{},
				ErrBPMAlertUnavailable
		}
	}

	if err := insertBPMHistory(
		ctx,
		tx,
		requestID,
		userID,
		"BORRADOR",
		"SOLICITADA",
		"ENVIAR",
		observation,
	); err != nil {
		return models.ReplenishmentDetail{},
			err
	}

	if err := insertBPMAudit(
		ctx,
		tx,
		userID,
		requestID,
		"UPDATE",
		"SOLICITADA",
		number,
		ipAddress,
	); err != nil {
		return models.ReplenishmentDetail{},
			err
	}

	if err := tx.Commit(); err != nil {
		return models.ReplenishmentDetail{},
			fmt.Errorf(
				"no se pudo confirmar el envío: %w",
				err,
			)
	}

	committed = true

	return repository.GetReplenishment(
		ctx,
		number,
	)
}

func (repository *BPMRepository) listHistory(
	ctx context.Context,
	requestID int64,
) ([]models.ReplenishmentHistory, error) {
	rows, err := repository.db.QueryContext(
		ctx,
		`
			SELECT
				h.ID_HISTORIAL,
				h.ID_USUARIO,
				u.NOMBRE_USUARIO,
				h.ESTADO_ANTERIOR,
				h.ESTADO_NUEVO,
				h.ACCION,
				h.OBSERVACION,
				TO_CHAR(
					h.FECHA_EVENTO,
					'YYYY-MM-DD"T"HH24:MI:SS'
				)
			FROM HISTORIAL_REPOSICION h
			INNER JOIN USUARIO u
				ON u.ID_USUARIO = h.ID_USUARIO
			WHERE h.ID_SOLICITUD = :1
			ORDER BY
				h.FECHA_EVENTO,
				h.ID_HISTORIAL
		`,
		requestID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"no se pudo consultar el historial BPM: %w",
			err,
		)
	}
	defer rows.Close()

	history := make(
		[]models.ReplenishmentHistory,
		0,
	)

	for rows.Next() {
		var (
			item        models.ReplenishmentHistory
			previous    sql.NullString
			observation sql.NullString
		)

		if err := rows.Scan(
			&item.IDHistorial,
			&item.IDUsuario,
			&item.Usuario,
			&previous,
			&item.EstadoNuevo,
			&item.Accion,
			&observation,
			&item.FechaEvento,
		); err != nil {
			return nil, fmt.Errorf(
				"no se pudo leer el historial BPM: %w",
				err,
			)
		}

		if previous.Valid {
			item.EstadoAnterior =
				previous.String
		}

		if observation.Valid {
			item.Observacion =
				observation.String
		}

		history = append(
			history,
			item,
		)
	}

	return history, rows.Err()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanReplenishment(
	scanner rowScanner,
) (models.Replenishment, error) {
	var (
		item models.Replenishment

		alertID    sql.NullInt64
		alertType  sql.NullString
		alertState sql.NullString

		approverID sql.NullInt64
		approver   sql.NullString
		receiverID sql.NullInt64
		receiver   sql.NullString

		requestDate   sql.NullString
		approvalDate  sql.NullString
		rejectionDate sql.NullString
		orderDate     sql.NullString
		receptionDate sql.NullString
		closeDate     sql.NullString

		observation sql.NullString
		rejection   sql.NullString

		quantityRequested float64
		quantityReceived  float64
		unitCost          float64
		totalCost         float64
		currentStock      float64
		reservedStock     float64
		availableStock    float64
		minimumStock      float64
	)

	err := scanner.Scan(
		&item.IDSolicitud,
		&item.NumeroSolicitud,
		&item.CodigoProducto,
		&item.Producto,
		&item.UnidadMedida,
		&item.IDProveedor,
		&item.RUCProveedor,
		&item.Proveedor,
		&alertID,
		&alertType,
		&alertState,
		&quantityRequested,
		&quantityReceived,
		&unitCost,
		&totalCost,
		&item.Estado,
		&currentStock,
		&reservedStock,
		&availableStock,
		&minimumStock,
		&item.IDUsuarioSolicitante,
		&item.UsuarioSolicitante,
		&approverID,
		&approver,
		&receiverID,
		&receiver,
		&requestDate,
		&approvalDate,
		&rejectionDate,
		&orderDate,
		&receptionDate,
		&closeDate,
		&observation,
		&rejection,
		&item.FechaCreacion,
		&item.FechaActualizacion,
	)
	if err != nil {
		return models.Replenishment{}, err
	}

	item.CantidadSolicitada =
		decimal.NewFromFloat(
			quantityRequested,
		).StringFixed(3)

	item.CantidadRecibida =
		decimal.NewFromFloat(
			quantityReceived,
		).StringFixed(3)

	item.CostoUnitarioEstimado =
		decimal.NewFromFloat(
			unitCost,
		).StringFixed(2)

	item.CostoTotalEstimado =
		decimal.NewFromFloat(
			totalCost,
		).StringFixed(2)

	item.StockActual =
		decimal.NewFromFloat(
			currentStock,
		).StringFixed(3)

	item.StockReservado =
		decimal.NewFromFloat(
			reservedStock,
		).StringFixed(3)

	item.StockDisponible =
		decimal.NewFromFloat(
			availableStock,
		).StringFixed(3)

	item.StockMinimo =
		decimal.NewFromFloat(
			minimumStock,
		).StringFixed(3)

	if alertID.Valid {
		value := alertID.Int64
		item.IDAlerta = &value
	}

	if alertType.Valid {
		item.TipoAlerta = alertType.String
	}

	if alertState.Valid {
		item.EstadoAlerta = alertState.String
	}

	if approverID.Valid {
		value := approverID.Int64
		item.IDUsuarioAprobador = &value
	}

	if approver.Valid {
		item.UsuarioAprobador =
			approver.String
	}

	if receiverID.Valid {
		value := receiverID.Int64
		item.IDUsuarioReceptor = &value
	}

	if receiver.Valid {
		item.UsuarioReceptor =
			receiver.String
	}

	if requestDate.Valid {
		item.FechaSolicitud =
			requestDate.String
	}

	if approvalDate.Valid {
		item.FechaAprobacion =
			approvalDate.String
	}

	if rejectionDate.Valid {
		item.FechaRechazo =
			rejectionDate.String
	}

	if orderDate.Valid {
		item.FechaPedido =
			orderDate.String
	}

	if receptionDate.Valid {
		item.FechaRecepcion =
			receptionDate.String
	}

	if closeDate.Valid {
		item.FechaCierre =
			closeDate.String
	}

	if observation.Valid {
		item.Observacion =
			observation.String
	}

	if rejection.Valid {
		item.MotivoRechazo =
			rejection.String
	}

	return item, nil
}

func insertBPMHistory(
	ctx context.Context,
	tx *sql.Tx,
	requestID int64,
	userID int64,
	previousState string,
	newState string,
	action string,
	observation string,
) error {
	var previousValue any

	if previousState != "" {
		previousValue = previousState
	}

	_, err := tx.ExecContext(
		ctx,
		`
			INSERT INTO HISTORIAL_REPOSICION (
				ID_SOLICITUD,
				ID_USUARIO,
				ESTADO_ANTERIOR,
				ESTADO_NUEVO,
				ACCION,
				OBSERVACION,
				FECHA_EVENTO
			)
			VALUES (
				:1,
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
		requestID,
		userID,
		previousValue,
		newState,
		action,
		nullableText(observation),
	)
	if err != nil {
		return fmt.Errorf(
			"no se pudo registrar el historial BPM: %w",
			err,
		)
	}

	return nil
}

func insertBPMAudit(
	ctx context.Context,
	tx *sql.Tx,
	userID int64,
	requestID int64,
	operation string,
	state string,
	number string,
	ipAddress string,
) error {
	values, _ := json.Marshal(
		map[string]string{
			"numero_solicitud": number,
			"estado":           state,
		},
	)

	_, err := tx.ExecContext(
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
				'SOLICITUD_REPOSICION',
				:2,
				:3,
				:4,
				:5,
				'API REST BPM',
				CAST(
					SYSTIMESTAMP
					AT TIME ZONE '-05:00'
					AS TIMESTAMP
				)
			)
		`,
		userID,
		operation,
		fmt.Sprintf("%d", requestID),
		string(values),
		nullableText(ipAddress),
	)
	if err != nil {
		return fmt.Errorf(
			"no se pudo auditar el proceso BPM: %w",
			err,
		)
	}

	return nil
}

// ErrBPMInvalidReceivedQuantity indica que la cantidad
// recibida no corresponde a la solicitud.
var ErrBPMInvalidReceivedQuantity = errors.New(
	"cantidad recibida inválida",
)

// ErrBPMInventoryNotFound indica que el producto no tiene
// un registro de inventario disponible.
var ErrBPMInventoryNotFound = errors.New(
	"inventario no encontrado",
)

// Approve cambia SOLICITADA a APROBADA.
func (repository *BPMRepository) Approve(
	ctx context.Context,
	number string,
	userID int64,
	observation string,
	ipAddress string,
) (models.ReplenishmentDetail, error) {
	tx, err := repository.db.BeginTx(ctx, nil)
	if err != nil {
		return models.ReplenishmentDetail{},
			fmt.Errorf(
				"no se pudo iniciar la aprobación BPM: %w",
				err,
			)
	}

	committed := false

	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	requestID, state, err :=
		lockBPMRequestState(
			ctx,
			tx,
			number,
		)

	if err != nil {
		return models.ReplenishmentDetail{}, err
	}

	if state != "SOLICITADA" {
		return models.ReplenishmentDetail{},
			ErrBPMInvalidTransition
	}

	_, err = tx.ExecContext(
		ctx,
		`
			UPDATE SOLICITUD_REPOSICION
			SET
				ESTADO = 'APROBADA',
				ID_USUARIO_APROBADOR = :1,
				FECHA_APROBACION =
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
			WHERE ID_SOLICITUD = :2
		`,
		userID,
		requestID,
	)
	if err != nil {
		return models.ReplenishmentDetail{},
			fmt.Errorf(
				"no se pudo aprobar la reposición: %w",
				err,
			)
	}

	if err := insertBPMHistory(
		ctx,
		tx,
		requestID,
		userID,
		"SOLICITADA",
		"APROBADA",
		"APROBAR",
		observation,
	); err != nil {
		return models.ReplenishmentDetail{}, err
	}

	if err := insertBPMAudit(
		ctx,
		tx,
		userID,
		requestID,
		"UPDATE",
		"APROBADA",
		number,
		ipAddress,
	); err != nil {
		return models.ReplenishmentDetail{}, err
	}

	if err := tx.Commit(); err != nil {
		return models.ReplenishmentDetail{},
			fmt.Errorf(
				"no se pudo confirmar la aprobación: %w",
				err,
			)
	}

	committed = true

	return repository.GetReplenishment(
		ctx,
		number,
	)
}

// Reject cambia SOLICITADA a RECHAZADA.
func (repository *BPMRepository) Reject(
	ctx context.Context,
	number string,
	userID int64,
	reason string,
	ipAddress string,
) (models.ReplenishmentDetail, error) {
	tx, err := repository.db.BeginTx(ctx, nil)
	if err != nil {
		return models.ReplenishmentDetail{},
			fmt.Errorf(
				"no se pudo iniciar el rechazo BPM: %w",
				err,
			)
	}

	committed := false

	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	requestID, state, err :=
		lockBPMRequestState(
			ctx,
			tx,
			number,
		)

	if err != nil {
		return models.ReplenishmentDetail{}, err
	}

	if state != "SOLICITADA" {
		return models.ReplenishmentDetail{},
			ErrBPMInvalidTransition
	}

	_, err = tx.ExecContext(
		ctx,
		`
			UPDATE SOLICITUD_REPOSICION
			SET
				ESTADO = 'RECHAZADA',
				ID_USUARIO_APROBADOR = :1,
				MOTIVO_RECHAZO = :2,
				FECHA_RECHAZO =
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
			WHERE ID_SOLICITUD = :3
		`,
		userID,
		reason,
		requestID,
	)
	if err != nil {
		return models.ReplenishmentDetail{},
			fmt.Errorf(
				"no se pudo rechazar la reposición: %w",
				err,
			)
	}

	if err := insertBPMHistory(
		ctx,
		tx,
		requestID,
		userID,
		"SOLICITADA",
		"RECHAZADA",
		"RECHAZAR",
		reason,
	); err != nil {
		return models.ReplenishmentDetail{}, err
	}

	if err := insertBPMAudit(
		ctx,
		tx,
		userID,
		requestID,
		"UPDATE",
		"RECHAZADA",
		number,
		ipAddress,
	); err != nil {
		return models.ReplenishmentDetail{}, err
	}

	if err := tx.Commit(); err != nil {
		return models.ReplenishmentDetail{},
			fmt.Errorf(
				"no se pudo confirmar el rechazo: %w",
				err,
			)
	}

	committed = true

	return repository.GetReplenishment(
		ctx,
		number,
	)
}

// MarkOrder cambia APROBADA a EN_PEDIDO.
func (repository *BPMRepository) MarkOrder(
	ctx context.Context,
	number string,
	userID int64,
	observation string,
	ipAddress string,
) (models.ReplenishmentDetail, error) {
	tx, err := repository.db.BeginTx(ctx, nil)
	if err != nil {
		return models.ReplenishmentDetail{},
			fmt.Errorf(
				"no se pudo iniciar el pedido BPM: %w",
				err,
			)
	}

	committed := false

	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	requestID, state, err :=
		lockBPMRequestState(
			ctx,
			tx,
			number,
		)

	if err != nil {
		return models.ReplenishmentDetail{}, err
	}

	if state != "APROBADA" {
		return models.ReplenishmentDetail{},
			ErrBPMInvalidTransition
	}

	_, err = tx.ExecContext(
		ctx,
		`
			UPDATE SOLICITUD_REPOSICION
			SET
				ESTADO = 'EN_PEDIDO',
				FECHA_PEDIDO =
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
			WHERE ID_SOLICITUD = :1
		`,
		requestID,
	)
	if err != nil {
		return models.ReplenishmentDetail{},
			fmt.Errorf(
				"no se pudo registrar el pedido: %w",
				err,
			)
	}

	if err := insertBPMHistory(
		ctx,
		tx,
		requestID,
		userID,
		"APROBADA",
		"EN_PEDIDO",
		"MARCAR_PEDIDO",
		observation,
	); err != nil {
		return models.ReplenishmentDetail{}, err
	}

	if err := insertBPMAudit(
		ctx,
		tx,
		userID,
		requestID,
		"UPDATE",
		"EN_PEDIDO",
		number,
		ipAddress,
	); err != nil {
		return models.ReplenishmentDetail{}, err
	}

	if err := tx.Commit(); err != nil {
		return models.ReplenishmentDetail{},
			fmt.Errorf(
				"no se pudo confirmar el pedido: %w",
				err,
			)
	}

	committed = true

	return repository.GetReplenishment(
		ctx,
		number,
	)
}

// Receive registra la recepción y actualiza inventario
// dentro de una sola transacción.
func (repository *BPMRepository) Receive(
	ctx context.Context,
	number string,
	userID int64,
	quantity decimal.Decimal,
	observation string,
	ipAddress string,
) (models.ReplenishmentDetail, error) {
	tx, err := repository.db.BeginTx(ctx, nil)
	if err != nil {
		return models.ReplenishmentDetail{},
			fmt.Errorf(
				"no se pudo iniciar la recepción BPM: %w",
				err,
			)
	}

	committed := false

	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	var (
		requestID      int64
		productID      int64
		state          string
		requestedFloat float64
	)

	err = tx.QueryRowContext(
		ctx,
		`
			SELECT
				ID_SOLICITUD,
				ID_PRODUCTO,
				ESTADO,
				CANTIDAD_SOLICITADA
			FROM SOLICITUD_REPOSICION
			WHERE NUMERO_SOLICITUD = :1
			FOR UPDATE
		`,
		number,
	).Scan(
		&requestID,
		&productID,
		&state,
		&requestedFloat,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return models.ReplenishmentDetail{},
			ErrBPMRequestNotFound
	}

	if err != nil {
		return models.ReplenishmentDetail{},
			fmt.Errorf(
				"no se pudo bloquear la reposición: %w",
				err,
			)
	}

	if state != "EN_PEDIDO" {
		return models.ReplenishmentDetail{},
			ErrBPMInvalidTransition
	}

	requestedQuantity :=
		decimal.NewFromFloat(
			requestedFloat,
		)

	if !quantity.GreaterThan(decimal.Zero) ||
		quantity.GreaterThan(
			requestedQuantity,
		) {
		return models.ReplenishmentDetail{},
			ErrBPMInvalidReceivedQuantity
	}

	var (
		currentFloat  float64
		reservedFloat float64
		minimumFloat  float64
	)

	err = tx.QueryRowContext(
		ctx,
		`
			SELECT
				i.STOCK_ACTUAL,
				i.STOCK_RESERVADO,
				p.STOCK_MINIMO
			FROM INVENTARIO i
			INNER JOIN PRODUCTO p
				ON p.ID_PRODUCTO = i.ID_PRODUCTO
			WHERE i.ID_PRODUCTO = :1
			FOR UPDATE OF i.STOCK_ACTUAL
		`,
		productID,
	).Scan(
		&currentFloat,
		&reservedFloat,
		&minimumFloat,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return models.ReplenishmentDetail{},
			ErrBPMInventoryNotFound
	}

	if err != nil {
		return models.ReplenishmentDetail{},
			fmt.Errorf(
				"no se pudo bloquear el inventario: %w",
				err,
			)
	}

	currentStock :=
		decimal.NewFromFloat(currentFloat)

	reservedStock :=
		decimal.NewFromFloat(reservedFloat)

	minimumStock :=
		decimal.NewFromFloat(minimumFloat)

	newCurrentStock :=
		currentStock.Add(quantity)

	newAvailableStock :=
		newCurrentStock.Sub(reservedStock)

	result, err := tx.ExecContext(
		ctx,
		`
			UPDATE INVENTARIO
			SET
				STOCK_ACTUAL = :1,
				FECHA_ULTIMO_MOVIMIENTO =
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
			WHERE ID_PRODUCTO = :2
		`,
		newCurrentStock.InexactFloat64(),
		productID,
	)
	if err != nil {
		return models.ReplenishmentDetail{},
			fmt.Errorf(
				"no se pudo actualizar el inventario: %w",
				err,
			)
	}

	affectedRows, err := result.RowsAffected()
	if err != nil {
		return models.ReplenishmentDetail{},
			fmt.Errorf(
				"no se pudo verificar el inventario: %w",
				err,
			)
	}

	if affectedRows != 1 {
		return models.ReplenishmentDetail{},
			ErrBPMInventoryNotFound
	}

	movementReason :=
		"Entrada por reposición " + number

	if observation != "" {
		movementReason += ": " + observation
	}

	if len([]rune(movementReason)) > 300 {
		movementReason = string(
			[]rune(movementReason)[:300],
		)
	}

	_, err = tx.ExecContext(
		ctx,
		`
			INSERT INTO MOVIMIENTO_INVENTARIO (
				ID_PRODUCTO,
				ID_USUARIO,
				TIPO_MOVIMIENTO,
				CANTIDAD,
				STOCK_ANTERIOR,
				STOCK_NUEVO,
				MOTIVO,
				ID_SOLICITUD_REPOSICION,
				FECHA_MOVIMIENTO
			)
			VALUES (
				:1,
				:2,
				'ENTRADA_COMPRA',
				:3,
				:4,
				:5,
				:6,
				:7,
				CAST(
					SYSTIMESTAMP
					AT TIME ZONE '-05:00'
					AS TIMESTAMP
				)
			)
		`,
		productID,
		userID,
		quantity.InexactFloat64(),
		currentStock.InexactFloat64(),
		newCurrentStock.InexactFloat64(),
		movementReason,
		requestID,
	)
	if err != nil {
		return models.ReplenishmentDetail{},
			fmt.Errorf(
				"no se pudo registrar la entrada de inventario: %w",
				err,
			)
	}

	_, err = tx.ExecContext(
		ctx,
		`
			UPDATE SOLICITUD_REPOSICION
			SET
				ESTADO = 'RECIBIDA',
				CANTIDAD_RECIBIDA = :1,
				ID_USUARIO_RECEPTOR = :2,
				FECHA_RECEPCION =
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
			WHERE ID_SOLICITUD = :3
		`,
		quantity.InexactFloat64(),
		userID,
		requestID,
	)
	if err != nil {
		return models.ReplenishmentDetail{},
			fmt.Errorf(
				"no se pudo confirmar la recepción BPM: %w",
				err,
			)
	}

	// Una alerta pendiente solo se cierra cuando
	// el stock disponible supera el mínimo.
	if newAvailableStock.GreaterThan(
		minimumStock,
	) {
		_, err = tx.ExecContext(
			ctx,
			`
				UPDATE ALERTA_STOCK
				SET
					ESTADO = 'ATENDIDA',
					ID_USUARIO_ATENCION = :1,
					FECHA_ATENCION =
						CAST(
							SYSTIMESTAMP
							AT TIME ZONE '-05:00'
							AS TIMESTAMP
						),
					OBSERVACION_ATENCION =
						'Stock normalizado mediante '
						|| :2,
					FECHA_ACTUALIZACION =
						CAST(
							SYSTIMESTAMP
							AT TIME ZONE '-05:00'
							AS TIMESTAMP
						)
				WHERE ID_PRODUCTO = :3
				  AND ESTADO = 'PENDIENTE'
			`,
			userID,
			number,
			productID,
		)
		if err != nil {
			return models.ReplenishmentDetail{},
				fmt.Errorf(
					"no se pudo actualizar la alerta de stock: %w",
					err,
				)
		}
	}

	if err := insertBPMHistory(
		ctx,
		tx,
		requestID,
		userID,
		"EN_PEDIDO",
		"RECIBIDA",
		"RECIBIR",
		observation,
	); err != nil {
		return models.ReplenishmentDetail{}, err
	}

	if err := insertBPMAudit(
		ctx,
		tx,
		userID,
		requestID,
		"UPDATE",
		"RECIBIDA",
		number,
		ipAddress,
	); err != nil {
		return models.ReplenishmentDetail{}, err
	}

	if err := tx.Commit(); err != nil {
		return models.ReplenishmentDetail{},
			fmt.Errorf(
				"no se pudo confirmar la recepción: %w",
				err,
			)
	}

	committed = true

	return repository.GetReplenishment(
		ctx,
		number,
	)
}

// Close cambia RECIBIDA a CERRADA.
func (repository *BPMRepository) Close(
	ctx context.Context,
	number string,
	userID int64,
	observation string,
	ipAddress string,
) (models.ReplenishmentDetail, error) {
	tx, err := repository.db.BeginTx(ctx, nil)
	if err != nil {
		return models.ReplenishmentDetail{},
			fmt.Errorf(
				"no se pudo iniciar el cierre BPM: %w",
				err,
			)
	}

	committed := false

	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	requestID, state, err :=
		lockBPMRequestState(
			ctx,
			tx,
			number,
		)

	if err != nil {
		return models.ReplenishmentDetail{}, err
	}

	if state != "RECIBIDA" {
		return models.ReplenishmentDetail{},
			ErrBPMInvalidTransition
	}

	_, err = tx.ExecContext(
		ctx,
		`
			UPDATE SOLICITUD_REPOSICION
			SET
				ESTADO = 'CERRADA',
				FECHA_CIERRE =
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
			WHERE ID_SOLICITUD = :1
		`,
		requestID,
	)
	if err != nil {
		return models.ReplenishmentDetail{},
			fmt.Errorf(
				"no se pudo cerrar la reposición: %w",
				err,
			)
	}

	if err := insertBPMHistory(
		ctx,
		tx,
		requestID,
		userID,
		"RECIBIDA",
		"CERRADA",
		"CERRAR",
		observation,
	); err != nil {
		return models.ReplenishmentDetail{}, err
	}

	if err := insertBPMAudit(
		ctx,
		tx,
		userID,
		requestID,
		"UPDATE",
		"CERRADA",
		number,
		ipAddress,
	); err != nil {
		return models.ReplenishmentDetail{}, err
	}

	if err := tx.Commit(); err != nil {
		return models.ReplenishmentDetail{},
			fmt.Errorf(
				"no se pudo confirmar el cierre BPM: %w",
				err,
			)
	}

	committed = true

	return repository.GetReplenishment(
		ctx,
		number,
	)
}

func lockBPMRequestState(
	ctx context.Context,
	tx *sql.Tx,
	number string,
) (int64, string, error) {
	var (
		requestID int64
		state     string
	)

	err := tx.QueryRowContext(
		ctx,
		`
			SELECT
				ID_SOLICITUD,
				ESTADO
			FROM SOLICITUD_REPOSICION
			WHERE NUMERO_SOLICITUD = :1
			FOR UPDATE
		`,
		number,
	).Scan(
		&requestID,
		&state,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return 0, "",
			ErrBPMRequestNotFound
	}

	if err != nil {
		return 0, "",
			fmt.Errorf(
				"no se pudo bloquear la solicitud BPM: %w",
				err,
			)
	}

	return requestID, state, nil
}
