import { useEffect, useState } from 'react';
import { getQueue, getQueueState } from '../api/queues';
import type { Queue } from '../api/types';
import { isActiveQueue } from './queueDisplay';
import { isBusinessId } from './businessDisplay';
import { APIError } from '../api/errors';

/** Serial reads: no overlapping polls, and all timers/requests belong to this mount. */
export function useQueueTicket(queueId: string, paused: boolean) {
  const [queue, setQueue] = useState<Queue | null>(null);
  const [error, setError] = useState<unknown>(null);
  const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    if (paused) return;
    if (!isBusinessId(queueId)) { setError(new APIError(400, 'INVALID_QUEUE_ID', 'Invalid ticket link.')); return; }
    const controller = new AbortController();
    let timer: ReturnType<typeof setTimeout> | undefined;
    let pending = false;
    let active = true;
    const read = async () => {
      if (pending || controller.signal.aborted || document.hidden) return;
      pending = true;
      try {
        const detail = await getQueue(queueId, { signal: controller.signal });
        if (controller.signal.aborted) return;
        const state = await getQueueState(queueId, { signal: controller.signal });
        if (controller.signal.aborted) return;
        const next = { ...detail, state: state.state };
        active = isActiveQueue(next.state);
        setQueue(next); setError(null);
        if (active) timer = setTimeout(() => { void read(); }, 5000);
      } catch (failure) {
        if (!controller.signal.aborted) { active = false; setError(failure); }
      } finally { pending = false; }
    };
    const visibility = () => {
      if (timer) clearTimeout(timer);
      if (!document.hidden && active) void read();
    };
    setError(null);
    void read();
    document.addEventListener('visibilitychange', visibility);
    return () => { controller.abort(); if (timer) clearTimeout(timer); document.removeEventListener('visibilitychange', visibility); };
  }, [queueId, paused, attempt]);
  return { queue, error, refresh: () => setAttempt(value => value + 1) };
}
