import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';

import {
  DashboardSummary,
  ProductListResponse,
  SaleDetail,
  SaleListResponse,
  StockAlertListResponse,
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

  getLowStockProducts(): Observable<ProductListResponse> {
    return this.http.get<ProductListResponse>(`${this.baseUrl}/productos`, {
      params: {
        stock_bajo: true,
      },
    });
  }

  getPendingAlerts(): Observable<StockAlertListResponse> {
    return this.http.get<StockAlertListResponse>(`${this.baseUrl}/alertas-stock`, {
      params: {
        estado: 'PENDIENTE',
      },
    });
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
