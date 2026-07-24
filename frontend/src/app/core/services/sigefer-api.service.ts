import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';

import {
  AlertManagementState,
  DashboardSummary,
  ManagedStockAlertListResponse,
  ProductListResponse,
  SaleDetail,
  SaleListResponse,
  StockAlertListResponse,
  UpdateStockAlertPayload,
  UpdateStockAlertResponse,
} from '../models/sigefer.models';

@Injectable({
  providedIn: 'root',
})
export class SigeferApiService {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = '/api/v1';

  getDashboard(): Observable<DashboardSummary> {
    return this.http.get<DashboardSummary>(`${this.baseUrl}/dashboard/resumen`);
  }

  getProducts(stockLowOnly = false): Observable<ProductListResponse> {
    const url = `${this.baseUrl}/productos`;

    if (stockLowOnly) {
      return this.http.get<ProductListResponse>(url, {
        params: {
          stock_bajo: 'true',
        },
      });
    }

    return this.http.get<ProductListResponse>(url);
  }

  getLowStockProducts(): Observable<ProductListResponse> {
    return this.getProducts(true);
  }

  getPendingAlerts(): Observable<StockAlertListResponse> {
    return this.http.get<StockAlertListResponse>(`${this.baseUrl}/alertas-stock`, {
      params: {
        estado: 'PENDIENTE',
      },
    });
  }

  getManagedStockAlerts(state: AlertManagementState): Observable<ManagedStockAlertListResponse> {
    return this.http.get<ManagedStockAlertListResponse>(`${this.baseUrl}/alertas-stock`, {
      params: {
        estado: state,
      },
    });
  }

  updateStockAlert(
    alertId: number,
    payload: UpdateStockAlertPayload,
  ): Observable<UpdateStockAlertResponse> {
    return this.http.patch<UpdateStockAlertResponse>(
      `${this.baseUrl}/alertas-stock/${alertId}`,
      payload,
    );
  }

  getSales(limit = 50): Observable<SaleListResponse> {
    return this.http.get<SaleListResponse>(`${this.baseUrl}/ventas`, {
      params: {
        limite: limit,
      },
    });
  }

  getSaleByNumber(saleNumber: string): Observable<SaleDetail> {
    return this.http.get<SaleDetail>(`${this.baseUrl}/ventas/${encodeURIComponent(saleNumber)}`);
  }
}
