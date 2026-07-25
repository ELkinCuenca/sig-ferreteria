package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/mail"
	"os"
	"strings"
	"time"
	"unicode"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"

	"sigefer.local/backend/internal/config"
	"sigefer.local/backend/internal/database"
)

const (
	minimumPasswordLength = 12
	maximumPasswordBytes  = 72
)

type commandArguments struct {
	username  string
	firstName string
	lastName  string
	email     string
	role      string
}

func main() {
	arguments := parseArguments()

	if err := validateArguments(arguments); err != nil {
		log.Fatalf("datos inválidos: %v", err)
	}

	password, err := requestPassword()
	if err != nil {
		log.Fatalf("contraseña inválida: %v", err)
	}

	passwordHash, err := bcrypt.GenerateFromPassword(
		password,
		bcrypt.DefaultCost,
	)
	clearBytes(password)

	if err != nil {
		log.Fatalf(
			"no se pudo generar el hash: %v",
			err,
		)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf(
			"configuración inválida: %v",
			err,
		)
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		20*time.Second,
	)
	defer cancel()

	db, err := database.OpenOracle(ctx, cfg)
	if err != nil {
		log.Fatalf(
			"error conectando con Oracle: %v",
			err,
		)
	}
	defer db.Close()

	userID, err := createUser(
		ctx,
		db,
		arguments,
		string(passwordHash),
	)
	if err != nil {
		log.Fatalf(
			"no se pudo crear el usuario: %v",
			err,
		)
	}

	fmt.Printf(
		"Usuario creado correctamente. ID: %d\n",
		userID,
	)
}

func parseArguments() commandArguments {
	var arguments commandArguments

	flag.StringVar(
		&arguments.username,
		"usuario",
		"",
		"nombre de usuario",
	)

	flag.StringVar(
		&arguments.firstName,
		"nombres",
		"",
		"nombres del usuario",
	)

	flag.StringVar(
		&arguments.lastName,
		"apellidos",
		"",
		"apellidos del usuario",
	)

	flag.StringVar(
		&arguments.email,
		"correo",
		"",
		"correo electrónico",
	)

	flag.StringVar(
		&arguments.role,
		"rol",
		"ADMINISTRADOR",
		"rol del usuario",
	)

	flag.Parse()

	arguments.username = strings.ToLower(
		strings.TrimSpace(arguments.username),
	)

	arguments.firstName = strings.TrimSpace(
		arguments.firstName,
	)

	arguments.lastName = strings.TrimSpace(
		arguments.lastName,
	)

	arguments.email = strings.ToLower(
		strings.TrimSpace(arguments.email),
	)

	arguments.role = strings.ToUpper(
		strings.TrimSpace(arguments.role),
	)

	return arguments
}

func validateArguments(
	arguments commandArguments,
) error {
	switch {
	case arguments.username == "":
		return errors.New(
			"usuario es obligatorio",
		)

	case len(arguments.username) > 50:
		return errors.New(
			"usuario supera 50 caracteres",
		)

	case arguments.firstName == "":
		return errors.New(
			"nombres es obligatorio",
		)

	case len(arguments.firstName) > 100:
		return errors.New(
			"nombres supera 100 caracteres",
		)

	case arguments.lastName == "":
		return errors.New(
			"apellidos es obligatorio",
		)

	case len(arguments.lastName) > 100:
		return errors.New(
			"apellidos supera 100 caracteres",
		)

	case arguments.email == "":
		return errors.New(
			"correo es obligatorio",
		)

	case len(arguments.email) > 150:
		return errors.New(
			"correo supera 150 caracteres",
		)
	}

	if _, err := mail.ParseAddress(
		arguments.email,
	); err != nil {
		return errors.New(
			"correo no es válido",
		)
	}

	allowedRoles := map[string]bool{
		"ADMINISTRADOR": true,
		"VENDEDOR":      true,
		"BODEGUERO":     true,
		"GERENTE":       true,
	}

	if !allowedRoles[arguments.role] {
		return errors.New(
			"rol debe ser ADMINISTRADOR, " +
				"VENDEDOR, BODEGUERO o GERENTE",
		)
	}

	return nil
}

func requestPassword() ([]byte, error) {
	fmt.Print("Contraseña: ")

	password, err := term.ReadPassword(
		int(os.Stdin.Fd()),
	)
	fmt.Println()

	if err != nil {
		return nil, fmt.Errorf(
			"no se pudo leer la contraseña: %w",
			err,
		)
	}

	fmt.Print("Confirmar contraseña: ")

	confirmation, err := term.ReadPassword(
		int(os.Stdin.Fd()),
	)
	fmt.Println()

	if err != nil {
		clearBytes(password)

		return nil, fmt.Errorf(
			"no se pudo confirmar la contraseña: %w",
			err,
		)
	}

	defer clearBytes(confirmation)

	if string(password) != string(confirmation) {
		clearBytes(password)

		return nil, errors.New(
			"las contraseñas no coinciden",
		)
	}

	if err := validatePassword(password); err != nil {
		clearBytes(password)
		return nil, err
	}

	return password, nil
}

func validatePassword(password []byte) error {
	runes := []rune(string(password))

	if len(runes) < minimumPasswordLength {
		return fmt.Errorf(
			"debe contener al menos %d caracteres",
			minimumPasswordLength,
		)
	}

	if len(password) > maximumPasswordBytes {
		return fmt.Errorf(
			"no puede superar %d bytes",
			maximumPasswordBytes,
		)
	}

	var (
		hasUpper  bool
		hasLower  bool
		hasNumber bool
		hasSymbol bool
	)

	for _, character := range runes {
		switch {
		case unicode.IsUpper(character):
			hasUpper = true

		case unicode.IsLower(character):
			hasLower = true

		case unicode.IsDigit(character):
			hasNumber = true

		case unicode.IsPunct(character) ||
			unicode.IsSymbol(character):
			hasSymbol = true
		}
	}

	if !hasUpper ||
		!hasLower ||
		!hasNumber ||
		!hasSymbol {
		return errors.New(
			"debe incluir mayúscula, minúscula, " +
				"número y símbolo",
		)
	}

	return nil
}

func createUser(
	ctx context.Context,
	db *sql.DB,
	arguments commandArguments,
	passwordHash string,
) (int64, error) {
	transaction, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf(
			"no se pudo iniciar la transacción: %w",
			err,
		)
	}

	committed := false

	defer func() {
		if !committed {
			_ = transaction.Rollback()
		}
	}()

	var roleID int64

	err = transaction.QueryRowContext(
		ctx,
		`
			SELECT ID_ROL
			FROM ROL
			WHERE NOMBRE = :1
			  AND ESTADO = 'A'
		`,
		arguments.role,
	).Scan(&roleID)

	if errors.Is(err, sql.ErrNoRows) {
		return 0, errors.New(
			"el rol solicitado no existe o está inactivo",
		)
	}

	if err != nil {
		return 0, fmt.Errorf(
			"no se pudo consultar el rol: %w",
			err,
		)
	}

	var duplicateCount int

	err = transaction.QueryRowContext(
		ctx,
		`
			SELECT COUNT(*)
			FROM USUARIO
			WHERE LOWER(NOMBRE_USUARIO) = :1
			   OR LOWER(CORREO) = :2
		`,
		arguments.username,
		arguments.email,
	).Scan(&duplicateCount)

	if err != nil {
		return 0, fmt.Errorf(
			"no se pudo validar el usuario: %w",
			err,
		)
	}

	if duplicateCount > 0 {
		return 0, errors.New(
			"el usuario o correo ya se encuentra registrado",
		)
	}

	_, err = transaction.ExecContext(
		ctx,
		`
			INSERT INTO USUARIO (
				ID_ROL,
				NOMBRE_USUARIO,
				NOMBRES,
				APELLIDOS,
				CORREO,
				CLAVE_HASH,
				ESTADO,
				INTENTOS_FALLIDOS,
				FECHA_CREACION,
				FECHA_ACTUALIZACION
			)
			VALUES (
				:1,
				:2,
				:3,
				:4,
				:5,
				:6,
				'ACTIVO',
				0,
				CAST(
					SYSTIMESTAMP AT TIME ZONE '-05:00'
					AS TIMESTAMP
				),
				CAST(
					SYSTIMESTAMP AT TIME ZONE '-05:00'
					AS TIMESTAMP
				)
			)
		`,
		roleID,
		arguments.username,
		arguments.firstName,
		arguments.lastName,
		arguments.email,
		passwordHash,
	)
	if err != nil {
		return 0, fmt.Errorf(
			"no se pudo insertar el usuario: %w",
			err,
		)
	}

	var userID int64

	err = transaction.QueryRowContext(
		ctx,
		`
			SELECT ID_USUARIO
			FROM USUARIO
			WHERE NOMBRE_USUARIO = :1
		`,
		arguments.username,
	).Scan(&userID)
	if err != nil {
		return 0, fmt.Errorf(
			"no se pudo recuperar el usuario: %w",
			err,
		)
	}

	_, err = transaction.ExecContext(
		ctx,
		`
			INSERT INTO AUDITORIA (
				ID_USUARIO,
				TABLA_AFECTADA,
				OPERACION,
				ID_REGISTRO,
				VALORES_NUEVOS,
				ORIGEN,
				FECHA_EVENTO
			)
			VALUES (
				:1,
				'USUARIO',
				'INSERT',
				:2,
				'{"estado":"ACTIVO","rol":"' ||
					:3 || '"}',
				'CLI cmd/create-user',
				CAST(
					SYSTIMESTAMP AT TIME ZONE '-05:00'
					AS TIMESTAMP
				)
			)
		`,
		userID,
		fmt.Sprintf("%d", userID),
		arguments.role,
	)
	if err != nil {
		return 0, fmt.Errorf(
			"no se pudo auditar el usuario: %w",
			err,
		)
	}

	if err := transaction.Commit(); err != nil {
		return 0, fmt.Errorf(
			"no se pudo confirmar la transacción: %w",
			err,
		)
	}

	committed = true

	return userID, nil
}

func clearBytes(value []byte) {
	for index := range value {
		value[index] = 0
	}
}
