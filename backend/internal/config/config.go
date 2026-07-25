package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config representa la configuración de la API,
// Oracle y el sistema de autenticación.
type Config struct {
	AppPort          string
	DBHost           string
	DBPort           int
	DBService        string
	DBUser           string
	DBPassword       string
	TaxRate          string
	SessionHours     int
	MaxLoginAttempts int
}

// Load carga y valida las variables de entorno.
func Load() (Config, error) {
	cfg := Config{
		AppPort:    valueOrDefault("APP_PORT", "8080"),
		DBHost:     valueOrDefault("DB_HOST", "127.0.0.1"),
		DBService:  valueOrDefault("DB_SERVICE", "FREEPDB1"),
		DBUser:     os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		TaxRate:    valueOrDefault("TAX_RATE", "0.15"),
	}

	dbPort, err := positiveInteger(
		"DB_PORT",
		"1521",
	)
	if err != nil {
		return Config{}, err
	}

	sessionHours, err := positiveInteger(
		"SESSION_HOURS",
		"12",
	)
	if err != nil {
		return Config{}, err
	}

	maxAttempts, err := positiveInteger(
		"MAX_LOGIN_ATTEMPTS",
		"5",
	)
	if err != nil {
		return Config{}, err
	}

	if sessionHours > 168 {
		return Config{}, fmt.Errorf(
			"SESSION_HOURS no puede superar 168",
		)
	}

	if maxAttempts > 20 {
		return Config{}, fmt.Errorf(
			"MAX_LOGIN_ATTEMPTS no puede superar 20",
		)
	}

	cfg.DBPort = dbPort
	cfg.SessionHours = sessionHours
	cfg.MaxLoginAttempts = maxAttempts

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

func positiveInteger(
	name string,
	defaultValue string,
) (int, error) {
	text := valueOrDefault(
		name,
		defaultValue,
	)

	value, err := strconv.Atoi(text)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf(
			"%s debe contener un entero positivo",
			name,
		)
	}

	return value, nil
}

func valueOrDefault(
	name string,
	defaultValue string,
) string {
	value := os.Getenv(name)

	if value == "" {
		return defaultValue
	}

	return value
}
