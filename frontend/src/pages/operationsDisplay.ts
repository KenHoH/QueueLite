import { APIError } from '../api/errors';
import type { Counter, Queue } from '../api/types';
export const activeStates = ['waiting', 'called', 'processing'] as const;
export function groupQueues(queues: Queue[]) {
  return { waiting: queues.filter(q => q.state === 'waiting'), called: queues.filter(q => q.state === 'called'), processing: queues.filter(q => q.state === 'processing') };
}
export function currentQueue(counter: Counter, queues: Queue[]) { return queues.find(q => q.id === counter.currentQueueId); }
export function counterStatus(counter: Counter, queues: Queue[]) {
  if (!counter.currentQueueId) return 'Available';
  const queue = currentQueue(counter, queues);
  return queue?.state === 'called' ? 'Called' : queue?.state === 'processing' ? 'In service' : 'State pending — refresh';
}
export function operationsError(error: unknown) {
  if (error instanceof APIError) {
    if (error.status === 401) return 'Your session has ended. Please sign in again.';
    if (error.status === 403) return 'You do not have permission to access this business or counter.';
    if (error.code === 'QUEUE_NOT_READY') return 'Queue persistence is pending. Refresh state before trying again.';
    if (error.status === 409 || ['QUEUE_NOT_CALLED', 'QUEUE_COUNTER_MISMATCH', 'QUEUE_INVALID_STATE'].includes(error.code)) return 'Counter state changed. Refresh state before trying again.';
    if (error.code === 'COUNTER_NAME_REQUIRED') return 'Enter a counter name.';
    if (error.status === 404) return 'These details are not available. Refresh to check again.';
  }
  return 'We couldn’t confirm the operation. Refresh state before trying again.';
}
