package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"sigefer.local/backend/internal/models"
	"sigefer.local/backend/internal/repository"
)

// SaleHandler administra las operaciones HTTP de ventas.
type SaleHandler struct {
	repository *repository.SaleRepository
}

// NewSaleHandler crea el handler de ventas.
func NewSaleHandler(
	saleRepository *repository.SaleRepository,
) *SaleHandler {
	return &SaleHandler{
		repository: saleRepository,
	}
}

// Create procesa POST /api/v1/ventas.
func (handler *SaleHandler) Create(
	writer http.ResponseWriter,
	request *http.Request,
) {
	writer.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)

	if request.Method != http.MethodPost {
		writer.WriteHeader(http.StatusMethodNotAllowed)

		writeJSON(
			writer,
			ErrorResponse{
				Status:  "error",
				Message: "método HTTP no permitido",
			},
		)
		return
	}

	request.Body = http.MaxBytesReader(
		writer,
		request.Body,
		1024*1024,
	)

	var payload models.SaleCreateRequest

	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&payload); err != nil {
		writeSaleError(
			writer,
			http.StatusBadRequest,
			"cuerpo JSON inválido",
		)
		return
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(
		err,
		io.EOF,
	) {
		writeSaleError(
			writer,
			http.StatusBadRequest,
			"el cuerpo debe contener un solo objeto JSON",
		)
		return
	}

	if message := validateSaleRequest(&payload); message != "" {
		writeSaleError(
			writer,
			http.StatusBadRequest,
			message,
		)
		return
	}

	ctx, cancel := context.WithTimeout(
		request.Context(),
		30*time.Second,
	)
	defer cancel()

	result, err := handler.repository.Create(
		ctx,
		payload,
		clientIP(request),
	)
	if err != nil {
		handleRepositorySaleError(writer, err)
		return
	}

	writer.Header().Set(
		"Location",
		"/api/v1/ventas/"+result.NumeroVenta,
	)
	writer.WriteHeader(http.StatusCreated)
	writeJSON(writer, result)
}

func validateSaleRequest(
	request *models.SaleCreateRequest,
) string {
	request.IdentificacionCliente = strings.TrimSpace(
		request.IdentificacionCliente,
	)

	request.MetodoPago = strings.ToUpper(
		strings.TrimSpace(request.MetodoPago),
	)

	request.Observacion = strings.TrimSpace(
		request.Observacion,
	)

	if request.IdentificacionCliente == "" {
		return "identificacion_cliente es obligatoria"
	}

	if len(request.IdentificacionCliente) > 20 {
		return "identificacion_cliente supera 20 caracteres"
	}

	if request.IDUsuario != nil && *request.IDUsuario <= 0 {
		return "id_usuario debe ser mayor que cero"
	}

	if request.MetodoPago == "" {
		request.MetodoPago = "EFECTIVO"
	}

	allowedMethods := map[string]bool{
		"EFECTIVO":      true,
		"TARJETA":       true,
		"TRANSFERENCIA": true,
		"MIXTO":         true,
	}

	if !allowedMethods[request.MetodoPago] {
		return "metodo_pago no es válido"
	}

	if len(request.Observacion) > 500 {
		return "observacion supera 500 caracteres"
	}

	if request.DescuentoGeneral.IsNegative() {
		return "descuento_general no puede ser negativo"
	}

	if !request.DescuentoGeneral.Equal(
		request.DescuentoGeneral.Round(2),
	) {
		return "descuento_general admite máximo dos decimales"
	}

	if len(request.Items) == 0 {
		return "la venta debe contener al menos un producto"
	}

	if len(request.Items) > 50 {
		return "la venta admite máximo 50 productos"
	}

	codes := make(map[string]struct{}, len(request.Items))

	for index := range request.Items {
		item := &request.Items[index]

		item.CodigoProducto = strings.ToUpper(
			strings.TrimSpace(item.CodigoProducto),
		)

		if item.CodigoProducto == "" {
			return "todos los productos requieren codigo_producto"
		}

		if len(item.CodigoProducto) > 30 {
			return "codigo_producto supera 30 caracteres"
		}

		if _, exists := codes[item.CodigoProducto]; exists {
			return "no se permiten productos duplicados"
		}

		codes[item.CodigoProducto] = struct{}{}

		if item.Cantidad.LessThanOrEqual(decimal.Zero) {
			return "cantidad debe ser mayor que cero"
		}

		if !item.Cantidad.Equal(
			item.Cantidad.Round(3),
		) {
			return "cantidad admite máximo tres decimales"
		}

		if item.Descuento.IsNegative() {
			return "el descuento de una línea no puede ser negativo"
		}

		if !item.Descuento.Equal(
			item.Descuento.Round(2),
		) {
			return "el descuento de una línea admite máximo dos decimales"
		}
	}

	return ""
}

func handleRepositorySaleError(
	writer http.ResponseWriter,
	err error,
) {
	var productNotFound *repository.ProductNotFoundError
	var insufficientStock *repository.InsufficientStockError
	var invalidDiscount *repository.InvalidDiscountError

	switch {
	case errors.Is(err, repository.ErrClientNotFound):
		writeSaleError(
			writer,
			http.StatusNotFound,
			err.Error(),
		)

	case errors.Is(err, repository.ErrUserNotFound):
		writeSaleError(
			writer,
			http.StatusNotFound,
			err.Error(),
		)

	case errors.As(err, &productNotFound):
		writeSaleError(
			writer,
			http.StatusNotFound,
			err.Error(),
		)

	case errors.As(err, &insufficientStock):
		writeSaleError(
			writer,
			http.StatusConflict,
			err.Error(),
		)

	case errors.As(err, &invalidDiscount):
		writeSaleError(
			writer,
			http.StatusBadRequest,
			err.Error(),
		)

	default:
		log.Printf(
			"error registrando venta: %v",
			err,
		)

		writeSaleError(
			writer,
			http.StatusInternalServerError,
			"no fue posible registrar la venta",
		)
	}
}

func writeSaleError(
	writer http.ResponseWriter,
	statusCode int,
	message string,
) {
	writer.WriteHeader(statusCode)

	writeJSON(
		writer,
		ErrorResponse{
			Status:  "error",
			Message: message,
		},
	)
}

func clientIP(request *http.Request) string {
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err == nil {
		return host
	}

	return request.RemoteAddr
}
