package models

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

// IoTTelemetry representa el JSON publicado
// por el ESP32.
type IoTTelemetry struct {
	DeviceCode string `json:"dispositivo"`
	BootID     string `json:"id_arranque"`
	Location   string `json:"ubicacion"`

	TemperatureC *float64 `json:"temperatura_c"`
	HumidityPct  *float64 `json:"humedad_pct"`
	RSSIDBM      *int64   `json:"senal_wifi_dbm"`

	SensorState string `json:"estado_sensor"`
	DeviceIP    string `json:"ip_dispositivo"`

	Sequence       uint64 `json:"secuencia"`
	UptimeSeconds  uint64 `json:"uptime_s"`
	ValidReadings  uint64 `json:"lecturas_validas"`
	FailedReadings uint64 `json:"lecturas_fallidas"`

	Origin string `json:"origen"`
}

// IoTProcessResult informa el resultado
// de persistir una lectura.
type IoTProcessResult struct {
	ReadingID int64
	Duplicate bool
	Alerts    []string
}

// Normalize limpia campos textuales antes
// de aplicar las validaciones.
func (telemetry *IoTTelemetry) Normalize() {
	telemetry.DeviceCode = strings.ToUpper(
		strings.TrimSpace(telemetry.DeviceCode),
	)

	telemetry.BootID = strings.ToUpper(
		strings.TrimSpace(telemetry.BootID),
	)

	telemetry.Location = strings.TrimSpace(
		telemetry.Location,
	)

	telemetry.SensorState = strings.ToUpper(
		strings.TrimSpace(telemetry.SensorState),
	)

	telemetry.DeviceIP = strings.TrimSpace(
		telemetry.DeviceIP,
	)

	telemetry.Origin = strings.ToUpper(
		strings.TrimSpace(telemetry.Origin),
	)
}

// Validate comprueba la estructura y los rangos
// admitidos por Oracle.
func (telemetry IoTTelemetry) Validate() error {
	switch {
	case telemetry.DeviceCode == "":
		return errors.New(
			"dispositivo es obligatorio",
		)

	case len(telemetry.DeviceCode) > 50:
		return errors.New(
			"dispositivo supera 50 caracteres",
		)

	case len(telemetry.BootID) != 16:
		return errors.New(
			"id_arranque debe contener 16 caracteres hexadecimales",
		)

	case telemetry.Sequence == 0:
		return errors.New(
			"secuencia debe ser mayor que cero",
		)

	case telemetry.RSSIDBM == nil:
		return errors.New(
			"senal_wifi_dbm es obligatoria",
		)

	case *telemetry.RSSIDBM < -150 ||
		*telemetry.RSSIDBM > 0:
		return errors.New(
			"senal_wifi_dbm está fuera del rango permitido",
		)

	case len(telemetry.Location) > 200:
		return errors.New(
			"ubicacion supera 200 caracteres",
		)

	case len(telemetry.DeviceIP) > 50:
		return errors.New(
			"ip_dispositivo supera 50 caracteres",
		)

	case telemetry.Origin == "":
		return errors.New(
			"origen es obligatorio",
		)

	case len(telemetry.Origin) > 50:
		return errors.New(
			"origen supera 50 caracteres",
		)
	}

	if _, err := hex.DecodeString(
		telemetry.BootID,
	); err != nil {
		return errors.New(
			"id_arranque no es hexadecimal",
		)
	}

	switch telemetry.SensorState {
	case "OPERATIVO":
		if telemetry.TemperatureC == nil ||
			telemetry.HumidityPct == nil {
			return errors.New(
				"sensor operativo requiere temperatura y humedad",
			)
		}

	case "ERROR_LECTURA":
		if telemetry.TemperatureC != nil ||
			telemetry.HumidityPct != nil {
			return errors.New(
				"ERROR_LECTURA debe enviar temperatura y humedad nulas",
			)
		}

	default:
		return fmt.Errorf(
			"estado_sensor no permitido: %s",
			telemetry.SensorState,
		)
	}

	if telemetry.TemperatureC != nil &&
		(*telemetry.TemperatureC < -50 ||
			*telemetry.TemperatureC > 100) {
		return errors.New(
			"temperatura_c está fuera del rango permitido",
		)
	}

	if telemetry.HumidityPct != nil &&
		(*telemetry.HumidityPct < 0 ||
			*telemetry.HumidityPct > 100) {
		return errors.New(
			"humedad_pct está fuera del rango permitido",
		)
	}

	return nil
}
