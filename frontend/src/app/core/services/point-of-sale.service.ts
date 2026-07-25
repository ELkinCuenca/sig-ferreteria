import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';

import { ProductListResponse } from '../models/sigefer.models';
import {
  ClientListResponse,
  SaleCreatePayload,
  SaleCreateResponse,
} from '../models/point-of-sale.models';

@Injectable({
  providedIn: 'root',
})
export class PointOfSaleService {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = '/api/v1';

  getClients(search = ''): Observable<ClientListResponse> {
    const normalizedSearch = search.trim();

    if (normalizedSearch !== '') {
      return this.http.get<ClientListResponse>(`${this.baseUrl}/clientes`, {
        params: {
          buscar: normalizedSearch,
        },
      });
    }

    return this.http.get<ClientListResponse>(`${this.baseUrl}/clientes`);
  }

  getProducts(): Observable<ProductListResponse> {
    return this.http.get<ProductListResponse>(`${this.baseUrl}/productos`);
  }

  createSale(payload: SaleCreatePayload): Observable<SaleCreateResponse> {
    return this.http.post<SaleCreateResponse>(`${this.baseUrl}/ventas`, payload);
  }
}
