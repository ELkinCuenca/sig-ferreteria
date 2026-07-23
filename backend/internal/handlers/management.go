package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"sigefer.local/backend/internal/models"
	"sigefer.local/backend/internal/repository"
)

// ManagementHandler procesa las consultas gerenciales.
type ManagementHandler struct {
	repository *repository.ManagementRepository
}

// NewManagementHandler crea el handler gerencial.
func NewManagementHandler(
	managementRepository *repository.ManagementRepository,
) *ManagementHandler {
	return &ManagementHandler{
		repository: managementRepository,
	}
}

// ListSales procesa GET /api/v1/ventas.
func (handler *ManagementHandler) ListSales(
	writer http.ResponseWriter,
	request *http.Request,
) {
	limit, err := parseLimit(request)
	if err != nil {
		writeManagementError(
			writer,
			http.StatusBadRequest,
			"limite debe ser un número entre 1 y 200",
		)
		return
	}

	ctx, cancel := context.WithTimeout(
		request.Context(),
		10*time.Second,
	)
	defer cancel()

	sales, err := handler.repository.ListSales(
		ctx,
		limit,
	)
	if err != nil {
		log.Printf(
			"error consultando ventas: %v",
			err,
		)

		writeManagementError(
			writer,
			http.StatusInternalServerError,
			"no fue posible consultar las ventas",
		)
		return
	}

	writer.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)
	writer.WriteHeader(http.StatusOK)

	writeJSON(
		writer,
		models.SaleListResponse{
			Status: "ok",
			Total:  len(sales),
			Ventas: sales,
		},
	)
}

// GetSale procesa GET /api/v1/ventas/{numero}.
func (handler *ManagementHandler) GetSale(
	writer http.ResponseWriter,
	request *http.Request,
) {
	saleNumber := strings.TrimSpace(
		request.PathValue("numero"),
	)

	if saleNumber == "" || len(saleNumber) > 40 {
		writeManagementError(
			writer,
			http.StatusBadRequest,
			"número de venta inválido",
		)
		return
	}

	ctx, cancel := context.WithTimeout(
		request.Context(),
		10*time.Second,
	)
	defer cancel()

	sale, err := handler.repository.FindSaleByNumber(
		ctx,
		saleNumber,
	)
	if errors.Is(err, repository.ErrSaleNotFound) {
		writeManagementError(
			writer,
			http.StatusNotFound,
			"venta no encontrada",
		)
		return
	}

	if err != nil {
		log.Printf(
			"error consultando venta: %v",
			err,
		)

		writeManagementError(
			writer,
			http.StatusInternalServerError,
			"no fue posible consultar la venta",
		)
		return
	}

	writer.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)
	writer.WriteHeader(http.StatusOK)
	writeJSON(writer, sale)
}

// ListAlerts procesa GET /api/v1/alertas-stock.
func (handler *ManagementHandler) ListAlerts(
	writer http.ResponseWriter,
	request *http.Request,
) {
	status := strings.ToUpper(
		strings.TrimSpace(
			request.URL.Query().Get("estado"),
		),
	)

	allowedStatuses := map[string]bool{
		"":           true,
		"PENDIENTE":  true,
		"ATENDIDA":   true,
		"DESCARTADA": true,
	}

	if !allowedStatuses[status] {
		writeManagementError(
			writer,
			http.StatusBadRequest,
			"estado debe ser PENDIENTE, ATENDIDA o DESCARTADA",
		)
		return
	}

	ctx, cancel := context.WithTimeout(
		request.Context(),
		10*time.Second,
	)
	defer cancel()

	alerts, err := handler.repository.ListStockAlerts(
		ctx,
		status,
	)
	if err != nil {
		log.Printf(
			"error consultando alertas: %v",
			err,
		)

		writeManagementError(
			writer,
			http.StatusInternalServerError,
			"no fue posible consultar las alertas",
		)
		return
	}

	writer.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)
	writer.WriteHeader(http.StatusOK)

	writeJSON(
		writer,
		models.StockAlertListResponse{
			Status:  "ok",
			Total:   len(alerts),
			Estado:  status,
			Alertas: alerts,
		},
	)
}

// Dashboard procesa GET /api/v1/dashboard/resumen.
func (handler *ManagementHandler) Dashboard(
	writer http.ResponseWriter,
	request *http.Request,
) {
	ctx, cancel := context.WithTimeout(
		request.Context(),
		10*time.Second,
	)
	defer cancel()

	dashboard, err := handler.repository.Dashboard(ctx)
	if err != nil {
		log.Printf(
			"error generando dashboard: %v",
			err,
		)

		writeManagementError(
			writer,
			http.StatusInternalServerError,
			"no fue posible generar el resumen gerencial",
		)
		return
	}

	writer.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)
	writer.WriteHeader(http.StatusOK)
	writeJSON(writer, dashboard)
}

func parseLimit(request *http.Request) (int, error) {
	value := request.URL.Query().Get("limite")

	if value == "" {
		return 50, nil
	}

	limit, err := strconv.Atoi(value)
	if err != nil || limit < 1 || limit > 200 {
		return 0, errors.New("límite inválido")
	}

	return limit, nil
}

func writeManagementError(
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
