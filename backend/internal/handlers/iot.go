package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"sigefer.local/backend/internal/middleware"
	"sigefer.local/backend/internal/models"
	"sigefer.local/backend/internal/repository"
)

const defaultIoTDeviceCode = "BODEGA-01"

var iotDeviceCodePattern = regexp.MustCompile(
	`^[A-Z0-9][A-Z0-9_-]{0,29}$`,
)

// IoTHandler procesa las operaciones REST
// del monitoreo ambiental.
type IoTHandler struct {
	repository *repository.IoTRepository
}

// NewIoTHandler crea el handler del módulo IoT.
func NewIoTHandler(
	iotRepository *repository.IoTRepository,
) *IoTHandler {
	return &IoTHandler{
		repository: iotRepository,
	}
}

// Summary procesa GET /api/v1/iot/resumen.
func (handler *IoTHandler) Summary(
	writer http.ResponseWriter,
	request *http.Request,
) {
	deviceCode, err :=
		parseIoTDeviceCode(request)

	if err != nil {
		writeIoTError(
			writer,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	ctx, cancel := context.WithTimeout(
		request.Context(),
		10*time.Second,
	)
	defer cancel()

	summary, err :=
		handler.repository.GetIoTSummary(
			ctx,
			deviceCode,
		)

	switch {
	case errors.Is(
		err,
		repository.ErrIoTDeviceNotFound,
	):
		writeIoTError(
			writer,
			http.StatusNotFound,
			"dispositivo IoT no encontrado",
		)

	case errors.Is(
		err,
		repository.ErrIoTConfigurationNotFound,
	):
		writeIoTError(
			writer,
			http.StatusNotFound,
			"configuración IoT no encontrada",
		)

	case err != nil:
		log.Printf(
			"error consultando resumen IoT de %s: %v",
			deviceCode,
			err,
		)

		writeIoTError(
			writer,
			http.StatusInternalServerError,
			"no fue posible consultar el resumen IoT",
		)

	default:
		writeIoTResponse(
			writer,
			http.StatusOK,
			summary,
		)
	}
}

// ListReadings procesa GET /api/v1/iot/lecturas.
func (handler *IoTHandler) ListReadings(
	writer http.ResponseWriter,
	request *http.Request,
) {
	deviceCode, err :=
		parseIoTDeviceCode(request)

	if err != nil {
		writeIoTError(
			writer,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	limit, err := parseIoTLimit(
		request,
		100,
	)

	if err != nil {
		writeIoTError(
			writer,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	ctx, cancel := context.WithTimeout(
		request.Context(),
		10*time.Second,
	)
	defer cancel()

	readings, err :=
		handler.repository.ListIoTReadings(
			ctx,
			deviceCode,
			limit,
		)

	switch {
	case errors.Is(
		err,
		repository.ErrIoTDeviceNotFound,
	):
		writeIoTError(
			writer,
			http.StatusNotFound,
			"dispositivo IoT no encontrado",
		)

	case err != nil:
		log.Printf(
			"error consultando lecturas IoT de %s: %v",
			deviceCode,
			err,
		)

		writeIoTError(
			writer,
			http.StatusInternalServerError,
			"no fue posible consultar las lecturas IoT",
		)

	default:
		writeIoTResponse(
			writer,
			http.StatusOK,
			models.IoTReadingListResponse{
				Status:      "ok",
				Total:       len(readings),
				Dispositivo: deviceCode,
				Lecturas:    readings,
			},
		)
	}
}

// ListAlerts procesa GET /api/v1/iot/alertas.
func (handler *IoTHandler) ListAlerts(
	writer http.ResponseWriter,
	request *http.Request,
) {
	deviceCode, err :=
		parseIoTDeviceCode(request)

	if err != nil {
		writeIoTError(
			writer,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	status := strings.ToUpper(
		strings.TrimSpace(
			request.URL.Query().Get("estado"),
		),
	)

	if status != "" &&
		status != "PENDIENTE" &&
		status != "ATENDIDA" {
		writeIoTError(
			writer,
			http.StatusBadRequest,
			"estado debe ser PENDIENTE o ATENDIDA",
		)
		return
	}

	limit, err := parseIoTLimit(
		request,
		100,
	)

	if err != nil {
		writeIoTError(
			writer,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	ctx, cancel := context.WithTimeout(
		request.Context(),
		10*time.Second,
	)
	defer cancel()

	alerts, err :=
		handler.repository.ListIoTAlerts(
			ctx,
			deviceCode,
			status,
			limit,
		)

	switch {
	case errors.Is(
		err,
		repository.ErrIoTDeviceNotFound,
	):
		writeIoTError(
			writer,
			http.StatusNotFound,
			"dispositivo IoT no encontrado",
		)

	case err != nil:
		log.Printf(
			"error consultando alertas IoT de %s: %v",
			deviceCode,
			err,
		)

		writeIoTError(
			writer,
			http.StatusInternalServerError,
			"no fue posible consultar las alertas IoT",
		)

	default:
		writeIoTResponse(
			writer,
			http.StatusOK,
			models.IoTAlertListResponse{
				Status:      "ok",
				Total:       len(alerts),
				Dispositivo: deviceCode,
				Estado:      status,
				Alertas:     alerts,
			},
		)
	}
}

// GetConfiguration procesa
// GET /api/v1/iot/configuracion.
func (handler *IoTHandler) GetConfiguration(
	writer http.ResponseWriter,
	request *http.Request,
) {
	deviceCode, err :=
		parseIoTDeviceCode(request)

	if err != nil {
		writeIoTError(
			writer,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	ctx, cancel := context.WithTimeout(
		request.Context(),
		10*time.Second,
	)
	defer cancel()

	configuration, err :=
		handler.repository.GetIoTConfiguration(
			ctx,
			deviceCode,
		)

	switch {
	case errors.Is(
		err,
		repository.ErrIoTConfigurationNotFound,
	):
		writeIoTError(
			writer,
			http.StatusNotFound,
			"configuración IoT no encontrada",
		)

	case err != nil:
		log.Printf(
			"error consultando configuración IoT de %s: %v",
			deviceCode,
			err,
		)

		writeIoTError(
			writer,
			http.StatusInternalServerError,
			"no fue posible consultar la configuración IoT",
		)

	default:
		writeIoTResponse(
			writer,
			http.StatusOK,
			models.IoTConfigurationResponse{
				Status:        "ok",
				Configuracion: configuration,
			},
		)
	}
}

// AttendAlert procesa
// PATCH /api/v1/iot/alertas/{id}/atender.
func (handler *IoTHandler) AttendAlert(
	writer http.ResponseWriter,
	request *http.Request,
) {
	alertID, err := strconv.ParseInt(
		strings.TrimSpace(
			request.PathValue("id"),
		),
		10,
		64,
	)

	if err != nil || alertID <= 0 {
		writeIoTError(
			writer,
			http.StatusBadRequest,
			"identificador de alerta IoT inválido",
		)
		return
	}

	principal, authenticated :=
		middleware.PrincipalFromContext(
			request.Context(),
		)

	if !authenticated ||
		principal.IDUsuario <= 0 {
		writeIoTError(
			writer,
			http.StatusUnauthorized,
			"sesión autenticada no disponible",
		)
		return
	}

	var payload models.AttendIoTAlertRequest

	if err := decodeIoTJSON(
		writer,
		request,
		&payload,
	); err != nil {
		writeIoTError(
			writer,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	payload.Observacion = strings.TrimSpace(
		payload.Observacion,
	)

	observationLength :=
		utf8.RuneCountInString(
			payload.Observacion,
		)

	if observationLength < 5 ||
		observationLength > 500 {
		writeIoTError(
			writer,
			http.StatusBadRequest,
			"observacion debe contener entre 5 y 500 caracteres",
		)
		return
	}

	if strings.ContainsRune(
		payload.Observacion,
		'\uFFFD',
	) {
		writeIoTError(
			writer,
			http.StatusBadRequest,
			"observacion contiene caracteres inválidos",
		)
		return
	}

	ctx, cancel := context.WithTimeout(
		request.Context(),
		15*time.Second,
	)
	defer cancel()

	alert, err :=
		handler.repository.AttendIoTAlert(
			ctx,
			alertID,
			principal.IDUsuario,
			payload.Observacion,
		)

	switch {
	case errors.Is(
		err,
		repository.ErrIoTAlertNotFound,
	):
		writeIoTError(
			writer,
			http.StatusNotFound,
			"alerta IoT no encontrada",
		)

	case errors.Is(
		err,
		repository.ErrIoTAlertAlreadyClosed,
	):
		writeIoTError(
			writer,
			http.StatusConflict,
			"la alerta IoT ya fue atendida",
		)

	case errors.Is(
		err,
		repository.ErrIoTAlertUserNotFound,
	):
		writeIoTError(
			writer,
			http.StatusBadRequest,
			"usuario responsable no encontrado o inactivo",
		)

	case err != nil:
		log.Printf(
			"error atendiendo alerta IoT %d: %v",
			alertID,
			err,
		)

		writeIoTError(
			writer,
			http.StatusInternalServerError,
			"no fue posible atender la alerta IoT",
		)

	default:
		writeIoTResponse(
			writer,
			http.StatusOK,
			alert,
		)
	}
}

// UpdateConfiguration procesa
// PATCH /api/v1/iot/configuracion.
func (handler *IoTHandler) UpdateConfiguration(
	writer http.ResponseWriter,
	request *http.Request,
) {
	deviceCode, err :=
		parseIoTDeviceCode(request)

	if err != nil {
		writeIoTError(
			writer,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	var payload models.UpdateIoTConfigurationRequest

	if err := decodeIoTJSON(
		writer,
		request,
		&payload,
	); err != nil {
		writeIoTError(
			writer,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	if message :=
		validateIoTConfiguration(payload); message != "" {
		writeIoTError(
			writer,
			http.StatusBadRequest,
			message,
		)
		return
	}

	ctx, cancel := context.WithTimeout(
		request.Context(),
		15*time.Second,
	)
	defer cancel()

	configuration, err :=
		handler.repository.UpdateIoTConfiguration(
			ctx,
			deviceCode,
			payload,
		)

	switch {
	case errors.Is(
		err,
		repository.ErrIoTDeviceNotFound,
	):
		writeIoTError(
			writer,
			http.StatusNotFound,
			"dispositivo IoT no encontrado",
		)

	case errors.Is(
		err,
		repository.ErrIoTConfigurationNotFound,
	):
		writeIoTError(
			writer,
			http.StatusNotFound,
			"configuración IoT no encontrada",
		)

	case err != nil:
		log.Printf(
			"error actualizando configuración IoT de %s: %v",
			deviceCode,
			err,
		)

		writeIoTError(
			writer,
			http.StatusInternalServerError,
			"no fue posible actualizar la configuración IoT",
		)

	default:
		writeIoTResponse(
			writer,
			http.StatusOK,
			models.IoTConfigurationResponse{
				Status:        "ok",
				Configuracion: configuration,
			},
		)
	}
}

func parseIoTDeviceCode(
	request *http.Request,
) (
	string,
	error,
) {
	deviceCode := strings.ToUpper(
		strings.TrimSpace(
			request.URL.Query().Get(
				"dispositivo",
			),
		),
	)

	if deviceCode == "" {
		deviceCode =
			defaultIoTDeviceCode
	}

	if !iotDeviceCodePattern.MatchString(
		deviceCode,
	) {
		return "", errors.New(
			"código de dispositivo IoT inválido",
		)
	}

	return deviceCode, nil
}

func parseIoTLimit(
	request *http.Request,
	defaultLimit int,
) (
	int,
	error,
) {
	rawLimit := strings.TrimSpace(
		request.URL.Query().Get("limite"),
	)

	if rawLimit == "" {
		return defaultLimit, nil
	}

	limit, err := strconv.Atoi(
		rawLimit,
	)

	if err != nil ||
		limit < 1 ||
		limit > 200 {
		return 0, errors.New(
			"limite debe estar entre 1 y 200",
		)
	}

	return limit, nil
}

func decodeIoTJSON(
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

func validateIoTConfiguration(
	request models.UpdateIoTConfigurationRequest,
) string {
	if request.TemperaturaMinC < -50 ||
		request.TemperaturaMinC > 100 {
		return "temperatura_min_c debe estar entre -50 y 100"
	}

	if request.TemperaturaMaxC < -50 ||
		request.TemperaturaMaxC > 100 {
		return "temperatura_max_c debe estar entre -50 y 100"
	}

	if request.TemperaturaMinC >=
		request.TemperaturaMaxC {
		return "temperatura_min_c debe ser menor que temperatura_max_c"
	}

	if request.HumedadMinPct < 0 ||
		request.HumedadMinPct > 100 {
		return "humedad_min_pct debe estar entre 0 y 100"
	}

	if request.HumedadMaxPct < 0 ||
		request.HumedadMaxPct > 100 {
		return "humedad_max_pct debe estar entre 0 y 100"
	}

	if request.HumedadMinPct >=
		request.HumedadMaxPct {
		return "humedad_min_pct debe ser menor que humedad_max_pct"
	}

	if request.SegundosSinComunicacion < 30 ||
		request.SegundosSinComunicacion > 86400 {
		return "segundos_sin_comunicacion debe estar entre 30 y 86400"
	}

	return ""
}

func writeIoTResponse(
	writer http.ResponseWriter,
	statusCode int,
	payload any,
) {
	writer.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)

	writer.WriteHeader(statusCode)

	writeJSON(
		writer,
		payload,
	)
}

func writeIoTError(
	writer http.ResponseWriter,
	statusCode int,
	message string,
) {
	writeIoTResponse(
		writer,
		statusCode,
		ErrorResponse{
			Status:  "error",
			Message: message,
		},
	)
}
