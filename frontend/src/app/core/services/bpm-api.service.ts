import { HttpClient, HttpParams } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';

import {
  BPMCreatePayload,
  BPMListResponse,
  BPMProviderListResponse,
  BPMReceivePayload,
  BPMRejectPayload,
  BPMReplenishmentDetail,
  BPMState,
  BPMTransitionPayload,
} from '../models/bpm.models';

@Injectable({
  providedIn: 'root',
})
export class BPMApiService {
  private readonly http = inject(HttpClient);

  private readonly baseUrl = '/api/v1/bpm';

  getProviders(): Observable<BPMProviderListResponse> {
    return this.http.get<BPMProviderListResponse>(`${this.baseUrl}/proveedores`);
  }

  getReplenishments(state?: BPMState, limit = 200): Observable<BPMListResponse> {
    let params = new HttpParams().set('limite', limit);

    if (state) {
      params = params.set('estado', state);
    }

    return this.http.get<BPMListResponse>(`${this.baseUrl}/reposiciones`, {
      params,
    });
  }

  getReplenishment(number: string): Observable<BPMReplenishmentDetail> {
    return this.http.get<BPMReplenishmentDetail>(
      `${this.baseUrl}/reposiciones/${encodeURIComponent(number)}`,
    );
  }

  create(payload: BPMCreatePayload): Observable<BPMReplenishmentDetail> {
    return this.http.post<BPMReplenishmentDetail>(`${this.baseUrl}/reposiciones`, payload);
  }

  send(number: string, payload: BPMTransitionPayload): Observable<BPMReplenishmentDetail> {
    return this.transition(number, 'enviar', payload);
  }

  approve(number: string, payload: BPMTransitionPayload): Observable<BPMReplenishmentDetail> {
    return this.transition(number, 'aprobar', payload);
  }

  reject(number: string, payload: BPMRejectPayload): Observable<BPMReplenishmentDetail> {
    return this.transition(number, 'rechazar', payload);
  }

  markOrder(number: string, payload: BPMTransitionPayload): Observable<BPMReplenishmentDetail> {
    return this.transition(number, 'pedido', payload);
  }

  receive(number: string, payload: BPMReceivePayload): Observable<BPMReplenishmentDetail> {
    return this.transition(number, 'recibir', payload);
  }

  close(number: string, payload: BPMTransitionPayload): Observable<BPMReplenishmentDetail> {
    return this.transition(number, 'cerrar', payload);
  }

  private transition<TPayload>(
    number: string,
    action: string,
    payload: TPayload,
  ): Observable<BPMReplenishmentDetail> {
    return this.http.patch<BPMReplenishmentDetail>(
      `${this.baseUrl}/reposiciones/${encodeURIComponent(number)}/${action}`,
      payload,
    );
  }
}
