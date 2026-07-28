import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';

import {
  CreateUserPayload,
  ResetUserPasswordPayload,
  RoleListResponse,
  UpdateUserPayload,
  UpdateUserStatePayload,
  UserAdmin,
  UserListResponse,
} from '../models/user-admin.models';

@Injectable({
  providedIn: 'root',
})
export class UserAdminApiService {
  private readonly http = inject(HttpClient);

  private readonly baseUrl = '/api/v1';

  getRoles(): Observable<RoleListResponse> {
    return this.http.get<RoleListResponse>(`${this.baseUrl}/roles`);
  }

  getUsers(): Observable<UserListResponse> {
    return this.http.get<UserListResponse>(`${this.baseUrl}/usuarios`);
  }

  getUser(userId: number): Observable<UserAdmin> {
    return this.http.get<UserAdmin>(`${this.baseUrl}/usuarios/${userId}`);
  }

  createUser(payload: CreateUserPayload): Observable<UserAdmin> {
    return this.http.post<UserAdmin>(`${this.baseUrl}/usuarios`, payload);
  }

  updateUser(userId: number, payload: UpdateUserPayload): Observable<UserAdmin> {
    return this.http.patch<UserAdmin>(`${this.baseUrl}/usuarios/${userId}`, payload);
  }

  updateState(userId: number, estado: 'ACTIVO' | 'INACTIVO'): Observable<UserAdmin> {
    const payload: UpdateUserStatePayload = {
      estado,
    };

    return this.http.patch<UserAdmin>(`${this.baseUrl}/usuarios/${userId}/estado`, payload);
  }

  unlockUser(userId: number): Observable<UserAdmin> {
    return this.http.patch<UserAdmin>(`${this.baseUrl}/usuarios/${userId}/desbloquear`, {});
  }

  resetPassword(userId: number, contrasena: string): Observable<UserAdmin> {
    const payload: ResetUserPasswordPayload = {
      contrasena,
    };

    return this.http.patch<UserAdmin>(`${this.baseUrl}/usuarios/${userId}/contrasena`, payload);
  }
}
