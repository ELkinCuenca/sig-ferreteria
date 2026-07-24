package handlers

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"sigefer.local/backend/internal/models"
	"sigefer.local/backend/internal/repository"
)

// ClientHandler contiene las operaciones HTTP de clientes.
type ClientHandler struct {
	repository *repository.ClientRepository
}

// ClientListResponse representa el listado de clientes.
type ClientListResponse struct {
	Status   string          `json:"status"`
	Total    int             `json:"total"`
	Busqueda string          `json:"busqueda,omitempty"`
	Clients  []models.Client `json:"clientes"`
	Message  string          `json:"message,omitempty"`
}

// NewClientHandler crea el handler de clientes.
func NewClientHandler(
	clientRepository *repository.ClientRepository,
) *ClientHandler {
	return &ClientHandler{
		repository: clientRepository,
	}
}

// List procesa GET /api/v1/clientes.
func (handler *ClientHandler) List(
	writer http.ResponseWriter,
	request *http.Request,
) {
	writer.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)

	search := strings.TrimSpace(
		request.URL.Query().Get("buscar"),
	)

	if utf8.RuneCountInString(search) > 100 {
		writer.WriteHeader(http.StatusBadRequest)

		writeJSON(
			writer,
			ClientListResponse{
				Status:  "error",
				Message: "buscar admite máximo 100 caracteres",
				Clients: []models.Client{},
			},
		)

		return
	}

	ctx, cancel := context.WithTimeout(
		request.Context(),
		10*time.Second,
	)
	defer cancel()

	clients, err := handler.repository.List(
		ctx,
		search,
	)
	if err != nil {
		log.Printf(
			"error consultando clientes: %v",
			err,
		)

		writer.WriteHeader(
			http.StatusInternalServerError,
		)

		writeJSON(
			writer,
			ClientListResponse{
				Status:  "error",
				Message: "no fue posible consultar los clientes",
				Clients: []models.Client{},
			},
		)

		return
	}

	writeJSON(
		writer,
		ClientListResponse{
			Status:   "ok",
			Total:    len(clients),
			Busqueda: search,
			Clients:  clients,
		},
	)
}
