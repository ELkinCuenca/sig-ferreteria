package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"sigefer.local/backend/internal/models"
)

// ErrIoTDeviceNotFound indica que el código
// recibido no corresponde a un dispositivo registrado.
var ErrIoTDeviceNotFound = errors.New(
	"dispositivo IoT no encontrado",
)

// ErrIoTConfigurationNotFound indica que el
// dispositivo no tiene configuración registrada.
var ErrIoTConfigurationNotFound = errors.New(
	"configuración IoT no encontrada",
)

// iotAPIRowScanner permite interpretar tanto
// *sql.Row como *sql.Rows.
type iotAPIRowScanner interface {
	Scan(dest ...any) error
}

// GetIoTSummary devuelve el estado actual del dispositivo,
// su configuración, última lectura y totales.
func (repository *IoTRepository) GetIoTSummary(
	ctx context.Context,
	deviceCode string,
) (
	models.IoTSummaryResponse,
	error,
) {
	deviceID, err := repository.findIoTDeviceID(
		ctx,
		deviceCode,
	)
	if err != nil {
		return models.IoTSummaryResponse{}, err
	}

	const query = `
		SELECT
			d.ID_DISPOSITIVO,
			d.CODIGO,
			d.NOMBRE,
			d.UBICACION,
			d.TIPO_SENSOR,
			d.ESTADO,

			CASE
				WHEN d.ULTIMA_COMUNICACION IS NULL
					THEN 'SIN_DATOS'

				WHEN d.ULTIMA_COMUNICACION <
					CAST(
						CURRENT_TIMESTAMP AS TIMESTAMP
					)
					-
					NUMTODSINTERVAL(
						c.SEGUNDOS_SIN_COMUNICACION,
						'SECOND'
					)
					THEN 'SIN_COMUNICACION'

				ELSE 'EN_LINEA'
			END,

			TO_CHAR(
				d.ULTIMA_COMUNICACION,
				'YYYY-MM-DD"T"HH24:MI:SS'
			),

			d.ULTIMA_IP,
			d.ULTIMO_RSSI_DBM,
			d.ULTIMO_ESTADO_SENSOR,

			c.ID_CONFIGURACION,
			c.TEMPERATURA_MIN_C,
			c.TEMPERATURA_MAX_C,
			c.HUMEDAD_MIN_PCT,
			c.HUMEDAD_MAX_PCT,
			c.SEGUNDOS_SIN_COMUNICACION,
			c.ESTADO,

			TO_CHAR(
				c.FECHA_ACTUALIZACION,
				'YYYY-MM-DD"T"HH24:MI:SS'
			),

			(
				SELECT COUNT(*)
				FROM SIGEFER_APP.IOT_LECTURA l
				WHERE l.ID_DISPOSITIVO =
					d.ID_DISPOSITIVO
			),

			(
				SELECT COUNT(*)
				FROM SIGEFER_APP.IOT_ALERTA a
				WHERE a.ID_DISPOSITIVO =
					d.ID_DISPOSITIVO
				  AND a.ESTADO = 'PENDIENTE'
			),

			TO_CHAR(
				SYSTIMESTAMP
					AT TIME ZONE '-05:00',
				'YYYY-MM-DD"T"HH24:MI:SS TZH:TZM'
			)

		FROM SIGEFER_APP.IOT_DISPOSITIVO d

		INNER JOIN SIGEFER_APP.IOT_CONFIGURACION c
			ON c.ID_DISPOSITIVO =
				d.ID_DISPOSITIVO

		WHERE d.ID_DISPOSITIVO = :1
	`

	var (
		summary models.IoTSummaryResponse

		location          sql.NullString
		lastCommunication sql.NullString
		lastIP            sql.NullString
		lastRSSI          sql.NullInt64
		lastSensorState   sql.NullString
	)

	err = repository.db.QueryRowContext(
		ctx,
		query,
		deviceID,
	).Scan(
		&summary.Dispositivo.IDDispositivo,
		&summary.Dispositivo.Codigo,
		&summary.Dispositivo.Nombre,
		&location,
		&summary.Dispositivo.TipoSensor,
		&summary.Dispositivo.Estado,
		&summary.Dispositivo.EstadoComunicacion,
		&lastCommunication,
		&lastIP,
		&lastRSSI,
		&lastSensorState,
		&summary.Configuracion.IDConfiguracion,
		&summary.Configuracion.TemperaturaMinC,
		&summary.Configuracion.TemperaturaMaxC,
		&summary.Configuracion.HumedadMinPct,
		&summary.Configuracion.HumedadMaxPct,
		&summary.Configuracion.SegundosSinComunicacion,
		&summary.Configuracion.Estado,
		&summary.Configuracion.FechaActualizacion,
		&summary.TotalLecturas,
		&summary.AlertasPendientes,
		&summary.FechaGeneracion,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return models.IoTSummaryResponse{},
			ErrIoTConfigurationNotFound
	}

	if err != nil {
		return models.IoTSummaryResponse{},
			fmt.Errorf(
				"no se pudo consultar el resumen IoT: %w",
				err,
			)
	}

	summary.Status = "ok"

	summary.Dispositivo.Ubicacion =
		iotAPIString(location)

	summary.Dispositivo.UltimaComunicacion =
		iotAPIString(lastCommunication)

	summary.Dispositivo.UltimaIP =
		iotAPIString(lastIP)

	summary.Dispositivo.UltimoRSSIDBM =
		iotAPIIntPointer(lastRSSI)

	summary.Dispositivo.UltimoEstadoSensor =
		iotAPIString(lastSensorState)

	summary.Configuracion.IDDispositivo =
		summary.Dispositivo.IDDispositivo

	summary.Configuracion.CodigoDispositivo =
		summary.Dispositivo.Codigo

	latestReading, found, err :=
		repository.findLatestIoTReading(
			ctx,
			summary.Dispositivo.IDDispositivo,
		)

	if err != nil {
		return models.IoTSummaryResponse{}, err
	}

	if found {
		summary.UltimaLectura =
			&latestReading
	}

	return summary, nil
}

// ListIoTReadings devuelve las lecturas más recientes
// de un dispositivo.
func (repository *IoTRepository) ListIoTReadings(
	ctx context.Context,
	deviceCode string,
	limit int,
) (
	[]models.IoTReading,
	error,
) {
	deviceID, err := repository.findIoTDeviceID(
		ctx,
		deviceCode,
	)
	if err != nil {
		return nil, err
	}

	const query = `
		SELECT
			ID_LECTURA,
			CODIGO,
			ID_ARRANQUE,
			SECUENCIA,
			TEMPERATURA_C,
			HUMEDAD_PCT,
			RSSI_DBM,
			ESTADO_SENSOR,
			IP_DISPOSITIVO,
			UPTIME_S,
			LECTURAS_VALIDAS,
			LECTURAS_FALLIDAS,
			ORIGEN,
			FECHA_RECEPCION

		FROM (
			SELECT
				l.ID_LECTURA,
				d.CODIGO,
				l.ID_ARRANQUE,
				l.SECUENCIA,
				l.TEMPERATURA_C,
				l.HUMEDAD_PCT,
				l.RSSI_DBM,
				l.ESTADO_SENSOR,
				l.IP_DISPOSITIVO,
				l.UPTIME_S,
				l.LECTURAS_VALIDAS,
				l.LECTURAS_FALLIDAS,
				l.ORIGEN,

				TO_CHAR(
					l.FECHA_RECEPCION,
					'YYYY-MM-DD"T"HH24:MI:SS'
				) AS FECHA_RECEPCION

			FROM SIGEFER_APP.IOT_LECTURA l

			INNER JOIN SIGEFER_APP.IOT_DISPOSITIVO d
				ON d.ID_DISPOSITIVO =
					l.ID_DISPOSITIVO

			WHERE l.ID_DISPOSITIVO = :1

			ORDER BY l.ID_LECTURA DESC
		)

		WHERE ROWNUM <= :2
	`

	rows, err := repository.db.QueryContext(
		ctx,
		query,
		deviceID,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"no se pudo consultar las lecturas IoT: %w",
			err,
		)
	}

	defer rows.Close()

	readings := make(
		[]models.IoTReading,
		0,
	)

	for rows.Next() {
		reading, err :=
			scanIoTReading(rows)

		if err != nil {
			return nil, fmt.Errorf(
				"no se pudo interpretar una lectura IoT: %w",
				err,
			)
		}

		readings = append(
			readings,
			reading,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"error recorriendo las lecturas IoT: %w",
			err,
		)
	}

	return readings, nil
}

// ListIoTAlerts devuelve las alertas ambientales
// del dispositivo aplicando un filtro opcional de estado.
func (repository *IoTRepository) ListIoTAlerts(
	ctx context.Context,
	deviceCode string,
	status string,
	limit int,
) (
	[]models.IoTAlert,
	error,
) {
	deviceID, err := repository.findIoTDeviceID(
		ctx,
		deviceCode,
	)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT
			ID_ALERTA_IOT,
			CODIGO,
			ID_LECTURA,
			TIPO_ALERTA,
			SEVERIDAD,
			MENSAJE,
			VALOR_DETECTADO,
			VALOR_LIMITE,
			ESTADO,
			FECHA_GENERACION,
			ID_USUARIO_ATENCION,
			FECHA_ATENCION,
			OBSERVACION_ATENCION,
			FECHA_ACTUALIZACION

		FROM (
			SELECT
				a.ID_ALERTA_IOT,
				d.CODIGO,
				a.ID_LECTURA,
				a.TIPO_ALERTA,
				a.SEVERIDAD,
				a.MENSAJE,
				a.VALOR_DETECTADO,
				a.VALOR_LIMITE,
				a.ESTADO,

				TO_CHAR(
					a.FECHA_GENERACION,
					'YYYY-MM-DD"T"HH24:MI:SS'
				) AS FECHA_GENERACION,

				a.ID_USUARIO_ATENCION,

				TO_CHAR(
					a.FECHA_ATENCION,
					'YYYY-MM-DD"T"HH24:MI:SS'
				) AS FECHA_ATENCION,

				a.OBSERVACION_ATENCION,

				TO_CHAR(
					a.FECHA_ACTUALIZACION,
					'YYYY-MM-DD"T"HH24:MI:SS'
				) AS FECHA_ACTUALIZACION

			FROM SIGEFER_APP.IOT_ALERTA a

			INNER JOIN SIGEFER_APP.IOT_DISPOSITIVO d
				ON d.ID_DISPOSITIVO =
					a.ID_DISPOSITIVO

			WHERE a.ID_DISPOSITIVO = :1
	`

	args := []any{
		deviceID,
	}

	if status != "" {
		query += `
			AND a.ESTADO = :2
		`

		args = append(
			args,
			status,
		)
	}

	query += `
			ORDER BY
				CASE
					WHEN a.ESTADO = 'PENDIENTE'
						THEN 1

					WHEN a.ESTADO = 'ATENDIDA'
						THEN 2

					ELSE 3
				END,

				CASE
					WHEN a.SEVERIDAD = 'CRITICA'
						THEN 1

					ELSE 2
				END,

				a.FECHA_GENERACION DESC
		)

		WHERE ROWNUM <= :
	` + fmt.Sprint(
		len(args)+1,
	)

	args = append(
		args,
		limit,
	)

	rows, err := repository.db.QueryContext(
		ctx,
		query,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"no se pudo consultar las alertas IoT: %w",
			err,
		)
	}

	defer rows.Close()

	alerts := make(
		[]models.IoTAlert,
		0,
	)

	for rows.Next() {
		var (
			alert models.IoTAlert

			readingID     sql.NullInt64
			detectedValue sql.NullFloat64
			limitValue    sql.NullFloat64
			userID        sql.NullInt64
			attendedAt    sql.NullString
			observation   sql.NullString
		)

		err := rows.Scan(
			&alert.IDAlertaIoT,
			&alert.CodigoDispositivo,
			&readingID,
			&alert.TipoAlerta,
			&alert.Severidad,
			&alert.Mensaje,
			&detectedValue,
			&limitValue,
			&alert.Estado,
			&alert.FechaGeneracion,
			&userID,
			&attendedAt,
			&observation,
			&alert.FechaActualizacion,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"no se pudo interpretar una alerta IoT: %w",
				err,
			)
		}

		alert.IDLectura =
			iotAPIIntPointer(readingID)

		alert.ValorDetectado =
			iotAPIFloatPointer(
				detectedValue,
			)

		alert.ValorLimite =
			iotAPIFloatPointer(
				limitValue,
			)

		alert.IDUsuarioAtencion =
			iotAPIIntPointer(userID)

		alert.FechaAtencion =
			iotAPIString(attendedAt)

		alert.ObservacionAtencion =
			iotAPIString(observation)

		alerts = append(
			alerts,
			alert,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"error recorriendo las alertas IoT: %w",
			err,
		)
	}

	return alerts, nil
}

// GetIoTConfiguration devuelve los umbrales
// configurados para el dispositivo.
func (repository *IoTRepository) GetIoTConfiguration(
	ctx context.Context,
	deviceCode string,
) (
	models.IoTConfiguration,
	error,
) {
	const query = `
		SELECT
			c.ID_CONFIGURACION,
			c.ID_DISPOSITIVO,
			d.CODIGO,
			c.TEMPERATURA_MIN_C,
			c.TEMPERATURA_MAX_C,
			c.HUMEDAD_MIN_PCT,
			c.HUMEDAD_MAX_PCT,
			c.SEGUNDOS_SIN_COMUNICACION,
			c.ESTADO,

			TO_CHAR(
				c.FECHA_ACTUALIZACION,
				'YYYY-MM-DD"T"HH24:MI:SS'
			)

		FROM SIGEFER_APP.IOT_CONFIGURACION c

		INNER JOIN SIGEFER_APP.IOT_DISPOSITIVO d
			ON d.ID_DISPOSITIVO =
				c.ID_DISPOSITIVO

		WHERE d.CODIGO = :1
	`

	var configuration models.IoTConfiguration

	err := repository.db.QueryRowContext(
		ctx,
		query,
		deviceCode,
	).Scan(
		&configuration.IDConfiguracion,
		&configuration.IDDispositivo,
		&configuration.CodigoDispositivo,
		&configuration.TemperaturaMinC,
		&configuration.TemperaturaMaxC,
		&configuration.HumedadMinPct,
		&configuration.HumedadMaxPct,
		&configuration.SegundosSinComunicacion,
		&configuration.Estado,
		&configuration.FechaActualizacion,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return models.IoTConfiguration{},
			ErrIoTConfigurationNotFound
	}

	if err != nil {
		return models.IoTConfiguration{},
			fmt.Errorf(
				"no se pudo consultar la configuración IoT: %w",
				err,
			)
	}

	return configuration, nil
}

// findIoTDeviceID resuelve el identificador interno
// a partir del código público del dispositivo.
func (repository *IoTRepository) findIoTDeviceID(
	ctx context.Context,
	deviceCode string,
) (
	int64,
	error,
) {
	const query = `
		SELECT ID_DISPOSITIVO

		FROM SIGEFER_APP.IOT_DISPOSITIVO

		WHERE CODIGO = :1
	`

	var deviceID int64

	err := repository.db.QueryRowContext(
		ctx,
		query,
		deviceCode,
	).Scan(
		&deviceID,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrIoTDeviceNotFound
	}

	if err != nil {
		return 0, fmt.Errorf(
			"no se pudo consultar el dispositivo IoT: %w",
			err,
		)
	}

	return deviceID, nil
}

// findLatestIoTReading devuelve la lectura más reciente.
// found será false cuando el dispositivo todavía no
// tenga información almacenada.
func (repository *IoTRepository) findLatestIoTReading(
	ctx context.Context,
	deviceID int64,
) (
	models.IoTReading,
	bool,
	error,
) {
	const query = `
		SELECT
			ID_LECTURA,
			CODIGO,
			ID_ARRANQUE,
			SECUENCIA,
			TEMPERATURA_C,
			HUMEDAD_PCT,
			RSSI_DBM,
			ESTADO_SENSOR,
			IP_DISPOSITIVO,
			UPTIME_S,
			LECTURAS_VALIDAS,
			LECTURAS_FALLIDAS,
			ORIGEN,
			FECHA_RECEPCION

		FROM (
			SELECT
				l.ID_LECTURA,
				d.CODIGO,
				l.ID_ARRANQUE,
				l.SECUENCIA,
				l.TEMPERATURA_C,
				l.HUMEDAD_PCT,
				l.RSSI_DBM,
				l.ESTADO_SENSOR,
				l.IP_DISPOSITIVO,
				l.UPTIME_S,
				l.LECTURAS_VALIDAS,
				l.LECTURAS_FALLIDAS,
				l.ORIGEN,

				TO_CHAR(
					l.FECHA_RECEPCION,
					'YYYY-MM-DD"T"HH24:MI:SS'
				) AS FECHA_RECEPCION

			FROM SIGEFER_APP.IOT_LECTURA l

			INNER JOIN SIGEFER_APP.IOT_DISPOSITIVO d
				ON d.ID_DISPOSITIVO =
					l.ID_DISPOSITIVO

			WHERE l.ID_DISPOSITIVO = :1

			ORDER BY l.ID_LECTURA DESC
		)

		WHERE ROWNUM = 1
	`

	reading, err := scanIoTReading(
		repository.db.QueryRowContext(
			ctx,
			query,
			deviceID,
		),
	)

	if errors.Is(err, sql.ErrNoRows) {
		return models.IoTReading{},
			false,
			nil
	}

	if err != nil {
		return models.IoTReading{},
			false,
			fmt.Errorf(
				"no se pudo consultar la última lectura IoT: %w",
				err,
			)
	}

	return reading, true, nil
}

// scanIoTReading interpreta una lectura y convierte
// correctamente las columnas opcionales de Oracle.
func scanIoTReading(
	scanner iotAPIRowScanner,
) (
	models.IoTReading,
	error,
) {
	var (
		reading models.IoTReading

		temperature sql.NullFloat64
		humidity    sql.NullFloat64
		rssi        sql.NullInt64
		deviceIP    sql.NullString
	)

	err := scanner.Scan(
		&reading.IDLectura,
		&reading.CodigoDispositivo,
		&reading.IDArranque,
		&reading.Secuencia,
		&temperature,
		&humidity,
		&rssi,
		&reading.EstadoSensor,
		&deviceIP,
		&reading.UptimeSegundos,
		&reading.LecturasValidas,
		&reading.LecturasFallidas,
		&reading.Origen,
		&reading.FechaRecepcion,
	)

	if err != nil {
		return models.IoTReading{}, err
	}

	reading.TemperaturaC =
		iotAPIFloatPointer(
			temperature,
		)

	reading.HumedadPct =
		iotAPIFloatPointer(
			humidity,
		)

	reading.RSSIDBM =
		iotAPIIntPointer(rssi)

	reading.IPDispositivo =
		iotAPIString(deviceIP)

	return reading, nil
}

func iotAPIString(
	value sql.NullString,
) string {
	if !value.Valid {
		return ""
	}

	return value.String
}

func iotAPIIntPointer(
	value sql.NullInt64,
) *int64 {
	if !value.Valid {
		return nil
	}

	result := value.Int64

	return &result
}

func iotAPIFloatPointer(
	value sql.NullFloat64,
) *float64 {
	if !value.Valid {
		return nil
	}

	result := value.Float64

	return &result
}

// ErrIoTAlertNotFound indica que la alerta
// solicitada no existe.
var ErrIoTAlertNotFound = errors.New(
	"alerta IoT no encontrada",
)

// ErrIoTAlertAlreadyClosed indica que la alerta
// ya no se encuentra pendiente.
var ErrIoTAlertAlreadyClosed = errors.New(
	"la alerta IoT ya fue procesada",
)

// ErrIoTAlertUserNotFound indica que el usuario
// responsable no existe.
var ErrIoTAlertUserNotFound = errors.New(
	"usuario responsable de alerta IoT no encontrado",
)

// AttendIoTAlert marca una alerta como atendida
// dentro de una transacción Oracle.
func (repository *IoTRepository) AttendIoTAlert(
	ctx context.Context,
	alertID int64,
	userID int64,
	observation string,
) (
	models.IoTAlertUpdateResult,
	error,
) {
	tx, err := repository.db.BeginTx(
		ctx,
		nil,
	)
	if err != nil {
		return models.IoTAlertUpdateResult{},
			fmt.Errorf(
				"no se pudo iniciar la transacción IoT: %w",
				err,
			)
	}

	committed := false

	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	const lockQuery = `
		SELECT
			ID_DISPOSITIVO,
			ESTADO

		FROM SIGEFER_APP.IOT_ALERTA

		WHERE ID_ALERTA_IOT = :1

		FOR UPDATE
	`

	var (
		deviceID     int64
		currentState string
	)

	err = tx.QueryRowContext(
		ctx,
		lockQuery,
		alertID,
	).Scan(
		&deviceID,
		&currentState,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return models.IoTAlertUpdateResult{},
			ErrIoTAlertNotFound
	}

	if err != nil {
		return models.IoTAlertUpdateResult{},
			fmt.Errorf(
				"no se pudo bloquear la alerta IoT: %w",
				err,
			)
	}

	if currentState != "PENDIENTE" {
		return models.IoTAlertUpdateResult{},
			ErrIoTAlertAlreadyClosed
	}

	const userQuery = `
		SELECT COUNT(*)

		FROM SIGEFER_APP.USUARIO

		WHERE ID_USUARIO = :1
		  AND ESTADO = 'ACTIVO'
	`

	var userCount int64

	err = tx.QueryRowContext(
		ctx,
		userQuery,
		userID,
	).Scan(
		&userCount,
	)
	if err != nil {
		return models.IoTAlertUpdateResult{},
			fmt.Errorf(
				"no se pudo validar el usuario responsable: %w",
				err,
			)
	}

	if userCount != 1 {
		return models.IoTAlertUpdateResult{},
			ErrIoTAlertUserNotFound
	}

	const updateStatement = `
		UPDATE SIGEFER_APP.IOT_ALERTA

		SET
			ESTADO = 'ATENDIDA',

			ID_USUARIO_ATENCION = :1,

			OBSERVACION_ATENCION = :2,

			FECHA_ATENCION = CAST(
				SYSTIMESTAMP AT TIME ZONE '-05:00'
				AS TIMESTAMP
			),

			FECHA_ACTUALIZACION = CAST(
				SYSTIMESTAMP AT TIME ZONE '-05:00'
				AS TIMESTAMP
			)

		WHERE ID_ALERTA_IOT = :3
		  AND ID_DISPOSITIVO = :4
		  AND ESTADO = 'PENDIENTE'
	`

	result, err := tx.ExecContext(
		ctx,
		updateStatement,
		userID,
		observation,
		alertID,
		deviceID,
	)
	if err != nil {
		return models.IoTAlertUpdateResult{},
			fmt.Errorf(
				"no se pudo atender la alerta IoT: %w",
				err,
			)
	}

	affectedRows, err := result.RowsAffected()
	if err != nil {
		return models.IoTAlertUpdateResult{},
			fmt.Errorf(
				"no se pudo comprobar la atención de la alerta IoT: %w",
				err,
			)
	}

	if affectedRows != 1 {
		return models.IoTAlertUpdateResult{},
			ErrIoTAlertAlreadyClosed
	}

	const resultQuery = `
		SELECT
			a.ID_ALERTA_IOT,
			d.CODIGO,
			a.TIPO_ALERTA,
			a.SEVERIDAD,
			a.MENSAJE,
			a.VALOR_DETECTADO,
			a.VALOR_LIMITE,
			a.ESTADO,

			TO_CHAR(
				a.FECHA_GENERACION,
				'YYYY-MM-DD"T"HH24:MI:SS'
			),

			a.ID_USUARIO_ATENCION,

			TO_CHAR(
				a.FECHA_ATENCION,
				'YYYY-MM-DD"T"HH24:MI:SS'
			),

			a.OBSERVACION_ATENCION

		FROM SIGEFER_APP.IOT_ALERTA a

		INNER JOIN SIGEFER_APP.IOT_DISPOSITIVO d
			ON d.ID_DISPOSITIVO =
				a.ID_DISPOSITIVO

		WHERE a.ID_ALERTA_IOT = :1
	`

	var (
		updatedAlert models.IoTAlertUpdateResult

		detectedValue sql.NullFloat64
		limitValue    sql.NullFloat64
	)

	err = tx.QueryRowContext(
		ctx,
		resultQuery,
		alertID,
	).Scan(
		&updatedAlert.IDAlertaIoT,
		&updatedAlert.CodigoDispositivo,
		&updatedAlert.TipoAlerta,
		&updatedAlert.Severidad,
		&updatedAlert.Mensaje,
		&detectedValue,
		&limitValue,
		&updatedAlert.Estado,
		&updatedAlert.FechaGeneracion,
		&updatedAlert.IDUsuarioAtencion,
		&updatedAlert.FechaAtencion,
		&updatedAlert.ObservacionAtencion,
	)
	if err != nil {
		return models.IoTAlertUpdateResult{},
			fmt.Errorf(
				"no se pudo consultar la alerta IoT atendida: %w",
				err,
			)
	}

	updatedAlert.Status = "ok"

	updatedAlert.ValorDetectado =
		iotAPIFloatPointer(
			detectedValue,
		)

	updatedAlert.ValorLimite =
		iotAPIFloatPointer(
			limitValue,
		)

	if err := tx.Commit(); err != nil {
		return models.IoTAlertUpdateResult{},
			fmt.Errorf(
				"no se pudo confirmar la atención de la alerta IoT: %w",
				err,
			)
	}

	committed = true

	return updatedAlert, nil
}

// UpdateIoTConfiguration modifica los umbrales
// ambientales dentro de una transacción Oracle.
func (repository *IoTRepository) UpdateIoTConfiguration(
	ctx context.Context,
	deviceCode string,
	request models.UpdateIoTConfigurationRequest,
) (
	models.IoTConfiguration,
	error,
) {
	deviceID, err := repository.findIoTDeviceID(
		ctx,
		deviceCode,
	)
	if err != nil {
		return models.IoTConfiguration{}, err
	}

	tx, err := repository.db.BeginTx(
		ctx,
		nil,
	)
	if err != nil {
		return models.IoTConfiguration{},
			fmt.Errorf(
				"no se pudo iniciar la transacción de configuración IoT: %w",
				err,
			)
	}

	committed := false

	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	const lockQuery = `
		SELECT ID_CONFIGURACION

		FROM SIGEFER_APP.IOT_CONFIGURACION

		WHERE ID_DISPOSITIVO = :1

		FOR UPDATE
	`

	var configurationID int64

	err = tx.QueryRowContext(
		ctx,
		lockQuery,
		deviceID,
	).Scan(
		&configurationID,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return models.IoTConfiguration{},
			ErrIoTConfigurationNotFound
	}

	if err != nil {
		return models.IoTConfiguration{},
			fmt.Errorf(
				"no se pudo bloquear la configuración IoT: %w",
				err,
			)
	}

	const updateStatement = `
		UPDATE SIGEFER_APP.IOT_CONFIGURACION

		SET
			TEMPERATURA_MIN_C = :1,
			TEMPERATURA_MAX_C = :2,
			HUMEDAD_MIN_PCT = :3,
			HUMEDAD_MAX_PCT = :4,
			SEGUNDOS_SIN_COMUNICACION = :5,

			FECHA_ACTUALIZACION = CAST(
				SYSTIMESTAMP AT TIME ZONE '-05:00'
				AS TIMESTAMP
			)

		WHERE ID_CONFIGURACION = :6
		  AND ID_DISPOSITIVO = :7
	`

	result, err := tx.ExecContext(
		ctx,
		updateStatement,
		request.TemperaturaMinC,
		request.TemperaturaMaxC,
		request.HumedadMinPct,
		request.HumedadMaxPct,
		request.SegundosSinComunicacion,
		configurationID,
		deviceID,
	)
	if err != nil {
		return models.IoTConfiguration{},
			fmt.Errorf(
				"no se pudo actualizar la configuración IoT: %w",
				err,
			)
	}

	affectedRows, err := result.RowsAffected()
	if err != nil {
		return models.IoTConfiguration{},
			fmt.Errorf(
				"no se pudo comprobar la actualización IoT: %w",
				err,
			)
	}

	if affectedRows != 1 {
		return models.IoTConfiguration{},
			ErrIoTConfigurationNotFound
	}

	const resultQuery = `
		SELECT
			c.ID_CONFIGURACION,
			c.ID_DISPOSITIVO,
			d.CODIGO,
			c.TEMPERATURA_MIN_C,
			c.TEMPERATURA_MAX_C,
			c.HUMEDAD_MIN_PCT,
			c.HUMEDAD_MAX_PCT,
			c.SEGUNDOS_SIN_COMUNICACION,
			c.ESTADO,

			TO_CHAR(
				c.FECHA_ACTUALIZACION,
				'YYYY-MM-DD"T"HH24:MI:SS'
			)

		FROM SIGEFER_APP.IOT_CONFIGURACION c

		INNER JOIN SIGEFER_APP.IOT_DISPOSITIVO d
			ON d.ID_DISPOSITIVO =
				c.ID_DISPOSITIVO

		WHERE c.ID_CONFIGURACION = :1
	`

	var configuration models.IoTConfiguration

	err = tx.QueryRowContext(
		ctx,
		resultQuery,
		configurationID,
	).Scan(
		&configuration.IDConfiguracion,
		&configuration.IDDispositivo,
		&configuration.CodigoDispositivo,
		&configuration.TemperaturaMinC,
		&configuration.TemperaturaMaxC,
		&configuration.HumedadMinPct,
		&configuration.HumedadMaxPct,
		&configuration.SegundosSinComunicacion,
		&configuration.Estado,
		&configuration.FechaActualizacion,
	)
	if err != nil {
		return models.IoTConfiguration{},
			fmt.Errorf(
				"no se pudo consultar la configuración IoT actualizada: %w",
				err,
			)
	}

	if err := tx.Commit(); err != nil {
		return models.IoTConfiguration{},
			fmt.Errorf(
				"no se pudo confirmar la configuración IoT: %w",
				err,
			)
	}

	committed = true

	return configuration, nil
}
