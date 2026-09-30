import { api, pathID, type RequestOptions } from './client';
import type { Counter, CreateCounterRequest, UpdateCounterRequest, MessageResponse, NextQueueResponse, ProcessQueueResponse } from './types';
export const getCounter = (counterId: string, options?: RequestOptions) => api<Counter>(`/counters/${pathID(counterId)}`, options);
export const createCounter = (body: CreateCounterRequest, options?: RequestOptions) => api<Counter>('/counters/', { ...options, method: 'POST', body });
export const updateCounter = (counterId: string, body: UpdateCounterRequest, options?: RequestOptions) => api<MessageResponse>(`/counters/${pathID(counterId)}`, { ...options, method: 'PUT', body });
export const deleteCounter = (counterId: string, options?: RequestOptions) => api<MessageResponse>(`/counters/${pathID(counterId)}`, { ...options, method: 'DELETE' });
/** May complete an occupied counter's current queue before calling the next. */
export const callNextQueue = (counterId: string, businessId: string, options?: RequestOptions) => api<NextQueueResponse>(`/counters/${pathID(counterId)}/business/${pathID(businessId)}/call-next`, { ...options, method: 'POST' });
export const processCalledQueue = (counterId: string, queueId: string, options?: RequestOptions) => api<ProcessQueueResponse>(`/counters/${pathID(counterId)}/queues/${pathID(queueId)}/process`, { ...options, method: 'POST' });
/** Returns the NEXT queue, not the skipped queue. */
export const skipQueue = (counterId: string, queueId: string, options?: RequestOptions) => api<NextQueueResponse>(`/counters/${pathID(counterId)}/queues/${pathID(queueId)}/skip`, { ...options, method: 'POST' });
/** Completes the queue and clears the counter assignment. */
export const removeQueueFromCounter = (counterId: string, queueId: string, options?: RequestOptions) => api<MessageResponse>(`/counters/${pathID(counterId)}/queues/${pathID(queueId)}`, { ...options, method: 'DELETE' });
