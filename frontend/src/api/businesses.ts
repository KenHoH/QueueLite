import { api, pathID, type RequestOptions } from './client';
import type { Business, BusinessMember, BusinessMembership, CursorPage, PageQuery, CreateBusinessRequest, UpdateBusinessRequest, UpsertBusinessMemberRequest, MessageResponse } from './types';
import { APIError } from './errors';
export const getMyBusinesses = async (options?: RequestOptions) => {
  const items = await api<BusinessMembership[]>('/businesses/mine', options);
  if (!Array.isArray(items)) throw new APIError(200, 'CLIENT_INVALID_RESPONSE', 'QueueLite returned an unexpected response.', 'response');
  return items;
};
/** Full business DTOs with the backend's opaque cursor; search returns a separate array. */
export const getBusinesses = (query: PageQuery = {}, options?: RequestOptions) => api<CursorPage<Business>>('/businesses/', { ...options, query });
export const searchBusinesses = (name: string, options?: RequestOptions) => api<Business[]>('/businesses/search', { ...options, query: { name } });
export const getBusiness = (businessId: string, options?: RequestOptions) => api<Business>(`/businesses/${pathID(businessId)}`, options);
export const createBusiness = (body: CreateBusinessRequest, options?: RequestOptions) => api<Business>('/businesses/', { ...options, method: 'POST', body });
export const updateBusiness = (businessId: string, body: UpdateBusinessRequest, options?: RequestOptions) => api<MessageResponse>(`/businesses/${pathID(businessId)}`, { ...options, method: 'PUT', body });
export const upsertBusinessMember = (businessId: string, body: UpsertBusinessMemberRequest, options?: RequestOptions) => api<BusinessMember>(`/businesses/${pathID(businessId)}/members`, { ...options, method: 'PUT', body });
export const deleteBusiness = (businessId: string, options?: RequestOptions) => api<MessageResponse>(`/businesses/${pathID(businessId)}`, { ...options, method: 'DELETE' });
