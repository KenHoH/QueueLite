import { useEffect, useState } from 'react';
import { APIError } from '../api/errors';
import { getCustomerQueueStatus } from '../api/queues';
import type { CustomerQueueStatus } from '../api/types';
import { isBusinessId } from './businessDisplay';
import { isActiveQueue, isQueuePersistencePending } from './queueDisplay';

/** Polls without overlapping requests and pauses while the tab is hidden. */
export function useCustomerQueueStatus(queueId: string, paused: boolean) {
  const [data, setData] = useState<CustomerQueueStatus | null>(null);
  const [error, setError] = useState<unknown>(null);
  const [persistencePending, setPersistencePending] = useState(false);
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    if (paused) return;
    if (!isBusinessId(queueId)) {
      setError(new APIError(400, 'INVALID_QUEUE_ID', 'Invalid ticket link.'));
      return;
    }
    const controller = new AbortController();
    let timer: ReturnType<typeof setTimeout> | undefined;
    let pending = false;
    let active = true;
    const read = async () => {
      if (pending || controller.signal.aborted || document.hidden) return;
      pending = true;
      try {
        const next = await getCustomerQueueStatus(queueId, { signal: controller.signal });
        if (controller.signal.aborted) return;
        active = isActiveQueue(next.queue.state);
        setData(next);
        setError(null);
        setPersistencePending(false);
        if (active) timer = setTimeout(() => { void read(); }, 5000);
      } catch (failure) {
        if (!controller.signal.aborted && isQueuePersistencePending(failure)) {
          active = true;
          setError(null);
          setPersistencePending(true);
          timer = setTimeout(() => { void read(); }, 2000);
        } else if (!controller.signal.aborted) {
          active = false;
          setPersistencePending(false);
          setError(failure);
        }
      } finally {
        pending = false;
      }
    };
    const visibility = () => {
      if (timer) clearTimeout(timer);
      if (!document.hidden && active) void read();
    };
    setError(null);
    setPersistencePending(false);
    void read();
    document.addEventListener('visibilitychange', visibility);
    return () => {
      controller.abort();
      if (timer) clearTimeout(timer);
      document.removeEventListener('visibilitychange', visibility);
    };
  }, [queueId, paused, attempt]);

  return { data, error, persistencePending, refresh: () => setAttempt(value => value + 1) };
}
