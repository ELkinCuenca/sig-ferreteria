package config

import (
	"fmt"
	"os"
	"strings"
)

// IoTConfig reúne la conexión Oracle y MQTT
// requerida por el consumidor de telemetría.
type IoTConfig struct {
	Base         Config
	MQTTBroker   string
	MQTTClientID string
	MQTTUsername string
	MQTTPassword string
	MQTTTopic    string
}

// LoadIoT carga y valida la configuración
// exclusiva del consumidor IoT.
func LoadIoT() (IoTConfig, error) {
	base, err := Load()
	if err != nil {
		return IoTConfig{}, err
	}

	cfg := IoTConfig{
		Base: base,

		MQTTBroker: strings.TrimSpace(
			valueOrDefault(
				"MQTT_BROKER",
				"tcp://127.0.0.1:1883",
			),
		),

		MQTTClientID: strings.TrimSpace(
			valueOrDefault(
				"MQTT_CLIENT_ID",
				"sigefer-iot-consumer",
			),
		),

		MQTTUsername: strings.TrimSpace(
			os.Getenv("MQTT_USERNAME"),
		),

		MQTTPassword: os.Getenv(
			"MQTT_PASSWORD",
		),

		MQTTTopic: strings.TrimSpace(
			valueOrDefault(
				"MQTT_TOPIC",
				"sigefer/iot/+/telemetria",
			),
		),
	}

	switch {
	case cfg.MQTTBroker == "":
		return IoTConfig{}, fmt.Errorf(
			"MQTT_BROKER es obligatorio",
		)

	case cfg.MQTTClientID == "":
		return IoTConfig{}, fmt.Errorf(
			"MQTT_CLIENT_ID es obligatorio",
		)

	case cfg.MQTTUsername == "":
		return IoTConfig{}, fmt.Errorf(
			"MQTT_USERNAME es obligatorio",
		)

	case cfg.MQTTPassword == "":
		return IoTConfig{}, fmt.Errorf(
			"MQTT_PASSWORD es obligatorio",
		)

	case cfg.MQTTTopic == "":
		return IoTConfig{}, fmt.Errorf(
			"MQTT_TOPIC es obligatorio",
		)
	}

	return cfg, nil
}
