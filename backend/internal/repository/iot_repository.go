package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"sigefer.local/backend/internal/models"
)

type IoTRepository struct {
	db *sql.DB
}

type iotThresholds struct {
	temperatureMin float64
	temperatureMax float64
	humidityMin    float64
	humidityMax    float64
}

func NewIoTRepository(
	db *sql.DB,
) *IoTRepository {
	return &IoTRepository{
		db: db,
	}
}

// ProcessTelemetry guarda una lectura, actualiza el
// dispositivo y crea las alertas correspondientes.
func (repository *IoTRepository) ProcessTelemetry(
	ctx context.Context,
	telemetry models.IoTTelemetry,
	rawPayload string,
) (
	models.IoTProcessResult,
	error,
) {
	tx, err := repository.db.BeginTx(
		ctx,
		nil,
	)
	if err != nil {
		return models.IoTProcessResult{},
			fmt.Errorf(
				"no se pudo iniciar la transacción IoT: %w",
				err,
			)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	deviceID, err := lockDevice(
		ctx,
		tx,
		telemetry.DeviceCode,
	)
	if err != nil {
		return models.IoTProcessResult{},
			err
	}

	existingReadingID, found, err :=
		findExistingReading(
			ctx,
			tx,
			deviceID,
			telemetry.BootID,
			telemetry.Sequence,
		)
	if err != nil {
		return models.IoTProcessResult{},
			err
	}

	if found {
		return models.IoTProcessResult{
			ReadingID: existingReadingID,
			Duplicate: true,
		}, nil
	}

	if err := insertReading(
		ctx,
		tx,
		deviceID,
		telemetry,
		rawPayload,
	); err != nil {
		return models.IoTProcessResult{},
			err
	}

	readingID, err := selectReadingID(
		ctx,
		tx,
		deviceID,
		telemetry.BootID,
		telemetry.Sequence,
	)
	if err != nil {
		return models.IoTProcessResult{},
			err
	}

	if err := updateDeviceStatus(
		ctx,
		tx,
		deviceID,
		telemetry,
	); err != nil {
		return models.IoTProcessResult{},
			err
	}

	thresholds, configured, err :=
		loadThresholds(
			ctx,
			tx,
			deviceID,
		)
	if err != nil {
		return models.IoTProcessResult{},
			err
	}

	alerts := make([]string, 0, 3)

	if configured {
		alerts, err = evaluateAlerts(
			ctx,
			tx,
			deviceID,
			readingID,
			telemetry,
			thresholds,
		)
		if err != nil {
			return models.IoTProcessResult{},
				err
		}
	}

	if err := tx.Commit(); err != nil {
		return models.IoTProcessResult{},
			fmt.Errorf(
				"no se pudo confirmar la lectura IoT: %w",
				err,
			)
	}

	return models.IoTProcessResult{
		ReadingID: readingID,
		Alerts:    alerts,
	}, nil
}

func lockDevice(
	ctx context.Context,
	tx *sql.Tx,
	deviceCode string,
) (
	int64,
	error,
) {
	const query = `
		SELECT ID_DISPOSITIVO
		FROM SIGEFER_APP.IOT_DISPOSITIVO
		WHERE CODIGO = :1
		  AND ESTADO = 'A'
		FOR UPDATE
	`

	var deviceID int64

	err := tx.QueryRowContext(
		ctx,
		query,
		deviceCode,
	).Scan(
		&deviceID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf(
			"dispositivo IoT no autorizado: %s",
			deviceCode,
		)
	}
	if err != nil {
		return 0, fmt.Errorf(
			"no se pudo consultar el dispositivo IoT: %w",
			err,
		)
	}

	return deviceID, nil
}

func findExistingReading(
	ctx context.Context,
	tx *sql.Tx,
	deviceID int64,
	bootID string,
	sequence uint64,
) (
	int64,
	bool,
	error,
) {
	const query = `
		SELECT ID_LECTURA
		FROM SIGEFER_APP.IOT_LECTURA
		WHERE ID_DISPOSITIVO = :1
		  AND ID_ARRANQUE = :2
		  AND SECUENCIA = :3
	`

	var readingID int64

	err := tx.QueryRowContext(
		ctx,
		query,
		deviceID,
		bootID,
		sequence,
	).Scan(
		&readingID,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}

	if err != nil {
		return 0, false, fmt.Errorf(
			"no se pudo comprobar la duplicidad IoT: %w",
			err,
		)
	}

	return readingID, true, nil
}

func insertReading(
	ctx context.Context,
	tx *sql.Tx,
	deviceID int64,
	telemetry models.IoTTelemetry,
	rawPayload string,
) error {
	const statement = `
		INSERT INTO SIGEFER_APP.IOT_LECTURA (
			ID_DISPOSITIVO,
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
			PAYLOAD_JSON
		)
		VALUES (
			:1, :2, :3, :4, :5, :6, :7,
			:8, :9, :10, :11, :12, :13
		)
	`

	_, err := tx.ExecContext(
		ctx,
		statement,
		deviceID,
		telemetry.BootID,
		telemetry.Sequence,
		iotNullableFloat(telemetry.TemperatureC),
		iotNullableFloat(telemetry.HumidityPct),
		iotNullableInt(telemetry.RSSIDBM),
		telemetry.SensorState,
		iotNullableText(telemetry.DeviceIP),
		telemetry.UptimeSeconds,
		telemetry.ValidReadings,
		telemetry.FailedReadings,
		telemetry.Origin,
		rawPayload,
	)
	if err != nil {
		return fmt.Errorf(
			"no se pudo insertar la lectura IoT: %w",
			err,
		)
	}

	return nil
}

func selectReadingID(
	ctx context.Context,
	tx *sql.Tx,
	deviceID int64,
	bootID string,
	sequence uint64,
) (
	int64,
	error,
) {
	const query = `
		SELECT ID_LECTURA
		FROM SIGEFER_APP.IOT_LECTURA
		WHERE ID_DISPOSITIVO = :1
		  AND ID_ARRANQUE = :2
		  AND SECUENCIA = :3
	`

	var readingID int64

	err := tx.QueryRowContext(
		ctx,
		query,
		deviceID,
		bootID,
		sequence,
	).Scan(
		&readingID,
	)
	if err != nil {
		return 0, fmt.Errorf(
			"no se pudo recuperar la lectura IoT: %w",
			err,
		)
	}

	return readingID, nil
}

func updateDeviceStatus(
	ctx context.Context,
	tx *sql.Tx,
	deviceID int64,
	telemetry models.IoTTelemetry,
) error {
	const statement = `
		UPDATE SIGEFER_APP.IOT_DISPOSITIVO
		SET
			ULTIMA_COMUNICACION =
				CURRENT_TIMESTAMP,
			ULTIMA_IP = :1,
			ULTIMO_RSSI_DBM = :2,
			ULTIMO_ESTADO_SENSOR = :3,
			FECHA_ACTUALIZACION =
				CURRENT_TIMESTAMP
		WHERE ID_DISPOSITIVO = :4
	`

	_, err := tx.ExecContext(
		ctx,
		statement,
		iotNullableText(telemetry.DeviceIP),
		iotNullableInt(telemetry.RSSIDBM),
		telemetry.SensorState,
		deviceID,
	)
	if err != nil {
		return fmt.Errorf(
			"no se pudo actualizar el dispositivo IoT: %w",
			err,
		)
	}

	return nil
}

func loadThresholds(
	ctx context.Context,
	tx *sql.Tx,
	deviceID int64,
) (
	iotThresholds,
	bool,
	error,
) {
	const query = `
		SELECT
			TEMPERATURA_MIN_C,
			TEMPERATURA_MAX_C,
			HUMEDAD_MIN_PCT,
			HUMEDAD_MAX_PCT
		FROM SIGEFER_APP.IOT_CONFIGURACION
		WHERE ID_DISPOSITIVO = :1
		  AND ESTADO = 'A'
	`

	var thresholds iotThresholds

	err := tx.QueryRowContext(
		ctx,
		query,
		deviceID,
	).Scan(
		&thresholds.temperatureMin,
		&thresholds.temperatureMax,
		&thresholds.humidityMin,
		&thresholds.humidityMax,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return iotThresholds{}, false, nil
	}

	if err != nil {
		return iotThresholds{}, false,
			fmt.Errorf(
				"no se pudo consultar la configuración IoT: %w",
				err,
			)
	}

	return thresholds, true, nil
}

func evaluateAlerts(
	ctx context.Context,
	tx *sql.Tx,
	deviceID int64,
	readingID int64,
	telemetry models.IoTTelemetry,
	thresholds iotThresholds,
) (
	[]string,
	error,
) {
	alerts := make([]string, 0, 3)

	if telemetry.SensorState == "ERROR_LECTURA" {
		err := mergeAlert(
			ctx,
			tx,
			deviceID,
			readingID,
			"ERROR_SENSOR",
			"CRITICA",
			"El dispositivo informó un error de lectura del sensor",
			nil,
			nil,
		)
		if err != nil {
			return nil, err
		}

		alerts = append(
			alerts,
			"ERROR_SENSOR",
		)

		return alerts, nil
	}

	if telemetry.TemperatureC != nil {
		temperature := *telemetry.TemperatureC

		if temperature > thresholds.temperatureMax {
			message := fmt.Sprintf(
				"Temperatura %.1f C supera el limite %.1f C",
				temperature,
				thresholds.temperatureMax,
			)

			err := mergeAlert(
				ctx,
				tx,
				deviceID,
				readingID,
				"TEMPERATURA_ALTA",
				"ADVERTENCIA",
				message,
				&temperature,
				&thresholds.temperatureMax,
			)
			if err != nil {
				return nil, err
			}

			alerts = append(
				alerts,
				"TEMPERATURA_ALTA",
			)
		}

		if temperature < thresholds.temperatureMin {
			message := fmt.Sprintf(
				"Temperatura %.1f C es inferior al limite %.1f C",
				temperature,
				thresholds.temperatureMin,
			)

			err := mergeAlert(
				ctx,
				tx,
				deviceID,
				readingID,
				"TEMPERATURA_BAJA",
				"ADVERTENCIA",
				message,
				&temperature,
				&thresholds.temperatureMin,
			)
			if err != nil {
				return nil, err
			}

			alerts = append(
				alerts,
				"TEMPERATURA_BAJA",
			)
		}
	}

	if telemetry.HumidityPct != nil {
		humidity := *telemetry.HumidityPct

		if humidity > thresholds.humidityMax {
			message := fmt.Sprintf(
				"Humedad %.1f por ciento supera el limite %.1f",
				humidity,
				thresholds.humidityMax,
			)

			err := mergeAlert(
				ctx,
				tx,
				deviceID,
				readingID,
				"HUMEDAD_ALTA",
				"ADVERTENCIA",
				message,
				&humidity,
				&thresholds.humidityMax,
			)
			if err != nil {
				return nil, err
			}

			alerts = append(
				alerts,
				"HUMEDAD_ALTA",
			)
		}

		if humidity < thresholds.humidityMin {
			message := fmt.Sprintf(
				"Humedad %.1f por ciento es inferior al limite %.1f",
				humidity,
				thresholds.humidityMin,
			)

			err := mergeAlert(
				ctx,
				tx,
				deviceID,
				readingID,
				"HUMEDAD_BAJA",
				"ADVERTENCIA",
				message,
				&humidity,
				&thresholds.humidityMin,
			)
			if err != nil {
				return nil, err
			}

			alerts = append(
				alerts,
				"HUMEDAD_BAJA",
			)
		}
	}

	return alerts, nil
}

func mergeAlert(
	ctx context.Context,
	tx *sql.Tx,
	deviceID int64,
	readingID int64,
	alertType string,
	severity string,
	message string,
	detectedValue *float64,
	limitValue *float64,
) error {
	const statement = `
		MERGE INTO SIGEFER_APP.IOT_ALERTA alerta
		USING (
			SELECT
				:1 AS ID_DISPOSITIVO,
				:2 AS ID_LECTURA,
				:3 AS TIPO_ALERTA,
				:4 AS SEVERIDAD,
				:5 AS MENSAJE,
				:6 AS VALOR_DETECTADO,
				:7 AS VALOR_LIMITE
			FROM DUAL
		) origen
		ON (
			alerta.ID_DISPOSITIVO =
				origen.ID_DISPOSITIVO
			AND alerta.TIPO_ALERTA =
				origen.TIPO_ALERTA
			AND alerta.ESTADO = 'PENDIENTE'
		)
		WHEN MATCHED THEN
			UPDATE SET
				alerta.ID_LECTURA =
					origen.ID_LECTURA,
				alerta.SEVERIDAD =
					origen.SEVERIDAD,
				alerta.MENSAJE =
					origen.MENSAJE,
				alerta.VALOR_DETECTADO =
					origen.VALOR_DETECTADO,
				alerta.VALOR_LIMITE =
					origen.VALOR_LIMITE,
				alerta.FECHA_ACTUALIZACION =
					CURRENT_TIMESTAMP
		WHEN NOT MATCHED THEN
			INSERT (
				ID_DISPOSITIVO,
				ID_LECTURA,
				TIPO_ALERTA,
				SEVERIDAD,
				MENSAJE,
				VALOR_DETECTADO,
				VALOR_LIMITE,
				ESTADO
			)
			VALUES (
				origen.ID_DISPOSITIVO,
				origen.ID_LECTURA,
				origen.TIPO_ALERTA,
				origen.SEVERIDAD,
				origen.MENSAJE,
				origen.VALOR_DETECTADO,
				origen.VALOR_LIMITE,
				'PENDIENTE'
			)
	`

	_, err := tx.ExecContext(
		ctx,
		statement,
		deviceID,
		readingID,
		alertType,
		severity,
		message,
		iotNullableFloat(detectedValue),
		iotNullableFloat(limitValue),
	)
	if err != nil {
		return fmt.Errorf(
			"no se pudo generar la alerta %s: %w",
			alertType,
			err,
		)
	}

	return nil
}

// CheckOfflineDevices genera o actualiza alertas
// cuando un dispositivo deja de comunicarse.
func (repository *IoTRepository) CheckOfflineDevices(
	ctx context.Context,
) (
	int64,
	error,
) {
	const statement = `
		MERGE INTO SIGEFER_APP.IOT_ALERTA alerta
		USING (
			SELECT
				dispositivo.ID_DISPOSITIVO,
				ROUND(
					(
						CAST(CURRENT_TIMESTAMP AS DATE)
						-
						CAST(
							dispositivo.ULTIMA_COMUNICACION
							AS DATE
						)
					) * 86400,
					2
				) AS SEGUNDOS_TRANSCURRIDOS,
				configuracion.SEGUNDOS_SIN_COMUNICACION
			FROM SIGEFER_APP.IOT_DISPOSITIVO dispositivo
			INNER JOIN SIGEFER_APP.IOT_CONFIGURACION configuracion
				ON configuracion.ID_DISPOSITIVO =
					dispositivo.ID_DISPOSITIVO
			WHERE dispositivo.ESTADO = 'A'
			  AND configuracion.ESTADO = 'A'
			  AND dispositivo.ULTIMA_COMUNICACION
					IS NOT NULL
			  AND dispositivo.ULTIMA_COMUNICACION <
					CURRENT_TIMESTAMP
					-
					NUMTODSINTERVAL(
						configuracion.SEGUNDOS_SIN_COMUNICACION,
						'SECOND'
					)
		) origen
		ON (
			alerta.ID_DISPOSITIVO =
				origen.ID_DISPOSITIVO
			AND alerta.TIPO_ALERTA =
				'SIN_COMUNICACION'
			AND alerta.ESTADO =
				'PENDIENTE'
		)
		WHEN MATCHED THEN
			UPDATE SET
				alerta.MENSAJE =
					'Dispositivo sin comunicacion dentro del tiempo configurado',
				alerta.VALOR_DETECTADO =
					origen.SEGUNDOS_TRANSCURRIDOS,
				alerta.VALOR_LIMITE =
					origen.SEGUNDOS_SIN_COMUNICACION,
				alerta.FECHA_ACTUALIZACION =
					CURRENT_TIMESTAMP
		WHEN NOT MATCHED THEN
			INSERT (
				ID_DISPOSITIVO,
				ID_LECTURA,
				TIPO_ALERTA,
				SEVERIDAD,
				MENSAJE,
				VALOR_DETECTADO,
				VALOR_LIMITE,
				ESTADO
			)
			VALUES (
				origen.ID_DISPOSITIVO,
				NULL,
				'SIN_COMUNICACION',
				'CRITICA',
				'Dispositivo sin comunicacion dentro del tiempo configurado',
				origen.SEGUNDOS_TRANSCURRIDOS,
				origen.SEGUNDOS_SIN_COMUNICACION,
				'PENDIENTE'
			)
	`

	result, err := repository.db.ExecContext(
		ctx,
		statement,
	)
	if err != nil {
		return 0, fmt.Errorf(
			"no se pudo comprobar la comunicación IoT: %w",
			err,
		)
	}

	affectedRows, err := result.RowsAffected()
	if err != nil {
		return 0, nil
	}

	return affectedRows, nil
}

func iotNullableFloat(
	value *float64,
) any {
	if value == nil {
		return nil
	}

	return *value
}

func iotNullableInt(
	value *int64,
) any {
	if value == nil {
		return nil
	}

	return *value
}

func iotNullableText(
	value string,
) any {
	if value == "" {
		return nil
	}

	return value
}
