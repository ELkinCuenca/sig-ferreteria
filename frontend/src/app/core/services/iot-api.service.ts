import { HttpClient, HttpParams } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';

import {
  AttendIoTAlertRequest,
  IoTAlertListResponse,
  IoTAlertStatus,
  IoTAlertUpdateResult,
  IoTConfigurationResponse,
  IoTReadingListResponse,
  IoTSummaryResponse,
  UpdateIoTConfigurationRequest,
} from '../models/iot.models';

@Injectable({
  providedIn: 'root',
})
export class IoTApiService {
  private readonly http = inject(HttpClient);

  private readonly baseUrl = '/api/v1/iot';

  getSummary(deviceCode = 'BODEGA-01'): Observable<IoTSummaryResponse> {
    const params = new HttpParams().set('dispositivo', deviceCode);

    return this.http.get<IoTSummaryResponse>(`${this.baseUrl}/resumen`, {
      params,
    });
  }

  getReadings(limit = 50, deviceCode = 'BODEGA-01'): Observable<IoTReadingListResponse> {
    const params = new HttpParams().set('dispositivo', deviceCode).set('limite', String(limit));

    return this.http.get<IoTReadingListResponse>(`${this.baseUrl}/lecturas`, {
      params,
    });
  }

  getAlerts(
    status: IoTAlertStatus | '' = '',
    limit = 100,
    deviceCode = 'BODEGA-01',
  ): Observable<IoTAlertListResponse> {
    let params = new HttpParams().set('dispositivo', deviceCode).set('limite', String(limit));

    if (status !== '') {
      params = params.set('estado', status);
    }

    return this.http.get<IoTAlertListResponse>(`${this.baseUrl}/alertas`, {
      params,
    });
  }

  getConfiguration(deviceCode = 'BODEGA-01'): Observable<IoTConfigurationResponse> {
    const params = new HttpParams().set('dispositivo', deviceCode);

    return this.http.get<IoTConfigurationResponse>(`${this.baseUrl}/configuracion`, {
      params,
    });
  }

  attendAlert(alertID: number, payload: AttendIoTAlertRequest): Observable<IoTAlertUpdateResult> {
    return this.http.patch<IoTAlertUpdateResult>(
      `${this.baseUrl}/alertas/${alertID}/atender`,
      payload,
    );
  }

  updateConfiguration(
    payload: UpdateIoTConfigurationRequest,
    deviceCode = 'BODEGA-01',
  ): Observable<IoTConfigurationResponse> {
    const params = new HttpParams().set('dispositivo', deviceCode);

    return this.http.patch<IoTConfigurationResponse>(`${this.baseUrl}/configuracion`, payload, {
      params,
    });
  }
}
