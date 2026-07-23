package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sigefer.local/backend/internal/config"
	"sigefer.local/backend/internal/database"
	"sigefer.local/backend/internal/handlers"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf(
			"configuración inválida: %v",
			err,
		)
	}

	rootCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	db, err := database.OpenOracle(rootCtx, cfg)
	if err != nil {
		log.Fatalf(
			"error conectando con Oracle: %v",
			err,
		)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf(
				"error cerrando Oracle: %v",
				err,
			)
		}
	}()

	router := http.NewServeMux()

	router.HandleFunc(
		"/api/v1/health",
		handlers.Health(db),
	)

	server := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		<-rootCtx.Done()

		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf(
				"error apagando la API: %v",
				err,
			)
		}
	}()

	log.Printf(
		"SIGEFER API escuchando en el puerto %s",
		cfg.AppPort,
	)

	err = server.ListenAndServe()

	if err != nil &&
		!errors.Is(err, http.ErrServerClosed) {
		log.Fatalf(
			"error del servidor HTTP: %v",
			err,
		)
	}

	log.Println(
		"SIGEFER API detenida correctamente",
	)
}
