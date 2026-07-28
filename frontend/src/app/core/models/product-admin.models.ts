export type ProductState = 'A' | 'I';

export type ProductStateFilter = ProductState | 'TODOS';

export type DecimalValue = number | string;

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
  descripcion?: string | null;
  estado?: string;
}

export interface CategoryListResponse {
  status: string;
  total: number;
  categorias: ProductCategory[];
}

export interface ProductSummary {
  id_producto: number;

  codigo: string;
  nombre: string;
  categoria: string;
  unidad_medida: string;
  estado: ProductState;

  precio_compra: DecimalValue;
  precio_venta: DecimalValue;
  margen_unitario: DecimalValue;

  stock_actual: DecimalValue;
  stock_reservado: DecimalValue;
  stock_disponible: DecimalValue;
  stock_minimo: DecimalValue;

  estado_stock: string;
}

export interface ProductAdminListResponse {
  status: string;
  total: number;

  filtro_stock_bajo: boolean;
  filtro_estado: ProductStateFilter;

  productos: ProductSummary[];
}

export interface ProductDetail {
  id_producto: number;
  id_categoria: number;

  categoria: string;
  codigo: string;
  nombre: string;
  descripcion?: string | null;
  unidad_medida: string;

  precio_compra: DecimalValue;
  precio_venta: DecimalValue;

  stock_minimo: DecimalValue;
  stock_actual: DecimalValue;
  stock_reservado: DecimalValue;
  stock_disponible: DecimalValue;

  ubicacion?: string | null;
  estado: ProductState;

  fecha_creacion?: string;
  fecha_actualizacion?: string;
}

export interface ProductDetailEnvelope {
  status: string;
  producto: ProductDetail;
}

export type ProductDetailApiResponse = ProductDetail | ProductDetailEnvelope;

export interface CreateProductPayload {
  id_categoria: number;

  codigo: string;
  nombre: string;
  descripcion?: string;
  unidad_medida: string;

  precio_compra: number;
  precio_venta: number;

  stock_minimo: number;
  stock_inicial: number;

  ubicacion?: string;
}

export interface UpdateProductPayload {
  id_categoria: number;

  nombre: string;
  descripcion?: string;
  unidad_medida: string;

  precio_compra: number;
  precio_venta: number;

  stock_minimo: number;
  ubicacion?: string;
}

export interface UpdateProductStatePayload {
  estado: ProductState;
}

export interface InventoryAdjustmentPayload {
  codigo: string;
  tipo_ajuste: InventoryAdjustmentType;
  cantidad: number;
  motivo: string;
}

export interface InventoryAdjustmentResult {
  status?: string;

  codigo: string;
  tipo_ajuste: InventoryAdjustmentType;

  cantidad: DecimalValue;
  stock_anterior: DecimalValue;
  stock_nuevo: DecimalValue;

  movimiento?: InventoryMovement;
}

export interface InventoryMovement {
  id_movimiento: number;
  id_producto?: number;

  codigo_producto: string;
  producto: string;

  tipo_movimiento: InventoryMovementType;

  cantidad: DecimalValue;
  stock_anterior: DecimalValue;
  stock_nuevo: DecimalValue;

  motivo?: string | null;
  usuario?: string | null;

  fecha_movimiento: string;
}

export interface InventoryMovementListResponse {
  status: string;
  total: number;
  movimientos: InventoryMovement[];
}

export interface InventoryMovementFilters {
  codigo?: string;
  tipo?: InventoryMovementType | '';
  limite?: number;
}

export interface ProductApiErrorResponse {
  status: string;
  message: string;
}
