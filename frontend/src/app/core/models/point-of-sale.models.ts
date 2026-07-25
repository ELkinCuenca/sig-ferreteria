export type PaymentMethod = 'EFECTIVO' | 'TARJETA' | 'TRANSFERENCIA' | 'MIXTO';

export interface Client {
  id_cliente: number;
  tipo_identificacion: string;
  identificacion: string;
  nombre_completo: string;
  telefono?: string;
  correo?: string;
  direccion?: string;
  estado: string;
}

export interface ClientListResponse {
  status: string;
  total: number;
  busqueda?: string;
  clientes: Client[];
}

export interface SaleItemPayload {
  codigo_producto: string;
  cantidad: string;
  descuento: string;
}

export interface SaleCreatePayload {
  identificacion_cliente: string;
  metodo_pago: PaymentMethod;
  descuento_general: string;
  observacion?: string;
  items: SaleItemPayload[];
}

export interface SaleItemResult {
  codigo_producto: string;
  nombre_producto: string;
  cantidad: string;
  precio_unitario: string;
  descuento: string;
  subtotal_linea: string;
  stock_anterior: string;
  stock_nuevo: string;
  stock_disponible: string;
}

export interface GeneratedStockAlert {
  codigo_producto: string;
  tipo_alerta: string;
  stock_detectado: string;
  stock_minimo: string;
}

export interface SaleCreateResponse {
  status: string;
  id_venta: number;
  numero_venta: string;
  subtotal: string;
  descuento: string;
  impuesto: string;
  total: string;
  metodo_pago: PaymentMethod;
  estado: string;
  items: SaleItemResult[];
  alertas_stock: GeneratedStockAlert[];
}
