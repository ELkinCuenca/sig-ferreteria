package handlers

import (
	"context"
	"errors"
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

type InventoryMovementListResponse struct {
	Status    string                     `json:"status"`
	Total     int                        `json:"total"`
	Movements []models.InventoryMovement `json:"movimientos"`
}

// ListInventoryMovements procesa
// GET /api/v1/inventario/movimientos.
func (handler *ProductHandler) ListInventoryMovements(
	writer http.ResponseWriter,
	request *http.Request,
) {
	limit := 50

	if rawLimit := strings.TrimSpace(
		request.URL.Query().Get("limite"),
	); rawLimit != "" {
		parsedLimit, err :=
			strconv.Atoi(rawLimit)

		if err != nil ||
			parsedLimit < 1 ||
			parsedLimit > 200 {
			writeProductAdminError(
				writer,
				http.StatusBadRequest,
				"limite debe estar entre 1 y 200",
			)
			return
		}

		limit = parsedLimit
	}

	code := normalizeProductCode(
		request.URL.Query().Get("codigo"),
	)

	if code != "" &&
		!productCodePattern.MatchString(code) {
		writeProductAdminError(
			writer,
			http.StatusBadRequest,
			"código de producto inválido",
		)
		return
	}

	movementType := strings.ToUpper(
		strings.TrimSpace(
			request.URL.Query().Get("tipo"),
		),
	)

	if movementType != "" &&
		!validInventoryMovementType(
			movementType,
		) {
		writeProductAdminError(
			writer,
			http.StatusBadRequest,
			"tipo de movimiento no permitido",
		)
		return
	}

	ctx, cancel := context.WithTimeout(
		request.Context(),
		15*time.Second,
	)
	defer cancel()

	movements, err :=
		handler.repository.ListInventoryMovements(
			ctx,
			limit,
			code,
			movementType,
		)

	if err != nil {
		log.Printf(
			"error consultando movimientos: %v",
			err,
		)

		writeProductAdminError(
			writer,
			http.StatusInternalServerError,
			"no fue posible consultar los movimientos",
		)
		return
	}

	writeJSON(
		writer,
		InventoryMovementListResponse{
			Status:    "ok",
			Total:     len(movements),
			Movements: movements,
		},
	)
}

// AdjustInventory procesa
// POST /api/v1/inventario/ajustes.
func (handler *ProductHandler) AdjustInventory(
	writer http.ResponseWriter,
	request *http.Request,
) {
	principal, valid :=
		middleware.PrincipalFromContext(
			request.Context(),
		)

	if !valid {
		writeProductAdminError(
			writer,
			http.StatusUnauthorized,
			"sesión autenticada no disponible",
		)
		return
	}

	var payload models.InventoryAdjustmentRequest

	if err := decodeProductAdminJSON(
		writer,
		request,
		&payload,
	); err != nil {
		writeProductAdminError(
			writer,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	normalizeInventoryAdjustment(
		&payload,
	)

	if message :=
		validateInventoryAdjustment(
			payload,
		); message != "" {
		writeProductAdminError(
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

	result, err :=
		handler.repository.AdjustInventory(
			ctx,
			payload,
			principal.IDUsuario,
			clientIP(request),
		)

	switch {
	case errors.Is(
		err,
		repository.ErrProductNotFound,
	):
		writeProductAdminError(
			writer,
			http.StatusNotFound,
			"producto no encontrado",
		)

	case errors.Is(
		err,
		repository.ErrInventoryProductInactive,
	):
		writeProductAdminError(
			writer,
			http.StatusConflict,
			"no se puede ajustar un producto inactivo",
		)

	case errors.Is(
		err,
		repository.ErrInventoryReservedConflict,
	):
		writeProductAdminError(
			writer,
			http.StatusConflict,
			"el ajuste dejaría el stock por debajo de las unidades reservadas",
		)

	case errors.Is(
		err,
		repository.ErrInventoryAdjustmentType,
	):
		writeProductAdminError(
			writer,
			http.StatusBadRequest,
			"tipo de ajuste no permitido",
		)

	case err != nil:
		log.Printf(
			"error ajustando inventario de %s: %v",
			payload.CodigoProducto,
			err,
		)

		writeProductAdminError(
			writer,
			http.StatusInternalServerError,
			"no fue posible registrar el ajuste",
		)

	default:
		writer.WriteHeader(
			http.StatusCreated,
		)

		writeJSON(writer, result)
	}
}

func normalizeInventoryAdjustment(
	request *models.InventoryAdjustmentRequest,
) {
	request.CodigoProducto =
		normalizeProductCode(
			request.CodigoProducto,
		)

	request.TipoAjuste =
		strings.ToUpper(
			strings.TrimSpace(
				request.TipoAjuste,
			),
		)

	request.Motivo =
		strings.TrimSpace(
			request.Motivo,
		)
}

func validateInventoryAdjustment(
	request models.InventoryAdjustmentRequest,
) string {
	if !productCodePattern.MatchString(
		request.CodigoProducto,
	) {
		return "codigo_producto es inválido"
	}

	if request.TipoAjuste != "POSITIVO" &&
		request.TipoAjuste != "NEGATIVO" {
		return "tipo_ajuste debe ser POSITIVO o NEGATIVO"
	}

	if request.Cantidad.IsZero() ||
		request.Cantidad.IsNegative() {
		return "cantidad debe ser mayor que cero"
	}

	if !validProductDecimalScale(
		request.Cantidad,
		3,
	) {
		return "cantidad admite máximo 3 decimales"
	}

	maximumQuantity := decimal.New(
		99999999999999,
		-3,
	)

	if request.Cantidad.GreaterThan(
		maximumQuantity,
	) {
		return "cantidad supera el máximo permitido"
	}

	if strings.ContainsRune(
		request.Motivo,
		'\uFFFD',
	) {
		return "motivo contiene caracteres inválidos"
	}

	reasonLength :=
		utf8.RuneCountInString(
			request.Motivo,
		)

	if reasonLength < 5 ||
		reasonLength > 300 {
		return "motivo debe contener entre 5 y 300 caracteres"
	}

	return ""
}

func validInventoryMovementType(
	value string,
) bool {
	switch value {
	case "ENTRADA_COMPRA",
		"SALIDA_VENTA",
		"AJUSTE_POSITIVO",
		"AJUSTE_NEGATIVO",
		"DEVOLUCION_VENTA",
		"DEVOLUCION_COMPRA":
		return true

	default:
		return false
	}
}
