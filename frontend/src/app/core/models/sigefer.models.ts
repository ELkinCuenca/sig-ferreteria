export interface DashboardSummary {
  status: string;
  fecha_generacion: string;
  ventas_hoy: number;
  ingresos_hoy: string;
  unidades_vendidas_hoy: string;
  productos_reposicion: number;
  alertas_pendientes: number;
  valor_inventario_costo: string;
  valor_inventario_venta: string;
  margen_potencial: string;
  costo_reposicion_estimado: string;
}

export interface Product {
  id_producto: number;
  codigo: string;
  nombre: string;
  categoria: string;
  unidad_medida: string;
  precio_compra: number;
  precio_venta: number;
  margen_unitario: number;
  stock_actual: number;
  stock_reservado: number;
  stock_disponible: number;
  stock_minimo: number;
  estado_stock: string;
}

export interface ProductListResponse {
  status: string;
  total: number;
  filtro_stock_bajo: boolean;
  productos: Product[];
}

export interface StockAlert {
  id_alerta: number;
  codigo_producto: string;
  producto: string;
  tipo_alerta: string;
  stock_detectado: string;
  stock_minimo: string;
  estado: string;
  mensaje: string;
  fecha_generacion: string;
  fecha_atencion?: string;
}

export interface StockAlertListResponse {
  status: string;
  total: number;
  filtro_estado?: string;
  alertas: StockAlert[];
}

export interface SaleSummary {
  id_venta: number;
  numero_venta: string;
  cliente: string;
  fecha_venta: string;
  subtotal: string;
  descuento: string;
  impuesto: string;
  total: string;
  metodo_pago: string;
  estado: string;
  total_items: number;
}

export interface SaleListResponse {
  status: string;
  total: number;
  ventas: SaleSummary[];
}

export interface SaleDetailItem {
  codigo_producto: string;
  nombre_producto: string;
  cantidad: string;
  precio_unitario: string;
  descuento: string;
  subtotal_linea: string;
}

export interface SaleDetail {
  status: string;
  id_venta: number;
  numero_venta: string;
  cliente: string;
  identificacion_cliente: string;
  fecha_venta: string;
  subtotal: string;
  descuento: string;
  impuesto: string;
  total: string;
  metodo_pago: string;
  estado: string;
  observacion: string;
  items: SaleDetailItem[];
}

export interface SaleSummary {
  id_venta: number;
  numero_venta: string;
  cliente: string;
  fecha_venta: string;
  subtotal: string;
  descuento: string;
  impuesto: string;
  total: string;
  metodo_pago: string;
  estado: string;
  total_items: number;
}

export interface SaleListResponse {
  status: string;
  total: number;
  ventas: SaleSummary[];
}

export interface SaleDetailItem {
  codigo_producto: string;
  nombre_producto: string;
  cantidad: string;
  precio_unitario: string;
  descuento: string;
  subtotal_linea: string;
}

export interface SaleDetail {
  status: string;
  id_venta: number;
  numero_venta: string;
  cliente: string;
  identificacion_cliente: string;
  fecha_venta: string;
  subtotal: string;
  descuento: string;
  impuesto: string;
  total: string;
  metodo_pago: string;
  estado: string;
  observacion: string;
  items: SaleDetailItem[];
}

export type AlertManagementState = 'PENDIENTE' | 'ATENDIDA' | 'DESCARTADA';

export interface ManagedStockAlert {
  id_alerta: number;
  codigo_producto: string;
  producto: string;
  tipo_alerta: string;
  stock_detectado: string;
  stock_minimo: string;
  estado: AlertManagementState;
  mensaje: string;
  fecha_generacion: string;
  fecha_atencion?: string;
  observacion_atencion?: string;
  id_usuario_atencion?: number;
}

export interface ManagedStockAlertListResponse {
  status: string;
  total: number;
  filtro_estado: AlertManagementState;
  alertas: ManagedStockAlert[];
}

export interface UpdateStockAlertPayload {
  estado: Exclude<AlertManagementState, 'PENDIENTE'>;
  observacion: string;
  id_usuario?: number;
}

export interface UpdateStockAlertResponse extends ManagedStockAlert {
  status: string;
}
