import { api, pathID, type RequestOptions } from './client';
import type { Queue, QueueState, QueueQRResolution, GuestQueueRequest, CustomerQueueJoinRequest, CreateQueueRequest, UpdateQueueRequest, MessageResponse, CustomerQueueStatus } from './types';
export const resolveQueueQR = (businessId: string, options?: RequestOptions) => api<QueueQRResolution>(`/queues/qr/${pathID(businessId)}/resolve`, options);
/** Guest body required when unauthenticated; authenticated requests ignore the body. */
export const registerQueueByQR = (businessId: string, body?: GuestQueueRequest, options?: RequestOptions) => api<Queue>(`/queues/qr/${pathID(businessId)}`, { ...options, method: 'POST', body });
/** Legacy alias; prefer joinBusinessQueue. */
export const registerQueue = (body: CreateQueueRequest, options?: RequestOptions) => api<Queue>('/queues/', { ...options, method: 'POST', body });
/** Account identity comes from HttpOnly login; guests supply only name and phone. */
export const joinBusinessQueue = (businessId: string, body?: CustomerQueueJoinRequest, options?: RequestOptions) => api<Queue>(`/queues/business/${pathID(businessId)}/join`, { ...options, method: 'POST', body });
export const getQueue = (queueId: string, options?: RequestOptions) => api<Queue>(`/queues/${pathID(queueId)}`, options);
export const getQueueState = (queueId: string, options?: RequestOptions) => api<{ state: QueueState }>(`/queues/${pathID(queueId)}/state`, options);
export const getCustomerQueueStatus = (queueId: string, options?: RequestOptions) => api<CustomerQueueStatus>(`/queues/${pathID(queueId)}/customer-status`, options);
export const getQueuesByBusiness = (businessId: string, options?: RequestOptions) => api<Queue[]>(`/queues/business/${pathID(businessId)}`, options);
export const getMyQueues = (options?: RequestOptions) => api<Queue[]>('/queues/me', options);
export const updateQueue = (queueId: string, body: UpdateQueueRequest, options?: RequestOptions) => api<MessageResponse>(`/queues/${pathID(queueId)}`, { ...options, method: 'PUT', body });
export const updateQueueState = (queueId: string, state: QueueState, options?: RequestOptions) => api<MessageResponse>(`/queues/${pathID(queueId)}/state`, { ...options, method: 'PATCH', body: { state } });
export const markQueueDone = (queueId: string, options?: RequestOptions) => api<MessageResponse>(`/queues/${pathID(queueId)}/done`, { ...options, method: 'PATCH' });
/** Hard delete, not a cancellation transition. */
export const deleteQueue = (queueId: string, options?: RequestOptions) => api<MessageResponse>(`/queues/${pathID(queueId)}`, { ...options, method: 'DELETE' });
