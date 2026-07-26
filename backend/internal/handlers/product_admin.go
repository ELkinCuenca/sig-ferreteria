package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/shopspring/decimal"

	"sigefer.local/backend/internal/middleware"
	"sigefer.local/backend/internal/models"
	"sigefer.local/backend/internal/repository"
)

var productCodePattern = regexp.MustCompile(
	`^[A-Z0-9][A-Z0-9._-]{0,29}$`,
)

type CategoryListResponse struct {
	Status     string            `json:"status"`
	Total      int               `json:"total"`
	Categories []models.Category `json:"categorias"`
}

// ListCategories procesa GET /api/v1/categorias.
func (handler *ProductHandler) ListCategories(
	writer http.ResponseWriter,
	request *http.Request,
) {
	ctx, cancel := context.WithTimeout(
		request.Context(),
		10*time.Second,
	)
	defer cancel()

	categories, err :=
		handler.repository.ListCategories(ctx)

	if err != nil {
		log.Printf(
			"error consultando categorías: %v",
			err,
		)

		writeProductAdminError(
			writer,
			http.StatusInternalServerError,
			"no fue posible consultar las categorías",
		)
		return
	}

	writeJSON(
		writer,
		CategoryListResponse{
			Status:     "ok",
			Total:      len(categories),
			Categories: categories,
		},
	)
}

// Get procesa GET /api/v1/productos/{codigo}.
func (handler *ProductHandler) Get(
	writer http.ResponseWriter,
	request *http.Request,
) {
	code := normalizeProductCode(
		request.PathValue("codigo"),
	)

	if !productCodePattern.MatchString(code) {
		writeProductAdminError(
			writer,
			http.StatusBadRequest,
			"código de producto inválido",
		)
		return
	}

	ctx, cancel := context.WithTimeout(
		request.Context(),
		10*time.Second,
	)
	defer cancel()

	product, err :=
		handler.repository.GetByCode(
			ctx,
			code,
		)

	if errors.Is(
		err,
		repository.ErrProductNotFound,
	) {
		writeProductAdminError(
			writer,
			http.StatusNotFound,
			"producto no encontrado",
		)
		return
	}

	if err != nil {
		log.Printf(
			"error consultando producto %s: %v",
			code,
			err,
		)

		writeProductAdminError(
			writer,
			http.StatusInternalServerError,
			"no fue posible consultar el producto",
		)
		return
	}

	writeJSON(writer, product)
}

// Create procesa POST /api/v1/productos.
func (handler *ProductHandler) Create(
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

	var payload models.CreateProductRequest

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

	normalizeCreateProduct(&payload)

	if message :=
		validateCreateProduct(payload); message != "" {
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

	product, err :=
		handler.repository.CreateProduct(
			ctx,
			payload,
			principal.IDUsuario,
			clientIP(request),
		)

	switch {
	case errors.Is(
		err,
		repository.ErrProductCategoryNotFound,
	):
		writeProductAdminError(
			writer,
			http.StatusBadRequest,
			"categoría no encontrada o inactiva",
		)

	case errors.Is(
		err,
		repository.ErrProductCodeExists,
	):
		writeProductAdminError(
			writer,
			http.StatusConflict,
			"el código de producto ya está registrado",
		)

	case err != nil:
		log.Printf(
			"error creando producto %s: %v",
			payload.Codigo,
			err,
		)

		writeProductAdminError(
			writer,
			http.StatusInternalServerError,
			"no fue posible crear el producto",
		)

	default:
		writer.Header().Set(
			"Location",
			"/api/v1/productos/"+
				product.Codigo,
		)

		writer.WriteHeader(
			http.StatusCreated,
		)

		writeJSON(writer, product)
	}
}

func normalizeCreateProduct(
	request *models.CreateProductRequest,
) {
	request.Codigo =
		normalizeProductCode(
			request.Codigo,
		)

	request.Nombre =
		strings.TrimSpace(
			request.Nombre,
		)

	request.Descripcion =
		strings.TrimSpace(
			request.Descripcion,
		)

	request.UnidadMedida =
		strings.ToUpper(
			strings.TrimSpace(
				request.UnidadMedida,
			),
		)

	request.Ubicacion =
		strings.TrimSpace(
			request.Ubicacion,
		)
}

func validateCreateProduct(
	request models.CreateProductRequest,
) string {
	if request.IDCategoria <= 0 {
		return "id_categoria debe ser mayor que cero"
	}

	if !productCodePattern.MatchString(
		request.Codigo,
	) {
		return "codigo solo admite letras, números, punto, guion y guion bajo"
	}

	if length := utf8.RuneCountInString(
		request.Nombre,
	); length < 2 || length > 150 {
		return "nombre debe contener entre 2 y 150 caracteres"
	}

	if utf8.RuneCountInString(
		request.Descripcion,
	) > 500 {
		return "descripcion supera 500 caracteres"
	}

	if length := utf8.RuneCountInString(
		request.UnidadMedida,
	); length < 1 || length > 30 {
		return "unidad_medida es obligatoria y admite máximo 30 caracteres"
	}

	if utf8.RuneCountInString(
		request.Ubicacion,
	) > 100 {
		return "ubicacion supera 100 caracteres"
	}

	if request.PrecioCompra.IsNegative() ||
		!validProductDecimalScale(
			request.PrecioCompra,
			2,
		) {
		return "precio_compra debe ser positivo y admitir máximo 2 decimales"
	}

	if request.PrecioVenta.IsNegative() ||
		!validProductDecimalScale(
			request.PrecioVenta,
			2,
		) {
		return "precio_venta debe ser positivo y admitir máximo 2 decimales"
	}

	if request.PrecioVenta.LessThan(
		request.PrecioCompra,
	) {
		return "precio_venta no puede ser menor que precio_compra"
	}

	if request.PrecioCompra.GreaterThan(
		decimal.NewFromInt(
			999999999999,
		),
	) ||
		request.PrecioVenta.GreaterThan(
			decimal.NewFromInt(
				999999999999,
			),
		) {
		return "los precios superan el máximo permitido"
	}

	if request.StockMinimo.IsNegative() ||
		!validProductDecimalScale(
			request.StockMinimo,
			3,
		) {
		return "stock_minimo debe ser positivo y admitir máximo 3 decimales"
	}

	if request.StockInicial.IsNegative() ||
		!validProductDecimalScale(
			request.StockInicial,
			3,
		) {
		return "stock_inicial debe ser positivo y admitir máximo 3 decimales"
	}

	maximumStock := decimal.NewFromInt(
		99999999999,
	)

	if request.StockMinimo.GreaterThan(
		maximumStock,
	) ||
		request.StockInicial.GreaterThan(
			maximumStock,
		) {
		return "las cantidades de stock superan el máximo permitido"
	}

	return ""
}

func validProductDecimalScale(
	value decimal.Decimal,
	scale int32,
) bool {
	return value.Equal(
		value.Truncate(scale),
	)
}

func normalizeProductCode(
	value string,
) string {
	return strings.ToUpper(
		strings.TrimSpace(value),
	)
}

func decodeProductAdminJSON(
	writer http.ResponseWriter,
	request *http.Request,
	destination any,
) error {
	request.Body = http.MaxBytesReader(
		writer,
		request.Body,
		128*1024,
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

func writeProductAdminError(
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
