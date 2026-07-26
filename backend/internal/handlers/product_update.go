package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/shopspring/decimal"

	"sigefer.local/backend/internal/middleware"
	"sigefer.local/backend/internal/models"
	"sigefer.local/backend/internal/repository"
)

// Update procesa PATCH /api/v1/productos/{codigo}.
func (handler *ProductHandler) Update(
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

	var payload models.UpdateProductRequest

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

	normalizeUpdateProduct(&payload)

	if message :=
		validateUpdateProduct(payload); message != "" {
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
		handler.repository.UpdateProduct(
			ctx,
			code,
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
		repository.ErrProductCategoryNotFound,
	):
		writeProductAdminError(
			writer,
			http.StatusBadRequest,
			"categoría no encontrada o inactiva",
		)

	case err != nil:
		log.Printf(
			"error actualizando producto %s: %v",
			code,
			err,
		)

		writeProductAdminError(
			writer,
			http.StatusInternalServerError,
			"no fue posible actualizar el producto",
		)

	default:
		writeJSON(writer, product)
	}
}

// UpdateState procesa
// PATCH /api/v1/productos/{codigo}/estado.
func (handler *ProductHandler) UpdateState(
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

	var payload models.UpdateProductStateRequest

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

	payload.Estado = strings.ToUpper(
		strings.TrimSpace(
			payload.Estado,
		),
	)

	if payload.Estado != "A" &&
		payload.Estado != "I" {
		writeProductAdminError(
			writer,
			http.StatusBadRequest,
			"estado debe ser A o I",
		)
		return
	}

	ctx, cancel := context.WithTimeout(
		request.Context(),
		20*time.Second,
	)
	defer cancel()

	product, err :=
		handler.repository.UpdateProductState(
			ctx,
			code,
			payload.Estado,
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

	case err != nil:
		log.Printf(
			"error cambiando estado del producto %s: %v",
			code,
			err,
		)

		writeProductAdminError(
			writer,
			http.StatusInternalServerError,
			"no fue posible cambiar el estado del producto",
		)

	default:
		writeJSON(writer, product)
	}
}

func normalizeUpdateProduct(
	request *models.UpdateProductRequest,
) {
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

func validateUpdateProduct(
	request models.UpdateProductRequest,
) string {
	if request.IDCategoria <= 0 {
		return "id_categoria debe ser mayor que cero"
	}

	if containsInvalidProductText(
		request.Nombre,
		request.Descripcion,
		request.UnidadMedida,
		request.Ubicacion,
	) {
		return "los campos de texto contienen caracteres inválidos"
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
		return "precio_compra debe ser mayor o igual que cero y admitir máximo 2 decimales"
	}

	if request.PrecioVenta.IsNegative() ||
		!validProductDecimalScale(
			request.PrecioVenta,
			2,
		) {
		return "precio_venta debe ser mayor o igual que cero y admitir máximo 2 decimales"
	}

	if request.PrecioVenta.LessThan(
		request.PrecioCompra,
	) {
		return "precio_venta no puede ser menor que precio_compra"
	}

	maximumPrice :=
		decimal.NewFromInt(
			999999999999,
		)

	if request.PrecioCompra.GreaterThan(
		maximumPrice,
	) ||
		request.PrecioVenta.GreaterThan(
			maximumPrice,
		) {
		return "los precios superan el máximo permitido"
	}

	if request.StockMinimo.IsNegative() ||
		!validProductDecimalScale(
			request.StockMinimo,
			3,
		) {
		return "stock_minimo debe ser mayor o igual que cero y admitir máximo 3 decimales"
	}

	if request.StockMinimo.GreaterThan(
		decimal.NewFromInt(
			99999999999,
		),
	) {
		return "stock_minimo supera el máximo permitido"
	}

	return ""
}

func containsInvalidProductText(
	values ...string,
) bool {
	for _, value := range values {
		if strings.ContainsRune(
			value,
			'\uFFFD',
		) {
			return true
		}
	}

	return false
}
