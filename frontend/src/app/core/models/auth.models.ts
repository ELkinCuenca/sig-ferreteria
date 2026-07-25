export type RoleName = 'ADMINISTRADOR' | 'VENDEDOR' | 'BODEGUERO' | 'GERENTE';

export interface AuthUser {
  id_usuario: number;
  nombre_usuario: string;
  nombres: string;
  apellidos: string;
  correo: string;
  rol: RoleName;
}

export interface LoginPayload {
  usuario: string;
  contrasena: string;
}

export interface LoginResponse {
  status: string;
  token: string;
  tipo_token: 'Bearer';
  expira_en_segundos: number;
  usuario: AuthUser;
}

export interface ProfileResponse {
  status: string;
  usuario: AuthUser;
}

export interface LogoutResponse {
  status: string;
  message: string;
}
