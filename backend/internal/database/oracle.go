package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	go_ora "github.com/sijms/go-ora/v2"

	"sigefer.local/backend/internal/config"
)

// OpenOracle crea el pool de conexiones y verifica Oracle.
func OpenOracle(
	ctx context.Context,
	cfg config.Config,
) (*sql.DB, error) {
	options := map[string]string{
		"TIMEOUT":         "15",
		"CONNECT TIMEOUT": "15",
	}

	connectionString := go_ora.BuildUrl(
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBService,
		cfg.DBUser,
		cfg.DBPassword,
		options,
	)

	db, err := sql.Open("oracle", connectionString)
	if err != nil {
		return nil, fmt.Errorf(
			"no se pudo crear la conexión Oracle: %w",
			err,
		)
	}

	// Configuración básica del pool.
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	pingCtx, cancel := context.WithTimeout(
		ctx,
		15*time.Second,
	)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()

		return nil, fmt.Errorf(
			"Oracle no respondió a la verificación: %w",
			err,
		)
	}

	return db, nil
}
