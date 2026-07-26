package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/shopspring/decimal"

	"sigefer.local/backend/internal/middleware"
	"sigefer.local/backend/internal/models"
	"sigefer.local/backend/internal/repository"
)

// BPMHandler procesa el BPM de reposición.
type BPMHandler struct {
	repository *repository.BPMRepository
}

// NewBPMHandler crea el handler BPM.
func NewBPMHandler(
	bpmRepository *repository.BPMRepository,
) *BPMHandler {
	return &BPMHandler{
		repository: bpmRepository,
	}
}

type bpmProviderListResponse struct {
	Status      string               `json:"status"`
	Total       int                  `json:"total"`
	Proveedores []models.BPMProvider `json:"proveedores"`
}

type bpmListResponse struct {
	Status       string                 `json:"status"`
	Total        int                    `json:"total"`
	FiltroEstado string                 `json:"filtro_estado,omitempty"`
	Reposiciones []models.Replenishment `json:"reposiciones"`
}

// ListProviders procesa GET /api/v1/bpm/proveedores.
func (handler *BPMHandler) ListProviders(
	writer http.ResponseWriter,
	request *http.Request,
) {
	ctx, cancel := context.WithTimeout(
		request.Context(),
		10*time.Second,
	)
	defer cancel()

	providers, err :=
		handler.repository.ListProviders(ctx)

	if err != nil {
		log.Printf(
			"error consultando proveedores BPM: %v",
			err,
		)

		writeBPMError(
			writer,
			http.StatusInternalServerError,
			"no fue posible consultar los proveedores",
		)
		return
	}

	writeJSON(
		writer,
		bpmProviderListResponse{
			Status:      "ok",
			Total:       len(providers),
			Proveedores: providers,
		},
	)
}

// List procesa GET /api/v1/bpm/reposiciones.
func (handler *BPMHandler) List(
	writer http.ResponseWriter,
	request *http.Request,
) {
	state := strings.ToUpper(
		strings.TrimSpace(
			request.URL.Query().Get("estado"),
		),
	)

	if state != "" &&
		!validBPMState(state) {
		writeBPMError(
			writer,
			http.StatusBadRequest,
			"estado BPM inválido",
		)
		return
	}

	limit := 100

	if text := strings.TrimSpace(
		request.URL.Query().Get("limite"),
	); text != "" {
		value, err := strconv.Atoi(text)

		if err != nil ||
			value < 1 ||
			value > 200 {
			writeBPMError(
				writer,
				http.StatusBadRequest,
				"limite debe estar entre 1 y 200",
			)
			return
		}

		limit = value
	}

	ctx, cancel := context.WithTimeout(
		request.Context(),
		10*time.Second,
	)
	defer cancel()

	items, err :=
		handler.repository.ListReplenishments(
			ctx,
			state,
			limit,
		)

	if err != nil {
		log.Printf(
			"error consultando reposiciones: %v",
			err,
		)

		writeBPMError(
			writer,
			http.StatusInternalServerError,
			"no fue posible consultar las reposiciones",
		)
		return
	}

	writeJSON(
		writer,
		bpmListResponse{
			Status:       "ok",
			Total:        len(items),
			FiltroEstado: state,
			Reposiciones: items,
		},
	)
}

// Get procesa GET /api/v1/bpm/reposiciones/{numero}.
func (handler *BPMHandler) Get(
	writer http.ResponseWriter,
	request *http.Request,
) {
	number := normalizeBPMNumber(
		request.PathValue("numero"),
	)

	if number == "" {
		writeBPMError(
			writer,
			http.StatusBadRequest,
			"número de solicitud inválido",
		)
		return
	}

	ctx, cancel := context.WithTimeout(
		request.Context(),
		10*time.Second,
	)
	defer cancel()

	item, err :=
		handler.repository.GetReplenishment(
			ctx,
			number,
		)

	if errors.Is(
		err,
		repository.ErrBPMRequestNotFound,
	) {
		writeBPMError(
			writer,
			http.StatusNotFound,
			"solicitud de reposición no encontrada",
		)
		return
	}

	if err != nil {
		log.Printf(
			"error consultando reposición %s: %v",
			number,
			err,
		)

		writeBPMError(
			writer,
			http.StatusInternalServerError,
			"no fue posible consultar la reposición",
		)
		return
	}

	writeJSON(writer, item)
}

// Create procesa POST /api/v1/bpm/reposiciones.
func (handler *BPMHandler) Create(
	writer http.ResponseWriter,
	request *http.Request,
) {
	principal, valid :=
		middleware.PrincipalFromContext(
			request.Context(),
		)

	if !valid {
		writeBPMError(
			writer,
			http.StatusUnauthorized,
			"sesión autenticada no disponible",
		)
		return
	}

	var payload models.CreateReplenishmentRequest

	if err := decodeBPMJSON(
		writer,
		request,
		&payload,
	); err != nil {
		writeBPMError(
			writer,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	payload.CodigoProducto = strings.ToUpper(
		strings.TrimSpace(
			payload.CodigoProducto,
		),
	)

	payload.Observacion = strings.TrimSpace(
		payload.Observacion,
	)

	if message := validateBPMCreate(
		payload,
	); message != "" {
		writeBPMError(
			writer,
			http.StatusBadRequest,
			message,
		)
		return
	}

	ctx, cancel := context.WithTimeout(
		request.Context(),
		20*time.Second,
	)
	defer cancel()

	item, err := handler.repository.CreateDraft(
		ctx,
		payload,
		principal.IDUsuario,
		clientIP(request),
	)

	switch {
	case errors.Is(
		err,
		repository.ErrBPMProductNotFound,
	):
		writeBPMError(
			writer,
			http.StatusBadRequest,
			"producto no encontrado o inactivo",
		)

	case errors.Is(
		err,
		repository.ErrBPMProviderNotFound,
	):
		writeBPMError(
			writer,
			http.StatusBadRequest,
			"proveedor no encontrado o inactivo",
		)

	case errors.Is(
		err,
		repository.ErrBPMAlertNotFound,
	):
		writeBPMError(
			writer,
			http.StatusBadRequest,
			"la alerta no corresponde al producto o no está pendiente",
		)

	case errors.Is(
		err,
		repository.ErrBPMActiveRequestExists,
	):
		writeBPMError(
			writer,
			http.StatusConflict,
			"ya existe una reposición activa para el producto",
		)

	case err != nil:
		log.Printf(
			"error creando reposición BPM: %v",
			err,
		)

		writeBPMError(
			writer,
			http.StatusInternalServerError,
			"no fue posible crear la reposición",
		)

	default:
		writer.Header().Set(
			"Location",
			"/api/v1/bpm/reposiciones/"+
				item.NumeroSolicitud,
		)

		writer.WriteHeader(http.StatusCreated)
		writeJSON(writer, item)
	}
}

// Send procesa PATCH .../{numero}/enviar.
func (handler *BPMHandler) Send(
	writer http.ResponseWriter,
	request *http.Request,
) {
	principal, valid :=
		middleware.PrincipalFromContext(
			request.Context(),
		)

	if !valid {
		writeBPMError(
			writer,
			http.StatusUnauthorized,
			"sesión autenticada no disponible",
		)
		return
	}

	number := normalizeBPMNumber(
		request.PathValue("numero"),
	)

	if number == "" {
		writeBPMError(
			writer,
			http.StatusBadRequest,
			"número de solicitud inválido",
		)
		return
	}

	var payload models.ReplenishmentTransitionRequest

	if err := decodeBPMJSON(
		writer,
		request,
		&payload,
	); err != nil {
		writeBPMError(
			writer,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	payload.Observacion = strings.TrimSpace(
		payload.Observacion,
	)

	if utf8.RuneCountInString(
		payload.Observacion,
	) > 1000 {
		writeBPMError(
			writer,
			http.StatusBadRequest,
			"observacion supera 1000 caracteres",
		)
		return
	}

	ctx, cancel := context.WithTimeout(
		request.Context(),
		20*time.Second,
	)
	defer cancel()

	item, err := handler.repository.Send(
		ctx,
		number,
		principal.IDUsuario,
		payload.Observacion,
		clientIP(request),
	)

	switch {
	case errors.Is(
		err,
		repository.ErrBPMRequestNotFound,
	):
		writeBPMError(
			writer,
			http.StatusNotFound,
			"solicitud de reposición no encontrada",
		)

	case errors.Is(
		err,
		repository.ErrBPMInvalidTransition,
	):
		writeBPMError(
			writer,
			http.StatusConflict,
			"solo un borrador puede enviarse",
		)

	case errors.Is(
		err,
		repository.ErrBPMAlertUnavailable,
	):
		writeBPMError(
			writer,
			http.StatusConflict,
			"la alerta vinculada ya no está pendiente",
		)

	case err != nil:
		log.Printf(
			"error enviando reposición %s: %v",
			number,
			err,
		)

		writeBPMError(
			writer,
			http.StatusInternalServerError,
			"no fue posible enviar la reposición",
		)

	default:
		writeJSON(writer, item)
	}
}

func decodeBPMJSON(
	writer http.ResponseWriter,
	request *http.Request,
	destination any,
) error {
	request.Body = http.MaxBytesReader(
		writer,
		request.Body,
		128*1024,
	)

	decoder := json.NewDecoder(request.Body)
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

func validateBPMCreate(
	request models.CreateReplenishmentRequest,
) string {
	if request.CodigoProducto == "" ||
		len(request.CodigoProducto) > 30 {
		return "codigo_producto es obligatorio y admite máximo 30 caracteres"
	}

	if request.IDProveedor <= 0 {
		return "id_proveedor debe ser mayor que cero"
	}

	if request.IDAlerta != nil &&
		*request.IDAlerta <= 0 {
		return "id_alerta debe ser mayor que cero"
	}

	if !request.CantidadSolicitada.
		GreaterThan(decimalZero()) {
		return "cantidad_solicitada debe ser mayor que cero"
	}

	if request.CantidadSolicitada.
		GreaterThan(decimalMaximumQuantity()) {
		return "cantidad_solicitada supera el máximo permitido"
	}

	if request.CostoUnitarioEstimado.
		IsNegative() {
		return "costo_unitario_estimado no puede ser negativo"
	}

	if request.CostoUnitarioEstimado.
		GreaterThan(decimalMaximumCost()) {
		return "costo_unitario_estimado supera el máximo permitido"
	}

	if utf8.RuneCountInString(
		request.Observacion,
	) > 1000 {
		return "observacion supera 1000 caracteres"
	}

	return ""
}

func validBPMState(state string) bool {
	switch state {
	case "BORRADOR",
		"SOLICITADA",
		"APROBADA",
		"RECHAZADA",
		"EN_PEDIDO",
		"RECIBIDA",
		"CERRADA":
		return true

	default:
		return false
	}
}

func normalizeBPMNumber(
	value string,
) string {
	value = strings.ToUpper(
		strings.TrimSpace(value),
	)

	if len(value) > 30 ||
		!strings.HasPrefix(
			value,
			"REP-",
		) {
		return ""
	}

	return value
}

func decimalZero() decimal.Decimal {
	return decimal.Zero
}

func decimalMaximumQuantity() decimal.Decimal {
	return decimal.NewFromInt(
		99999999999,
	)
}

func decimalMaximumCost() decimal.Decimal {
	return decimal.NewFromInt(
		999999999999,
	)
}

func writeBPMError(
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

// Approve procesa PATCH .../{numero}/aprobar.
func (handler *BPMHandler) Approve(
	writer http.ResponseWriter,
	request *http.Request,
) {
	principal, number, valid :=
		bpmPrincipalAndNumber(
			writer,
			request,
		)

	if !valid {
		return
	}

	var payload models.ReplenishmentTransitionRequest

	if err := decodeBPMJSON(
		writer,
		request,
		&payload,
	); err != nil {
		writeBPMError(
			writer,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	payload.Observacion = strings.TrimSpace(
		payload.Observacion,
	)

	if !validBPMObservation(
		payload.Observacion,
		1000,
	) {
		writeBPMError(
			writer,
			http.StatusBadRequest,
			"observacion supera 1000 caracteres",
		)
		return
	}

	ctx, cancel := context.WithTimeout(
		request.Context(),
		20*time.Second,
	)
	defer cancel()

	item, err := handler.repository.Approve(
		ctx,
		number,
		principal.IDUsuario,
		payload.Observacion,
		clientIP(request),
	)

	if handleBPMTransitionError(
		writer,
		err,
		"solo una solicitud enviada puede aprobarse",
	) {
		return
	}

	if err != nil {
		log.Printf(
			"error aprobando %s: %v",
			number,
			err,
		)

		writeBPMError(
			writer,
			http.StatusInternalServerError,
			"no fue posible aprobar la reposición",
		)
		return
	}

	writeJSON(writer, item)
}

// Reject procesa PATCH .../{numero}/rechazar.
func (handler *BPMHandler) Reject(
	writer http.ResponseWriter,
	request *http.Request,
) {
	principal, number, valid :=
		bpmPrincipalAndNumber(
			writer,
			request,
		)

	if !valid {
		return
	}

	var payload models.RejectReplenishmentRequest

	if err := decodeBPMJSON(
		writer,
		request,
		&payload,
	); err != nil {
		writeBPMError(
			writer,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	payload.MotivoRechazo =
		strings.TrimSpace(
			payload.MotivoRechazo,
		)

	length := utf8.RuneCountInString(
		payload.MotivoRechazo,
	)

	if length < 5 || length > 500 {
		writeBPMError(
			writer,
			http.StatusBadRequest,
			"motivo_rechazo debe contener entre 5 y 500 caracteres",
		)
		return
	}

	ctx, cancel := context.WithTimeout(
		request.Context(),
		20*time.Second,
	)
	defer cancel()

	item, err := handler.repository.Reject(
		ctx,
		number,
		principal.IDUsuario,
		payload.MotivoRechazo,
		clientIP(request),
	)

	if handleBPMTransitionError(
		writer,
		err,
		"solo una solicitud enviada puede rechazarse",
	) {
		return
	}

	if err != nil {
		log.Printf(
			"error rechazando %s: %v",
			number,
			err,
		)

		writeBPMError(
			writer,
			http.StatusInternalServerError,
			"no fue posible rechazar la reposición",
		)
		return
	}

	writeJSON(writer, item)
}

// MarkOrder procesa PATCH .../{numero}/pedido.
func (handler *BPMHandler) MarkOrder(
	writer http.ResponseWriter,
	request *http.Request,
) {
	principal, number, valid :=
		bpmPrincipalAndNumber(
			writer,
			request,
		)

	if !valid {
		return
	}

	var payload models.ReplenishmentTransitionRequest

	if err := decodeBPMJSON(
		writer,
		request,
		&payload,
	); err != nil {
		writeBPMError(
			writer,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	payload.Observacion = strings.TrimSpace(
		payload.Observacion,
	)

	if !validBPMObservation(
		payload.Observacion,
		1000,
	) {
		writeBPMError(
			writer,
			http.StatusBadRequest,
			"observacion supera 1000 caracteres",
		)
		return
	}

	ctx, cancel := context.WithTimeout(
		request.Context(),
		20*time.Second,
	)
	defer cancel()

	item, err := handler.repository.MarkOrder(
		ctx,
		number,
		principal.IDUsuario,
		payload.Observacion,
		clientIP(request),
	)

	if handleBPMTransitionError(
		writer,
		err,
		"solo una solicitud aprobada puede marcarse como pedido",
	) {
		return
	}

	if err != nil {
		log.Printf(
			"error registrando pedido %s: %v",
			number,
			err,
		)

		writeBPMError(
			writer,
			http.StatusInternalServerError,
			"no fue posible registrar el pedido",
		)
		return
	}

	writeJSON(writer, item)
}

// Receive procesa PATCH .../{numero}/recibir.
func (handler *BPMHandler) Receive(
	writer http.ResponseWriter,
	request *http.Request,
) {
	principal, number, valid :=
		bpmPrincipalAndNumber(
			writer,
			request,
		)

	if !valid {
		return
	}

	var payload models.ReceiveReplenishmentRequest

	if err := decodeBPMJSON(
		writer,
		request,
		&payload,
	); err != nil {
		writeBPMError(
			writer,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	payload.Observacion = strings.TrimSpace(
		payload.Observacion,
	)

	if !payload.CantidadRecibida.
		GreaterThan(decimal.Zero) {
		writeBPMError(
			writer,
			http.StatusBadRequest,
			"cantidad_recibida debe ser mayor que cero",
		)
		return
	}

	if payload.CantidadRecibida.
		GreaterThan(
			decimalMaximumQuantity(),
		) {
		writeBPMError(
			writer,
			http.StatusBadRequest,
			"cantidad_recibida supera el máximo permitido",
		)
		return
	}

	if !validBPMObservation(
		payload.Observacion,
		1000,
	) {
		writeBPMError(
			writer,
			http.StatusBadRequest,
			"observacion supera 1000 caracteres",
		)
		return
	}

	ctx, cancel := context.WithTimeout(
		request.Context(),
		30*time.Second,
	)
	defer cancel()

	item, err := handler.repository.Receive(
		ctx,
		number,
		principal.IDUsuario,
		payload.CantidadRecibida,
		payload.Observacion,
		clientIP(request),
	)

	switch {
	case errors.Is(
		err,
		repository.ErrBPMRequestNotFound,
	):
		writeBPMError(
			writer,
			http.StatusNotFound,
			"solicitud de reposición no encontrada",
		)
		return

	case errors.Is(
		err,
		repository.ErrBPMInvalidTransition,
	):
		writeBPMError(
			writer,
			http.StatusConflict,
			"solo una solicitud en pedido puede recibirse",
		)
		return

	case errors.Is(
		err,
		repository.ErrBPMInvalidReceivedQuantity,
	):
		writeBPMError(
			writer,
			http.StatusBadRequest,
			"cantidad_recibida debe ser mayor que cero y no superar la cantidad solicitada",
		)
		return

	case errors.Is(
		err,
		repository.ErrBPMInventoryNotFound,
	):
		writeBPMError(
			writer,
			http.StatusConflict,
			"el producto no tiene inventario disponible",
		)
		return

	case err != nil:
		log.Printf(
			"error recibiendo reposición %s: %v",
			number,
			err,
		)

		writeBPMError(
			writer,
			http.StatusInternalServerError,
			"no fue posible registrar la recepción",
		)
		return
	}

	writeJSON(writer, item)
}

// Close procesa PATCH .../{numero}/cerrar.
func (handler *BPMHandler) Close(
	writer http.ResponseWriter,
	request *http.Request,
) {
	principal, number, valid :=
		bpmPrincipalAndNumber(
			writer,
			request,
		)

	if !valid {
		return
	}

	var payload models.ReplenishmentTransitionRequest

	if err := decodeBPMJSON(
		writer,
		request,
		&payload,
	); err != nil {
		writeBPMError(
			writer,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	payload.Observacion = strings.TrimSpace(
		payload.Observacion,
	)

	if !validBPMObservation(
		payload.Observacion,
		1000,
	) {
		writeBPMError(
			writer,
			http.StatusBadRequest,
			"observacion supera 1000 caracteres",
		)
		return
	}

	ctx, cancel := context.WithTimeout(
		request.Context(),
		20*time.Second,
	)
	defer cancel()

	item, err := handler.repository.Close(
		ctx,
		number,
		principal.IDUsuario,
		payload.Observacion,
		clientIP(request),
	)

	if handleBPMTransitionError(
		writer,
		err,
		"solo una recepción confirmada puede cerrarse",
	) {
		return
	}

	if err != nil {
		log.Printf(
			"error cerrando reposición %s: %v",
			number,
			err,
		)

		writeBPMError(
			writer,
			http.StatusInternalServerError,
			"no fue posible cerrar la reposición",
		)
		return
	}

	writeJSON(writer, item)
}

func bpmPrincipalAndNumber(
	writer http.ResponseWriter,
	request *http.Request,
) (models.AuthPrincipal, string, bool) {
	principal, valid :=
		middleware.PrincipalFromContext(
			request.Context(),
		)

	if !valid {
		writeBPMError(
			writer,
			http.StatusUnauthorized,
			"sesión autenticada no disponible",
		)

		return models.AuthPrincipal{},
			"",
			false
	}

	number := normalizeBPMNumber(
		request.PathValue("numero"),
	)

	if number == "" {
		writeBPMError(
			writer,
			http.StatusBadRequest,
			"número de solicitud inválido",
		)

		return models.AuthPrincipal{},
			"",
			false
	}

	return principal, number, true
}

func validBPMObservation(
	value string,
	maximum int,
) bool {
	return utf8.RuneCountInString(
		value,
	) <= maximum
}

func handleBPMTransitionError(
	writer http.ResponseWriter,
	err error,
	transitionMessage string,
) bool {
	switch {
	case errors.Is(
		err,
		repository.ErrBPMRequestNotFound,
	):
		writeBPMError(
			writer,
			http.StatusNotFound,
			"solicitud de reposición no encontrada",
		)
		return true

	case errors.Is(
		err,
		repository.ErrBPMInvalidTransition,
	):
		writeBPMError(
			writer,
			http.StatusConflict,
			transitionMessage,
		)
		return true

	default:
		return false
	}
}
