package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config representa la configuración de la API y Oracle.
type Config struct {
	AppPort    string
	DBHost     string
	DBPort     int
	DBService  string
	DBUser     string
	DBPassword string
}

// Load carga y valida las variables de entorno.
func Load() (Config, error) {
	cfg := Config{
		AppPort:    valueOrDefault("APP_PORT", "8080"),
		DBHost:     valueOrDefault("DB_HOST", "127.0.0.1"),
		DBService:  valueOrDefault("DB_SERVICE", "FREEPDB1"),
		DBUser:     os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASSWORD"),
	}

	dbPortText := valueOrDefault("DB_PORT", "1521")

	dbPort, err := strconv.Atoi(dbPortText)
	if err != nil {
		return Config{}, fmt.Errorf(
			"DB_PORT debe contener un número válido: %w",
			err,
		)
	}

	cfg.DBPort = dbPort

	if cfg.DBUser == "" {
		return Config{}, fmt.Errorf(
			"la variable DB_USER es obligatoria",
		)
	}

	if cfg.DBPassword == "" {
		return Config{}, fmt.Errorf(
			"la variable DB_PASSWORD es obligatoria",
		)
	}

	return cfg, nil
}

func valueOrDefault(name, defaultValue string) string {
	value := os.Getenv(name)
	if value == "" {
		return defaultValue
	}

	return value
}
