package handlers

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"sigefer.local/backend/internal/models"
	"sigefer.local/backend/internal/repository"
)

// ProductHandler contiene las operaciones HTTP de productos.
type ProductHandler struct {
	repository *repository.ProductRepository
}

// ProductListResponse representa la respuesta del listado.
type ProductListResponse struct {
	Status    string           `json:"status"`
	Total     int              `json:"total"`
	StockBajo bool             `json:"filtro_stock_bajo"`
	Products  []models.Product `json:"productos"`
	Message   string           `json:"message,omitempty"`
}

// ErrorResponse representa una respuesta de error.
type ErrorResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// NewProductHandler crea el handler de productos.
func NewProductHandler(
	productRepository *repository.ProductRepository,
) *ProductHandler {
	return &ProductHandler{
		repository: productRepository,
	}
}

// List procesa GET /api/v1/productos.
func (handler *ProductHandler) List(
	writer http.ResponseWriter,
	request *http.Request,
) {
	writer.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)

	if request.Method != http.MethodGet {
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

	lowStockOnly, err := parseLowStockFilter(request)
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)

		writeJSON(
			writer,
			ErrorResponse{
				Status: "error",
				Message: "el parámetro stock_bajo debe ser " +
					"true o false",
			},
		)
		return
	}

	ctx, cancel := context.WithTimeout(
		request.Context(),
		10*time.Second,
	)
	defer cancel()

	products, err := handler.repository.List(
		ctx,
		lowStockOnly,
	)
	if err != nil {
		log.Printf(
			"error consultando productos: %v",
			err,
		)

		writer.WriteHeader(
			http.StatusInternalServerError,
		)

		writeJSON(
			writer,
			ErrorResponse{
				Status: "error",
				Message: "no fue posible consultar " +
					"los productos",
			},
		)
		return
	}

	writer.WriteHeader(http.StatusOK)

	writeJSON(
		writer,
		ProductListResponse{
			Status:    "ok",
			Total:     len(products),
			StockBajo: lowStockOnly,
			Products:  products,
		},
	)
}

func parseLowStockFilter(
	request *http.Request,
) (bool, error) {
	value := request.URL.Query().Get("stock_bajo")

	if value == "" {
		return false, nil
	}

	return strconv.ParseBool(value)
}
