package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

// HealthResponse representa el estado técnico del sistema.
type HealthResponse struct {
	Status          string `json:"status"`
	Service         string `json:"service"`
	Database        string `json:"database"`
	DatabaseUser    string `json:"database_user,omitempty"`
	DatabaseService string `json:"database_service,omitempty"`
	DatabaseTime    string `json:"database_time,omitempty"`
	Message         string `json:"message,omitempty"`
}

// Health consulta Oracle y devuelve el estado de la API.
func Health(db *sql.DB) http.HandlerFunc {
	return func(
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
				HealthResponse{
					Status:  "error",
					Service: "SIGEFER API",
					Message: "método HTTP no permitido",
				},
			)
			return
		}

		ctx, cancel := context.WithTimeout(
			request.Context(),
			5*time.Second,
		)
		defer cancel()

		const query = `
			SELECT
				USER,
				SYS_CONTEXT('USERENV', 'CON_NAME'),
				TO_CHAR(
					SYSTIMESTAMP,
					'YYYY-MM-DD"T"HH24:MI:SS.FF3 TZH:TZM'
				)
			FROM DUAL
		`

		var databaseUser string
		var databaseService string
		var databaseTime string

		err := db.QueryRowContext(ctx, query).Scan(
			&databaseUser,
			&databaseService,
			&databaseTime,
		)

		if err != nil {
			writer.WriteHeader(
				http.StatusServiceUnavailable,
			)

			writeJSON(
				writer,
				HealthResponse{
					Status:   "error",
					Service:  "SIGEFER API",
					Database: "disconnected",
					Message:  "Oracle no está disponible",
				},
			)
			return
		}

		writer.WriteHeader(http.StatusOK)

		writeJSON(
			writer,
			HealthResponse{
				Status:          "ok",
				Service:         "SIGEFER API",
				Database:        "connected",
				DatabaseUser:    databaseUser,
				DatabaseService: databaseService,
				DatabaseTime:    databaseTime,
			},
		)
	}
}

func writeJSON(
	writer http.ResponseWriter,
	value any,
) {
	_ = json.NewEncoder(writer).Encode(value)
}
