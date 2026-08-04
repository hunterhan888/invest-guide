import { request } from '../client';
import type { AuthResponse, LoginRequest, RegisterRequest } from './types';

export function login(req: LoginRequest): Promise<AuthResponse> {
  return request<AuthResponse>('POST', '/auth/login', req);
}

export function register(req: RegisterRequest): Promise<AuthResponse> {
  return request<AuthResponse>('POST', '/auth/register', req);
}
