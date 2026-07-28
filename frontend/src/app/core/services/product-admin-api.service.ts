import { HttpClient, HttpParams } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { map, Observable } from 'rxjs';

import type {
  CategoryListResponse,
  CreateProductPayload,
  InventoryAdjustmentPayload,
  InventoryAdjustmentResult,
  InventoryMovementFilters,
  InventoryMovementListResponse,
  ProductAdminListResponse,
  ProductDetail,
  ProductDetailApiResponse,
  ProductState,
  ProductStateFilter,
  UpdateProductPayload,
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

  getProducts(
    state: ProductStateFilter = 'TODOS',
    stockLowOnly = false,
  ): Observable<ProductAdminListResponse> {
    const params = new HttpParams().set('estado', state).set('stock_bajo', String(stockLowOnly));

    return this.http.get<ProductAdminListResponse>(`${this.baseUrl}/productos`, {
      params,
    });
  }

  getProduct(code: string): Observable<ProductDetail> {
    return this.http
      .get<ProductDetailApiResponse>(`${this.baseUrl}/productos/${encodeURIComponent(code)}`)
      .pipe(map((response) => this.unwrapProduct(response)));
  }

  createProduct(payload: CreateProductPayload): Observable<ProductDetail> {
    return this.http
      .post<ProductDetailApiResponse>(`${this.baseUrl}/productos`, payload)
      .pipe(map((response) => this.unwrapProduct(response)));
  }

  updateProduct(code: string, payload: UpdateProductPayload): Observable<ProductDetail> {
    return this.http
      .patch<ProductDetailApiResponse>(
        `${this.baseUrl}/productos/${encodeURIComponent(code)}`,
        payload,
      )
      .pipe(map((response) => this.unwrapProduct(response)));
  }

  updateProductState(code: string, state: ProductState): Observable<ProductDetail> {
    return this.http
      .patch<ProductDetailApiResponse>(
        `${this.baseUrl}/productos/${encodeURIComponent(code)}/estado`,
        {
          estado: state,
        },
      )
      .pipe(map((response) => this.unwrapProduct(response)));
  }

  adjustInventory(payload: InventoryAdjustmentPayload): Observable<InventoryAdjustmentResult> {
    return this.http.post<InventoryAdjustmentResult>(`${this.baseUrl}/inventario/ajustes`, payload);
  }

  getInventoryMovements(
    filters: InventoryMovementFilters = {},
  ): Observable<InventoryMovementListResponse> {
    let params = new HttpParams();

    if (filters.codigo) {
      params = params.set('codigo', filters.codigo);
    }

    if (filters.tipo) {
      params = params.set('tipo', filters.tipo);
    }

    if (filters.limite) {
      params = params.set('limite', String(filters.limite));
    }

    return this.http.get<InventoryMovementListResponse>(`${this.baseUrl}/inventario/movimientos`, {
      params,
    });
  }

  private unwrapProduct(response: ProductDetailApiResponse): ProductDetail {
    if ('producto' in response) {
      return response.producto;
    }

    return response;
  }
}
