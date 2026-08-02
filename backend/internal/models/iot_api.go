package models

// IoTReading representa una lectura ambiental
// almacenada en Oracle.
type IoTReading struct {
	IDLectura         int64    `json:"id_lectura"`
	CodigoDispositivo string   `json:"codigo_dispositivo"`
	IDArranque        string   `json:"id_arranque"`
	Secuencia         int64    `json:"secuencia"`
	TemperaturaC      *float64 `json:"temperatura_c"`
	HumedadPct        *float64 `json:"humedad_pct"`
	RSSIDBM           *int64   `json:"senal_wifi_dbm"`
	EstadoSensor      string   `json:"estado_sensor"`
	IPDispositivo     string   `json:"ip_dispositivo,omitempty"`
	UptimeSegundos    int64    `json:"uptime_s"`
	LecturasValidas   int64    `json:"lecturas_validas"`
	LecturasFallidas  int64    `json:"lecturas_fallidas"`
	Origen            string   `json:"origen"`
	FechaRecepcion    string   `json:"fecha_recepcion"`
}

// IoTReadingListResponse representa el historial
// de lecturas ambientales.
type IoTReadingListResponse struct {
	Status      string       `json:"status"`
	Total       int          `json:"total"`
	Dispositivo string       `json:"dispositivo"`
	Lecturas    []IoTReading `json:"lecturas"`
}

// IoTConfiguration contiene los umbrales
// configurados para un dispositivo.
type IoTConfiguration struct {
	IDConfiguracion         int64   `json:"id_configuracion"`
	IDDispositivo           int64   `json:"id_dispositivo"`
	CodigoDispositivo       string  `json:"codigo_dispositivo"`
	TemperaturaMinC         float64 `json:"temperatura_min_c"`
	TemperaturaMaxC         float64 `json:"temperatura_max_c"`
	HumedadMinPct           float64 `json:"humedad_min_pct"`
	HumedadMaxPct           float64 `json:"humedad_max_pct"`
	SegundosSinComunicacion int64   `json:"segundos_sin_comunicacion"`
	Estado                  string  `json:"estado"`
	FechaActualizacion      string  `json:"fecha_actualizacion"`
}

// IoTConfigurationResponse representa la configuración
// consultada desde la API.
type IoTConfigurationResponse struct {
	Status        string           `json:"status"`
	Configuracion IoTConfiguration `json:"configuracion"`
}

// IoTDeviceSummary representa el estado operativo
// actual del dispositivo.
type IoTDeviceSummary struct {
	IDDispositivo      int64  `json:"id_dispositivo"`
	Codigo             string `json:"codigo"`
	Nombre             string `json:"nombre"`
	Ubicacion          string `json:"ubicacion,omitempty"`
	TipoSensor         string `json:"tipo_sensor"`
	Estado             string `json:"estado"`
	EstadoComunicacion string `json:"estado_comunicacion"`
	UltimaComunicacion string `json:"ultima_comunicacion,omitempty"`
	UltimaIP           string `json:"ultima_ip,omitempty"`
	UltimoRSSIDBM      *int64 `json:"ultimo_rssi_dbm"`
	UltimoEstadoSensor string `json:"ultimo_estado_sensor,omitempty"`
}

// IoTSummaryResponse contiene los indicadores
// principales del monitoreo ambiental.
type IoTSummaryResponse struct {
	Status            string           `json:"status"`
	FechaGeneracion   string           `json:"fecha_generacion"`
	Dispositivo       IoTDeviceSummary `json:"dispositivo"`
	Configuracion     IoTConfiguration `json:"configuracion"`
	UltimaLectura     *IoTReading      `json:"ultima_lectura"`
	TotalLecturas     int64            `json:"total_lecturas"`
	AlertasPendientes int64            `json:"alertas_pendientes"`
}

// IoTAlert representa una alerta ambiental
// o de pérdida de comunicación.
type IoTAlert struct {
	IDAlertaIoT         int64    `json:"id_alerta_iot"`
	CodigoDispositivo   string   `json:"codigo_dispositivo"`
	IDLectura           *int64   `json:"id_lectura,omitempty"`
	TipoAlerta          string   `json:"tipo_alerta"`
	Severidad           string   `json:"severidad"`
	Mensaje             string   `json:"mensaje"`
	ValorDetectado      *float64 `json:"valor_detectado"`
	ValorLimite         *float64 `json:"valor_limite"`
	Estado              string   `json:"estado"`
	FechaGeneracion     string   `json:"fecha_generacion"`
	IDUsuarioAtencion   *int64   `json:"id_usuario_atencion,omitempty"`
	FechaAtencion       string   `json:"fecha_atencion,omitempty"`
	ObservacionAtencion string   `json:"observacion_atencion,omitempty"`
	FechaActualizacion  string   `json:"fecha_actualizacion"`
}

// IoTAlertListResponse representa el listado
// filtrado de alertas IoT.
type IoTAlertListResponse struct {
	Status      string     `json:"status"`
	Total       int        `json:"total"`
	Dispositivo string     `json:"dispositivo"`
	Estado      string     `json:"filtro_estado,omitempty"`
	Alertas     []IoTAlert `json:"alertas"`
}

// AttendIoTAlertRequest contiene la observación
// registrada al atender una alerta.
type AttendIoTAlertRequest struct {
	Observacion string `json:"observacion"`
}

// IoTAlertUpdateResult representa una alerta
// después de ser atendida.
type IoTAlertUpdateResult struct {
	Status              string   `json:"status"`
	IDAlertaIoT         int64    `json:"id_alerta_iot"`
	CodigoDispositivo   string   `json:"codigo_dispositivo"`
	TipoAlerta          string   `json:"tipo_alerta"`
	Severidad           string   `json:"severidad"`
	Mensaje             string   `json:"mensaje"`
	ValorDetectado      *float64 `json:"valor_detectado"`
	ValorLimite         *float64 `json:"valor_limite"`
	Estado              string   `json:"estado"`
	FechaGeneracion     string   `json:"fecha_generacion"`
	IDUsuarioAtencion   int64    `json:"id_usuario_atencion"`
	FechaAtencion       string   `json:"fecha_atencion"`
	ObservacionAtencion string   `json:"observacion_atencion"`
}

// UpdateIoTConfigurationRequest contiene los nuevos
// umbrales administrativos.
type UpdateIoTConfigurationRequest struct {
	TemperaturaMinC         float64 `json:"temperatura_min_c"`
	TemperaturaMaxC         float64 `json:"temperatura_max_c"`
	HumedadMinPct           float64 `json:"humedad_min_pct"`
	HumedadMaxPct           float64 `json:"humedad_max_pct"`
	SegundosSinComunicacion int64   `json:"segundos_sin_comunicacion"`
}
