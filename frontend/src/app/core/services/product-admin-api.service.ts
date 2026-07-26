import { HttpClient, HttpParams } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';

import {
  CategoryListResponse,
  CreateProductPayload,
  InventoryAdjustmentPayload,
  InventoryAdjustmentResult,
  InventoryMovementFilters,
  InventoryMovementListResponse,
  ProductDetail,
  ProductState,
  UpdateProductPayload,
  UpdateProductStatePayload,
} from '../models/product-admin.models';

@Injectable({
  providedIn: 'root',
})
export class ProductAdminApiService {
  private readonly http = inject(HttpClient);

  private readonly baseUrl = '/api/v1';

  getCategories(): Observable<CategoryListResponse> {
    return this.http.get<CategoryListResponse>(`${this.baseUrl}/categorias`);
  }

  getProduct(code: string): Observable<ProductDetail> {
    return this.http.get<ProductDetail>(`${this.baseUrl}/productos/${encodeURIComponent(code)}`);
  }

  createProduct(payload: CreateProductPayload): Observable<ProductDetail> {
    return this.http.post<ProductDetail>(`${this.baseUrl}/productos`, payload);
  }

  updateProduct(code: string, payload: UpdateProductPayload): Observable<ProductDetail> {
    return this.http.patch<ProductDetail>(
      `${this.baseUrl}/productos/${encodeURIComponent(code)}`,
      payload,
    );
  }

  updateProductState(code: string, state: ProductState): Observable<ProductDetail> {
    const payload: UpdateProductStatePayload = {
      estado: state,
    };

    return this.http.patch<ProductDetail>(
      `${this.baseUrl}/productos/${encodeURIComponent(code)}/estado`,
      payload,
    );
  }

  adjustInventory(payload: InventoryAdjustmentPayload): Observable<InventoryAdjustmentResult> {
    return this.http.post<InventoryAdjustmentResult>(`${this.baseUrl}/inventario/ajustes`, payload);
  }

  getInventoryMovements(
    filters: InventoryMovementFilters = {},
  ): Observable<InventoryMovementListResponse> {
    let params = new HttpParams();

    if (filters.limite !== undefined) {
      params = params.set('limite', filters.limite.toString());
    }

    if (filters.codigo) {
      params = params.set('codigo', filters.codigo);
    }

    if (filters.tipo) {
      params = params.set('tipo', filters.tipo);
    }

    return this.http.get<InventoryMovementListResponse>(`${this.baseUrl}/inventario/movimientos`, {
      params,
    });
  }
}
