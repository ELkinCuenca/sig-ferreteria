package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"
)

const maxJSONBodySize int64 = 2 * 1024 * 1024

type errorResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// RejectCorruptUnicode valida globalmente los cuerpos JSON
// enviados mediante POST, PUT y PATCH.
func RejectCorruptUnicode(
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(
		func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			if !mustValidateBody(request) {
				next.ServeHTTP(writer, request)
				return
			}

			body, err := io.ReadAll(
				io.LimitReader(
					request.Body,
					maxJSONBodySize+1,
				),
			)
			if err != nil {
				writeUnicodeError(
					writer,
					http.StatusBadRequest,
					"no fue posible leer el cuerpo JSON",
				)
				return
			}

			_ = request.Body.Close()

			if int64(len(body)) >
				maxJSONBodySize {
				writeUnicodeError(
					writer,
					http.StatusRequestEntityTooLarge,
					"el cuerpo JSON supera el tamaño permitido",
				)
				return
			}

			// Restaurar el cuerpo para que el handler pueda leerlo.
			request.Body = io.NopCloser(
				bytes.NewReader(body),
			)

			lowerBody := bytes.ToLower(body)

			hasReplacementCharacter :=
				bytes.Contains(
					body,
					[]byte("\uFFFD"),
				) ||
					bytes.Contains(
						lowerBody,
						[]byte(`\ufffd`),
					)

			if !utf8.Valid(body) ||
				hasReplacementCharacter {
				writeUnicodeError(
					writer,
					http.StatusBadRequest,
					"el cuerpo contiene texto Unicode corrupto",
				)
				return
			}

			next.ServeHTTP(writer, request)
		},
	)
}

func mustValidateBody(
	request *http.Request,
) bool {
	switch request.Method {
	case http.MethodPost,
		http.MethodPut,
		http.MethodPatch:
	default:
		return false
	}

	contentType := strings.ToLower(
		request.Header.Get("Content-Type"),
	)

	return strings.Contains(
		contentType,
		"application/json",
	)
}

func writeUnicodeError(
	writer http.ResponseWriter,
	statusCode int,
	message string,
) {
	writer.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)

	writer.WriteHeader(statusCode)

	_ = json.NewEncoder(writer).Encode(
		errorResponse{
			Status:  "error",
			Message: message,
		},
	)
}
