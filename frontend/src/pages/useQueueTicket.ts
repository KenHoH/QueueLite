import { useEffect, useState } from 'react';
import { customerQueueEventsURL, getQueue, getQueueState } from '../api/queues';
import { subscribeSSE } from '../api/sse';
import type { CustomerQueueStatus, Queue } from '../api/types';
import { isActiveQueue } from './queueDisplay';
import { isBusinessId } from './businessDisplay';
import { APIError } from '../api/errors';

/** Loads initial ticket details and subscribes to live Server-Sent Events (no client polling). */
export function useQueueTicket(queueId: string, paused: boolean) {
  const [queue, setQueue] = useState<Queue | null>(null);
  const [error, setError] = useState<unknown>(null);
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    if (paused) return;
    if (!isBusinessId(queueId)) {
      setError(new APIError(400, 'INVALID_QUEUE_ID', 'Invalid ticket link.'));
      return;
    }

    const controller = new AbortController();
    let unsubscribe: (() => void) | null = null;
    let disposed = false;

    setError(null);
    void Promise.all([
      getQueue(queueId, { signal: controller.signal }),
      getQueueState(queueId, { signal: controller.signal }),
    ]).then(([detail, state]) => {
      if (disposed || controller.signal.aborted) return;
      const next = { ...detail, state: state.state };
      setQueue(next);
      setError(null);
      if (!isActiveQueue(next.state)) return;

      unsubscribe = subscribeSSE({
        url: customerQueueEventsURL(queueId),
        event: 'status',
        onEvent: raw => {
          try {
            const parsed = JSON.parse(raw) as CustomerQueueStatus;
            if (!parsed?.queue) return;
            setQueue(parsed.queue);
            setError(null);
            if (!isActiveQueue(parsed.queue.state)) {
              unsubscribe?.();
              unsubscribe = null;
            }
          } catch {
            // Ignore malformed SSE frame.
          }
        },
      });
    }).catch(failure => {
      if (!disposed && !controller.signal.aborted) setError(failure);
    });

    return () => {
      disposed = true;
      controller.abort();
      unsubscribe?.();
    };
  }, [queueId, paused, attempt]);

  return { queue, error, refresh: () => setAttempt(value => value + 1) };
}
