import { api, pathID, type RequestOptions } from './client';
import type { User, UpdateUserRequest, MessageResponse } from './types';
export const getUser = (userId: string, options?: RequestOptions) => api<User>(`/users/${pathID(userId)}`, options);
export const updateUser = (userId: string, body: UpdateUserRequest, options?: RequestOptions) => api<MessageResponse>(`/users/${pathID(userId)}`, { ...options, method: 'PUT', body });
