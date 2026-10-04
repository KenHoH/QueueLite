import { api, type RequestOptions } from './client';
import type { LoginRequest, RegisterUserRequest, MessageResponse, User } from './types';
/** Sets HttpOnly token; response does not include user identity. */
export const login = (body: LoginRequest, options?: RequestOptions) => api<MessageResponse>('/users/login', { ...options, method: 'POST', body });
/** Registration does not log in. */
export const registerUser = (body: RegisterUserRequest, options?: RequestOptions) => api<User>('/users/', { ...options, method: 'POST', body });
export const getCurrentUser = (options?: RequestOptions) => api<User>('/users/me', options);
export const logout = (options?: RequestOptions) => api<MessageResponse>('/users/logout', { ...options, method: 'POST' });
