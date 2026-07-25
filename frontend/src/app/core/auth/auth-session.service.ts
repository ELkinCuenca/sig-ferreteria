import { computed, Injectable, signal } from '@angular/core';

import { AuthUser, RoleName } from '../models/auth.models';

const tokenStorageKey = 'sigefer.auth.token';

const userStorageKey = 'sigefer.auth.user';

@Injectable({
  providedIn: 'root',
})
export class AuthSessionService {
  private readonly tokenState = signal<string | null>(this.readStoredToken());

  private readonly userState = signal<AuthUser | null>(this.readStoredUser());

  private readonly validatedState = signal(false);

  readonly token = this.tokenState.asReadonly();

  readonly user = this.userState.asReadonly();

  readonly validated = this.validatedState.asReadonly();

  readonly isAuthenticated = computed(() => {
    return this.tokenState() !== null && this.userState() !== null;
  });

  readonly fullName = computed(() => {
    const user = this.userState();

    if (!user) {
      return '';
    }

    return [user.nombres, user.apellidos].filter(Boolean).join(' ').trim();
  });

  establish(token: string, user: AuthUser): void {
    this.tokenState.set(token);
    this.userState.set(user);
    this.validatedState.set(true);

    try {
      sessionStorage.setItem(tokenStorageKey, token);

      sessionStorage.setItem(userStorageKey, JSON.stringify(user));
    } catch {
      // La sesión continúa en memoria aunque
      // el navegador impida usar sessionStorage.
    }
  }

  updateUser(user: AuthUser): void {
    this.userState.set(user);

    try {
      sessionStorage.setItem(userStorageKey, JSON.stringify(user));
    } catch {
      // La copia en memoria sigue disponible.
    }
  }

  markValidated(): void {
    this.validatedState.set(true);
  }

  clear(): void {
    this.tokenState.set(null);
    this.userState.set(null);
    this.validatedState.set(false);

    try {
      sessionStorage.removeItem(tokenStorageKey);

      sessionStorage.removeItem(userStorageKey);
    } catch {
      // No se requiere otra acción.
    }
  }

  hasRole(role: RoleName): boolean {
    return this.userState()?.rol === role;
  }

  hasAnyRole(roles: readonly RoleName[]): boolean {
    const currentRole = this.userState()?.rol;

    return currentRole !== undefined && roles.includes(currentRole);
  }

  private readStoredToken(): string | null {
    try {
      const token = sessionStorage.getItem(tokenStorageKey);

      if (!token || token.length < 32 || token.length > 512) {
        return null;
      }

      return token;
    } catch {
      return null;
    }
  }

  private readStoredUser(): AuthUser | null {
    try {
      const value = sessionStorage.getItem(userStorageKey);

      if (!value) {
        return null;
      }

      const user = JSON.parse(value) as Partial<AuthUser>;

      if (
        typeof user.id_usuario !== 'number' ||
        typeof user.nombre_usuario !== 'string' ||
        typeof user.nombres !== 'string' ||
        typeof user.apellidos !== 'string' ||
        typeof user.correo !== 'string' ||
        !this.isValidRole(user.rol)
      ) {
        return null;
      }

      return user as AuthUser;
    } catch {
      return null;
    }
  }

  private isValidRole(value: unknown): value is RoleName {
    return (
      value === 'ADMINISTRADOR' ||
      value === 'VENDEDOR' ||
      value === 'BODEGUERO' ||
      value === 'GERENTE'
    );
  }
}
