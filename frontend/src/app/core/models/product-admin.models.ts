export type ProductState = 'A' | 'I';

export type InventoryAdjustmentType = 'POSITIVO' | 'NEGATIVO';

export type InventoryMovementType =
  | 'ENTRADA_COMPRA'
  | 'SALIDA_VENTA'
  | 'AJUSTE_POSITIVO'
  | 'AJUSTE_NEGATIVO'
  | 'DEVOLUCION_VENTA'
  | 'DEVOLUCION_COMPRA';

export interface ProductCategory {
  id_categoria: number;
  nombre: string;
  descripcion?: string;
  estado: ProductState;
}

export interface CategoryListResponse {
  status: string;
  total: number;
  categorias: ProductCategory[];
}

export interface ProductDetail {
  id_producto: number;
  id_categoria: number;

  codigo: string;
  nombre: string;
  descripcion?: string;

  categoria: string;
  unidad_medida: string;
  estado: ProductState;
  ubicacion?: string;

  precio_compra: string;
  precio_venta: string;
  margen_unitario: string;

  stock_actual: string;
  stock_reservado: string;
  stock_disponible: string;
  stock_minimo: string;

  estado_stock: string;

  fecha_creacion: string;
  fecha_actualizacion: string;
}

export interface CreateProductPayload {
  id_categoria: number;

  codigo: string;
  nombre: string;
  descripcion?: string;

  unidad_medida: string;
  ubicacion?: string;

  precio_compra: string;
  precio_venta: string;

  stock_minimo: string;
  stock_inicial: string;
}

export interface UpdateProductPayload {
  id_categoria: number;

  nombre: string;
  descripcion?: string;

  unidad_medida: string;
  ubicacion?: string;

  precio_compra: string;
  precio_venta: string;
  stock_minimo: string;
}

export interface UpdateProductStatePayload {
  estado: ProductState;
}

export interface InventoryAdjustmentPayload {
  codigo_producto: string;
  tipo_ajuste: InventoryAdjustmentType;
  cantidad: string;
  motivo: string;
}

export interface InventoryAdjustmentResult {
  status: string;
  id_movimiento: number;

  codigo_producto: string;
  producto: string;
  tipo_movimiento: InventoryMovementType;

  cantidad: string;
  stock_anterior: string;
  stock_nuevo: string;
  stock_reservado: string;
  stock_disponible: string;
  stock_minimo: string;

  estado_stock: string;
  resultado_alerta: string;

  motivo: string;
  fecha_movimiento: string;
}

export interface InventoryMovement {
  id_movimiento: number;
  id_producto: number;

  codigo_producto: string;
  producto: string;
  tipo_movimiento: InventoryMovementType;

  cantidad: string;
  stock_anterior: string;
  stock_nuevo: string;

  motivo?: string;

  id_usuario?: number;
  usuario: string;

  id_venta?: number;
  id_solicitud_reposicion?: number;

  referencia?: string;
  fecha_movimiento: string;
}

export interface InventoryMovementListResponse {
  status: string;
  total: number;
  movimientos: InventoryMovement[];
}

export interface InventoryMovementFilters {
  limite?: number;
  codigo?: string;
  tipo?: InventoryMovementType | '';
}
