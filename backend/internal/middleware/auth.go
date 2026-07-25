package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"sigefer.local/backend/internal/models"
	"sigefer.local/backend/internal/repository"
)

type authContextKey struct{}

// RequireAuthentication valida el token y,
// opcionalmente, los roles autorizados.
func RequireAuthentication(
	authRepository *repository.AuthRepository,
	allowedRoles ...string,
) func(http.Handler) http.Handler {
	roleSet := make(
		map[string]struct{},
		len(allowedRoles),
	)

	for _, role := range allowedRoles {
		roleSet[strings.ToUpper(role)] =
			struct{}{}
	}

	return func(
		next http.Handler,
	) http.Handler {
		return http.HandlerFunc(
			func(
				writer http.ResponseWriter,
				request *http.Request,
			) {
				token, valid :=
					extractBearerToken(request)

				if !valid {
					writeAuthError(
						writer,
						http.StatusUnauthorized,
						"se requiere una sesión válida",
					)
					return
				}

				ctx, cancel :=
					context.WithTimeout(
						request.Context(),
						10*time.Second,
					)
				defer cancel()

				principal, err :=
					authRepository.Authenticate(
						ctx,
						token,
					)

				if errors.Is(
					err,
					repository.ErrInvalidSession,
				) {
					writeAuthError(
						writer,
						http.StatusUnauthorized,
						"la sesión no es válida o ha expirado",
					)
					return
				}

				if err != nil {
					writeAuthError(
						writer,
						http.StatusInternalServerError,
						"no fue posible validar la sesión",
					)
					return
				}

				if len(roleSet) > 0 {
					if _, authorized :=
						roleSet[principal.Rol]; !authorized {
						writeAuthError(
							writer,
							http.StatusForbidden,
							"el usuario no tiene permiso para esta operación",
						)
						return
					}
				}

				requestContext :=
					context.WithValue(
						request.Context(),
						authContextKey{},
						principal,
					)

				next.ServeHTTP(
					writer,
					request.WithContext(
						requestContext,
					),
				)
			},
		)
	}
}

// PrincipalFromContext obtiene el usuario autenticado.
func PrincipalFromContext(
	ctx context.Context,
) (models.AuthPrincipal, bool) {
	principal, valid := ctx.Value(
		authContextKey{},
	).(models.AuthPrincipal)

	return principal, valid
}

func extractBearerToken(
	request *http.Request,
) (string, bool) {
	authorization := strings.TrimSpace(
		request.Header.Get("Authorization"),
	)

	parts := strings.Fields(authorization)

	if len(parts) != 2 ||
		!strings.EqualFold(
			parts[0],
			"Bearer",
		) ||
		len(parts[1]) < 32 ||
		len(parts[1]) > 512 {
		return "", false
	}

	return parts[1], true
}

func writeAuthError(
	writer http.ResponseWriter,
	statusCode int,
	message string,
) {
	writer.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)

	if statusCode ==
		http.StatusUnauthorized {
		writer.Header().Set(
			"WWW-Authenticate",
			"Bearer",
		)
	}

	writer.WriteHeader(statusCode)

	_ = json.NewEncoder(writer).Encode(
		map[string]string{
			"status":  "error",
			"message": message,
		},
	)
}
