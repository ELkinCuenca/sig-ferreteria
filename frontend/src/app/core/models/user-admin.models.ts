export type UserState = 'ACTIVO' | 'INACTIVO' | 'BLOQUEADO';

export interface UserRole {
  id_rol: number;
  nombre: string;
  descripcion?: string;
  estado: string;
}

export interface RoleListResponse {
  status: string;
  total: number;
  roles: UserRole[];
}

export interface UserAdmin {
  id_usuario: number;
  id_rol: number;

  nombre_usuario: string;
  nombres: string;
  apellidos: string;
  correo: string;

  rol: string;
  estado: UserState;

  intentos_fallidos: number;

  ultimo_acceso?: string;
  ultimo_intento_fallido?: string;
  fecha_bloqueo?: string;

  fecha_creacion: string;
  fecha_actualizacion: string;
}

export interface UserListResponse {
  status: string;
  total: number;
  usuarios: UserAdmin[];
}

export interface CreateUserPayload {
  id_rol: number;

  nombre_usuario: string;
  nombres: string;
  apellidos: string;
  correo: string;

  contrasena: string;
}

export interface UpdateUserPayload {
  id_rol: number;

  nombres: string;
  apellidos: string;
  correo: string;
}

export interface UpdateUserStatePayload {
  estado: 'ACTIVO' | 'INACTIVO';
}

export interface ResetUserPasswordPayload {
  contrasena: string;
}

export interface ApiErrorResponse {
  status: string;
  message: string;
}
