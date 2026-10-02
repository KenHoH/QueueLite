import { api, pathID, type RequestOptions } from './client';
import { APIError } from './errors';
import type { BusinessMember, Counter, Queue } from './types';

async function list<T>(path: string, options?: RequestOptions): Promise<T[]> {
  const result = await api<T[]>(path, options);
  if (!Array.isArray(result)) throw new APIError(200, 'CLIENT_INVALID_RESPONSE', 'Unexpected operations response.', 'response');
  return result;
}
export const getBusinessCounters = (businessId: string, options?: RequestOptions) => list<Counter>(`/businesses/${pathID(businessId)}/counters`, options);
export const getBusinessMembers = (businessId: string, options?: RequestOptions) => list<BusinessMember>(`/businesses/${pathID(businessId)}/members`, options);
export const getBusinessQueues = (businessId: string, options?: RequestOptions) => list<Queue>(`/businesses/${pathID(businessId)}/queues`, options);
