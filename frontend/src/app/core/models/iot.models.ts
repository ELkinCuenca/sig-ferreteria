export type IoTAlertStatus = 'PENDIENTE' | 'ATENDIDA';

export type IoTCommunicationStatus = 'EN_LINEA' | 'SIN_COMUNICACION' | 'SIN_DATOS';

export interface IoTDeviceSummary {
  id_dispositivo: number;
  codigo: string;
  nombre: string;
  ubicacion?: string;
  tipo_sensor: string;
  estado: string;
  estado_comunicacion: IoTCommunicationStatus;
  ultima_comunicacion?: string;
  ultima_ip?: string;
  ultimo_rssi_dbm: number | null;
  ultimo_estado_sensor?: string;
}

export interface IoTConfiguration {
  id_configuracion: number;
  id_dispositivo: number;
  codigo_dispositivo: string;
  temperatura_min_c: number;
  temperatura_max_c: number;
  humedad_min_pct: number;
  humedad_max_pct: number;
  segundos_sin_comunicacion: number;
  estado: string;
  fecha_actualizacion: string;
}

export interface IoTReading {
  id_lectura: number;
  codigo_dispositivo: string;
  id_arranque: string;
  secuencia: number;
  temperatura_c: number | null;
  humedad_pct: number | null;
  senal_wifi_dbm: number | null;
  estado_sensor: string;
  ip_dispositivo?: string;
  uptime_s: number;
  lecturas_validas: number;
  lecturas_fallidas: number;
  origen: string;
  fecha_recepcion: string;
}

export interface IoTSummaryResponse {
  status: string;
  fecha_generacion: string;
  dispositivo: IoTDeviceSummary;
  configuracion: IoTConfiguration;
  ultima_lectura: IoTReading | null;
  total_lecturas: number;
  alertas_pendientes: number;
}

export interface IoTReadingListResponse {
  status: string;
  total: number;
  dispositivo: string;
  lecturas: IoTReading[];
}

export interface IoTAlert {
  id_alerta_iot: number;
  codigo_dispositivo: string;
  id_lectura?: number;
  tipo_alerta: string;
  severidad: string;
  mensaje: string;
  valor_detectado: number | null;
  valor_limite: number | null;
  estado: IoTAlertStatus;
  fecha_generacion: string;
  id_usuario_atencion?: number;
  fecha_atencion?: string;
  observacion_atencion?: string;
  fecha_actualizacion: string;
}

export interface IoTAlertListResponse {
  status: string;
  total: number;
  dispositivo: string;
  filtro_estado?: IoTAlertStatus;
  alertas: IoTAlert[];
}

export interface AttendIoTAlertRequest {
  observacion: string;
}

export interface IoTAlertUpdateResult {
  status: string;
  id_alerta_iot: number;
  codigo_dispositivo: string;
  tipo_alerta: string;
  severidad: string;
  mensaje: string;
  valor_detectado: number | null;
  valor_limite: number | null;
  estado: IoTAlertStatus;
  fecha_generacion: string;
  id_usuario_atencion: number;
  fecha_atencion: string;
  observacion_atencion: string;
}

export interface IoTConfigurationResponse {
  status: string;
  configuracion: IoTConfiguration;
}

export interface UpdateIoTConfigurationRequest {
  temperatura_min_c: number;
  temperatura_max_c: number;
  humedad_min_pct: number;
  humedad_max_pct: number;
  segundos_sin_comunicacion: number;
}
