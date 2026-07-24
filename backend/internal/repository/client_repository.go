package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"sigefer.local/backend/internal/models"
)

// ClientRepository administra el acceso a clientes en Oracle.
type ClientRepository struct {
	db *sql.DB
}

// NewClientRepository crea el repositorio de clientes.
func NewClientRepository(
	db *sql.DB,
) *ClientRepository {
	return &ClientRepository{
		db: db,
	}
}

// List devuelve clientes activos y permite buscar por
// identificación, nombres, apellidos o razón social.
func (repository *ClientRepository) List(
	ctx context.Context,
	search string,
) ([]models.Client, error) {
	query := `
		SELECT
			c.ID_CLIENTE,
			c.TIPO_IDENTIFICACION,
			c.IDENTIFICACION,

			CASE
				WHEN c.RAZON_SOCIAL IS NOT NULL
					THEN TRIM(c.RAZON_SOCIAL)
				ELSE
					TRIM(
						NVL(c.NOMBRES, '')
						|| ' '
						|| NVL(c.APELLIDOS, '')
					)
			END AS NOMBRE_COMPLETO,

			c.TELEFONO,
			c.CORREO,
			c.DIRECCION,
			'ACTIVO' AS ESTADO

		FROM CLIENTE c

		WHERE c.ESTADO = 'A'
	`

	arguments := make([]any, 0, 3)
	normalizedSearch := normalizeClientSearch(search)

	if normalizedSearch != "" {
		pattern := "%" +
			escapeLikePattern(normalizedSearch) +
			"%"

		query += `
			AND (
				TRANSLATE(
					UPPER(c.IDENTIFICACION),
					'ÁÉÍÓÚÜÑ',
					'AEIOUUN'
				) LIKE :1 ESCAPE '\'

				OR TRANSLATE(
					UPPER(NVL(c.RAZON_SOCIAL, '')),
					'ÁÉÍÓÚÜÑ',
					'AEIOUUN'
				) LIKE :2 ESCAPE '\'

				OR TRANSLATE(
					UPPER(
						TRIM(
							NVL(c.NOMBRES, '')
							|| ' '
							|| NVL(c.APELLIDOS, '')
						)
					),
					'ÁÉÍÓÚÜÑ',
					'AEIOUUN'
				) LIKE :3 ESCAPE '\'
			)
		`

		arguments = append(
			arguments,
			pattern,
			pattern,
			pattern,
		)
	}

	query += `
		ORDER BY
			CASE
				WHEN c.TIPO_IDENTIFICACION =
					'CONSUMIDOR_FINAL'
					THEN 0
				ELSE 1
			END,
			NOMBRE_COMPLETO

		FETCH FIRST 50 ROWS ONLY
	`

	rows, err := repository.db.QueryContext(
		ctx,
		query,
		arguments...,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"no se pudieron consultar los clientes: %w",
			err,
		)
	}

	defer rows.Close()

	clients := make([]models.Client, 0)

	for rows.Next() {
		var (
			client  models.Client
			phone   sql.NullString
			email   sql.NullString
			address sql.NullString
		)

		err := rows.Scan(
			&client.IDCliente,
			&client.TipoIdentificacion,
			&client.Identificacion,
			&client.NombreCompleto,
			&phone,
			&email,
			&address,
			&client.Estado,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"no se pudo leer un cliente: %w",
				err,
			)
		}

		if phone.Valid {
			client.Telefono = phone.String
		}

		if email.Valid {
			client.Correo = email.String
		}

		if address.Valid {
			client.Direccion = address.String
		}

		clients = append(clients, client)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"error recorriendo los clientes: %w",
			err,
		)
	}

	return clients, nil
}

func normalizeClientSearch(
	value string,
) string {
	value = strings.ToUpper(
		strings.TrimSpace(value),
	)

	replacer := strings.NewReplacer(
		"Á", "A",
		"É", "E",
		"Í", "I",
		"Ó", "O",
		"Ú", "U",
		"Ü", "U",
		"Ñ", "N",
	)

	return replacer.Replace(value)
}

func escapeLikePattern(
	value string,
) string {
	value = strings.ReplaceAll(
		value,
		`\`,
		`\\`,
	)

	value = strings.ReplaceAll(
		value,
		`%`,
		`\%`,
	)

	value = strings.ReplaceAll(
		value,
		`_`,
		`\_`,
	)

	return value
}
