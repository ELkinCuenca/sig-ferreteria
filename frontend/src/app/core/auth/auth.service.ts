import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { catchError, map, Observable, of, tap } from 'rxjs';

import {
  LoginPayload,
  LoginResponse,
  LogoutResponse,
  ProfileResponse,
} from '../models/auth.models';
import { AuthSessionService } from './auth-session.service';

@Injectable({
  providedIn: 'root',
})
export class AuthService {
  private readonly http = inject(HttpClient);

  private readonly session = inject(AuthSessionService);

  private readonly baseUrl = '/api/v1/auth';

  login(payload: LoginPayload): Observable<LoginResponse> {
    return this.http.post<LoginResponse>(`${this.baseUrl}/login`, payload).pipe(
      tap((response) => {
        this.session.establish(response.token, response.usuario);
      }),
    );
  }

  profile(): Observable<ProfileResponse> {
    return this.http.get<ProfileResponse>(`${this.baseUrl}/perfil`).pipe(
      tap((response) => {
        this.session.updateUser(response.usuario);

        this.session.markValidated();
      }),
    );
  }

  ensureSession(): Observable<boolean> {
    if (!this.session.token()) {
      this.session.clear();
      return of(false);
    }

    if (this.session.isAuthenticated() && this.session.validated()) {
      return of(true);
    }

    return this.profile().pipe(
      map(() => true),

      catchError(() => {
        this.session.clear();
        return of(false);
      }),
    );
  }

  logout(): Observable<LogoutResponse> {
    return this.http.post<LogoutResponse>(`${this.baseUrl}/logout`, {}).pipe(
      tap(() => {
        this.session.clear();
      }),

      catchError((error: unknown) => {
        this.session.clear();
        throw error;
      }),
    );
  }
}
