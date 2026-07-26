export type BPMState =
  'BORRADOR' | 'SOLICITADA' | 'APROBADA' | 'RECHAZADA' | 'EN_PEDIDO' | 'RECIBIDA' | 'CERRADA';

export type BPMAction = 'ENVIAR' | 'APROBAR' | 'RECHAZAR' | 'PEDIDO' | 'RECIBIR' | 'CERRAR';

export interface BPMProvider {
  id_proveedor: number;
  ruc: string;
  razon_social: string;
  nombre_contacto?: string;
  telefono?: string;
  correo?: string;
}

export interface BPMProviderListResponse {
  status: string;
  total: number;
  proveedores: BPMProvider[];
}

export interface BPMReplenishment {
  id_solicitud: number;
  numero_solicitud: string;

  codigo_producto: string;
  producto: string;
  unidad_medida: string;

  id_proveedor: number;
  ruc_proveedor: string;
  proveedor: string;

  id_alerta?: number;
  tipo_alerta?: string;
  estado_alerta?: string;

  cantidad_solicitada: string;
  cantidad_recibida: string;

  costo_unitario_estimado: string;
  costo_total_estimado: string;

  estado: BPMState;

  stock_actual: string;
  stock_reservado: string;
  stock_disponible: string;
  stock_minimo: string;

  id_usuario_solicitante: number;
  usuario_solicitante: string;

  id_usuario_aprobador?: number;
  usuario_aprobador?: string;

  id_usuario_receptor?: number;
  usuario_receptor?: string;

  fecha_solicitud?: string;
  fecha_aprobacion?: string;
  fecha_rechazo?: string;
  fecha_pedido?: string;
  fecha_recepcion?: string;
  fecha_cierre?: string;

  observacion?: string;
  motivo_rechazo?: string;

  fecha_creacion: string;
  fecha_actualizacion: string;
}

export interface BPMHistory {
  id_historial: number;
  id_usuario: number;
  usuario: string;
  estado_anterior?: BPMState;
  estado_nuevo: BPMState;
  accion: string;
  observacion?: string;
  fecha_evento: string;
}

export interface BPMReplenishmentDetail extends BPMReplenishment {
  historial: BPMHistory[];
}

export interface BPMListResponse {
  status: string;
  total: number;
  filtro_estado?: BPMState;
  reposiciones: BPMReplenishment[];
}

export interface BPMCreatePayload {
  codigo_producto: string;
  id_proveedor: number;
  id_alerta?: number;
  cantidad_solicitada: string;
  costo_unitario_estimado: string;
  observacion?: string;
}

export interface BPMTransitionPayload {
  observacion?: string;
}

export interface BPMRejectPayload {
  motivo_rechazo: string;
}

export interface BPMReceivePayload {
  cantidad_recibida: string;
  observacion?: string;
}

export interface BPMKanbanColumn {
  state: BPMState;
  title: string;
  description: string;
}
