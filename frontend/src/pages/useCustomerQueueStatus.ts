import { useEffect, useState } from 'react';
import { APIError } from '../api/errors';
import { customerQueueEventsURL, getCustomerQueueStatus } from '../api/queues';
import { subscribeSSE, type SSEConnectionStatus } from '../api/sse';
import type { CustomerQueueStatus } from '../api/types';
import { isBusinessId } from './businessDisplay';
import { isActiveQueue, isQueuePersistencePending } from './queueDisplay';

/** Loads initial queue status and subscribes to live Server-Sent Events (no client polling). */
export function useCustomerQueueStatus(queueId: string, paused: boolean) {
  const [data, setData] = useState<CustomerQueueStatus | null>(null);
  const [error, setError] = useState<unknown>(null);
  const [persistencePending, setPersistencePending] = useState(false);
  const [streamStatus, setStreamStatus] = useState<SSEConnectionStatus>('closed');
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    if (paused) {
      setStreamStatus('closed');
      return;
    }
    if (!isBusinessId(queueId)) {
      setError(new APIError(400, 'INVALID_QUEUE_ID', 'Invalid ticket link.'));
      setStreamStatus('closed');
      return;
    }

    const controller = new AbortController();
    let unsubscribe: (() => void) | null = null;
    let disposed = false;

    const startStream = () => {
      if (disposed || unsubscribe) return;
      unsubscribe = subscribeSSE({
        url: customerQueueEventsURL(queueId),
        event: 'status',
        onStatus: nextStatus => {
          if (!disposed) setStreamStatus(nextStatus);
        },
        onEvent: raw => {
          try {
            const parsed = JSON.parse(raw) as CustomerQueueStatus;
            if (!parsed?.queue) return;
            setData(parsed);
            setError(null);
            setPersistencePending(false);
            if (!isActiveQueue(parsed.queue.state)) {
              unsubscribe?.();
              unsubscribe = null;
              setStreamStatus('closed');
            }
          } catch {
            // Ignore malformed SSE frame.
          }
        },
      });
    };

    setError(null);
    setPersistencePending(false);
    setStreamStatus('closed');
    void getCustomerQueueStatus(queueId, { signal: controller.signal }).then(next => {
      if (disposed || controller.signal.aborted) return;
      setData(next);
      setError(null);
      setPersistencePending(false);
      if (isActiveQueue(next.queue.state)) startStream();
    }).catch(failure => {
      if (disposed || controller.signal.aborted) return;
      if (isQueuePersistencePending(failure)) {
        setError(null);
        setPersistencePending(true);
        startStream();
      } else {
        setPersistencePending(false);
        setError(failure);
      }
    });

    return () => {
      disposed = true;
      controller.abort();
      unsubscribe?.();
    };
  }, [queueId, paused, attempt]);

  return { data, error, persistencePending, streamStatus, refresh: () => setAttempt(value => value + 1) };
}
