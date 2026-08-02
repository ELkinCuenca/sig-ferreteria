package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"sigefer.local/backend/internal/config"
	"sigefer.local/backend/internal/database"
	"sigefer.local/backend/internal/models"
	"sigefer.local/backend/internal/repository"
)

type inboundMessage struct {
	topic   string
	payload []byte
}

func main() {
	cfg, err := config.LoadIoT()
	if err != nil {
		log.Fatalf(
			"configuración IoT inválida: %v",
			err,
		)
	}

	rootCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	db, err := database.OpenOracle(
		rootCtx,
		cfg.Base,
	)
	if err != nil {
		log.Fatalf(
			"error conectando con Oracle: %v",
			err,
		)
	}
	defer db.Close()

	iotRepository :=
		repository.NewIoTRepository(db)

	messageQueue := make(
		chan inboundMessage,
		100,
	)

	messageHandler := func(
		_ mqtt.Client,
		message mqtt.Message,
	) {
		payloadCopy := append(
			[]byte(nil),
			message.Payload()...,
		)

		select {
		case messageQueue <- inboundMessage{
			topic:   message.Topic(),
			payload: payloadCopy,
		}:

		case <-rootCtx.Done():
		}
	}

	options := mqtt.NewClientOptions()

	options.AddBroker(
		cfg.MQTTBroker,
	)

	options.SetClientID(
		cfg.MQTTClientID,
	)

	options.SetUsername(
		cfg.MQTTUsername,
	)

	options.SetPassword(
		cfg.MQTTPassword,
	)

	options.SetCleanSession(
		true,
	)

	options.SetAutoReconnect(
		true,
	)

	options.SetConnectTimeout(
		10 * time.Second,
	)

	options.SetKeepAlive(
		30 * time.Second,
	)

	options.SetPingTimeout(
		10 * time.Second,
	)

	options.SetMaxReconnectInterval(
		30 * time.Second,
	)

	options.SetConnectionLostHandler(
		func(
			_ mqtt.Client,
			connectionError error,
		) {
			log.Printf(
				"conexión MQTT perdida: %v",
				connectionError,
			)
		},
	)

	options.SetOnConnectHandler(
		func(client mqtt.Client) {
			log.Printf(
				"MQTT conectado al broker %s",
				cfg.MQTTBroker,
			)

			token := client.Subscribe(
				cfg.MQTTTopic,
				1,
				messageHandler,
			)

			if !token.WaitTimeout(
				10 * time.Second,
			) {
				log.Printf(
					"tiempo agotado suscribiendo a %s",
					cfg.MQTTTopic,
				)

				return
			}

			if err := token.Error(); err != nil {
				log.Printf(
					"error suscribiendo a %s: %v",
					cfg.MQTTTopic,
					err,
				)

				return
			}

			log.Printf(
				"suscripción MQTT activa: %s",
				cfg.MQTTTopic,
			)
		},
	)

	client := mqtt.NewClient(
		options,
	)

	connectToken := client.Connect()

	if !connectToken.WaitTimeout(
		20 * time.Second,
	) {
		log.Fatal(
			"tiempo agotado conectando con MQTT",
		)
	}

	if err := connectToken.Error(); err != nil {
		log.Fatalf(
			"no se pudo conectar con MQTT: %v",
			err,
		)
	}

	log.Println(
		"Consumidor IoT SIGEFER iniciado.",
	)

	offlineTicker := time.NewTicker(
		30 * time.Second,
	)
	defer offlineTicker.Stop()

	for {
		select {
		case <-rootCtx.Done():
			log.Println(
				"Deteniendo consumidor IoT SIGEFER...",
			)

			if client.IsConnected() {
				client.Disconnect(
					1000,
				)
			}

			log.Println(
				"Consumidor IoT detenido correctamente.",
			)

			return

		case message := <-messageQueue:
			processMessage(
				rootCtx,
				iotRepository,
				message,
			)

		case <-offlineTicker.C:
			checkOfflineDevices(
				rootCtx,
				iotRepository,
			)
		}
	}
}

func processMessage(
	parentCtx context.Context,
	iotRepository *repository.IoTRepository,
	message inboundMessage,
) {
	if len(message.payload) == 0 {
		log.Printf(
			"mensaje MQTT vacío recibido en %s",
			message.topic,
		)

		return
	}

	if len(message.payload) > 16384 {
		log.Printf(
			"mensaje MQTT rechazado por tamaño: %d bytes",
			len(message.payload),
		)

		return
	}

	var telemetry models.IoTTelemetry

	if err := json.Unmarshal(
		message.payload,
		&telemetry,
	); err != nil {
		log.Printf(
			"JSON MQTT inválido en %s: %v",
			message.topic,
			err,
		)

		return
	}

	telemetry.Normalize()

	if err := telemetry.Validate(); err != nil {
		log.Printf(
			"telemetría MQTT inválida en %s: %v",
			message.topic,
			err,
		)

		return
	}

	ctx, cancel := context.WithTimeout(
		parentCtx,
		15*time.Second,
	)
	defer cancel()

	result, err := iotRepository.ProcessTelemetry(
		ctx,
		telemetry,
		string(message.payload),
	)
	if err != nil {
		log.Printf(
			"error persistiendo telemetría de %s: %v",
			telemetry.DeviceCode,
			err,
		)

		return
	}

	if result.Duplicate {
		log.Printf(
			"mensaje duplicado ignorado: dispositivo=%s arranque=%s secuencia=%d lectura=%d",
			telemetry.DeviceCode,
			telemetry.BootID,
			telemetry.Sequence,
			result.ReadingID,
		)

		return
	}

	log.Printf(
		"lectura guardada: id=%d dispositivo=%s arranque=%s secuencia=%d",
		result.ReadingID,
		telemetry.DeviceCode,
		telemetry.BootID,
		telemetry.Sequence,
	)

	for _, alertType := range result.Alerts {
		log.Printf(
			"alerta IoT activa: dispositivo=%s tipo=%s",
			telemetry.DeviceCode,
			alertType,
		)
	}
}

func checkOfflineDevices(
	parentCtx context.Context,
	iotRepository *repository.IoTRepository,
) {
	ctx, cancel := context.WithTimeout(
		parentCtx,
		15*time.Second,
	)
	defer cancel()

	affectedRows, err :=
		iotRepository.CheckOfflineDevices(ctx)
	if err != nil {
		log.Printf(
			"error comprobando dispositivos sin comunicación: %v",
			err,
		)

		return
	}

	if affectedRows > 0 {
		log.Printf(
			"comprobación de comunicación IoT actualizó %d alerta(s)",
			affectedRows,
		)
	}
}
