import { inject, Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

import {
  DashboardSummary,
  ProductListResponse,
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
    return this.http.get<ProductListResponse>(`${this.baseUrl}/productos?stock_bajo=true`);
  }

  getPendingAlerts(): Observable<StockAlertListResponse> {
    return this.http.get<StockAlertListResponse>(`${this.baseUrl}/alertas-stock?estado=PENDIENTE`);
  }
}
