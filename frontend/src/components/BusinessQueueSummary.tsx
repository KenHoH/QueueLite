import { useEffect, useState } from 'react';
import { getQueuesByBusiness } from '../api/queues';
import type { Queue } from '../api/types';
import { LoadingSkeleton } from './foundation';

const activeStates = new Set(['waiting', 'called', 'processing']);

function activeQueues(queues: Queue[]) {
  return queues.filter(queue => activeStates.has(queue.state));
}

function currentQueue(queues: Queue[]) {
  return queues.find(queue => queue.state === 'called')
    ?? queues.find(queue => queue.state === 'processing')
    ?? queues.find(queue => queue.state === 'waiting')
    ?? null;
}

export function BusinessQueueSummary({ businessId, compact = false }: { businessId: string; compact?: boolean }) {
  const [state, setState] = useState<{ status: 'loading' | 'ready' | 'error'; queues: Queue[] }>({ status: 'loading', queues: [] });

  useEffect(() => {
    const controller = new AbortController();
    setState({ status: 'loading', queues: [] });
    void getQueuesByBusiness(businessId, { signal: controller.signal }).then(queues => {
      if (!controller.signal.aborted) setState({ status: 'ready', queues });
    }).catch(() => {
      if (!controller.signal.aborted) setState({ status: 'error', queues: [] });
    });
    return () => controller.abort();
  }, [businessId]);

  if (state.status === 'loading') return <div className="ql-queue-summary" aria-label="Queue summary"><LoadingSkeleton label="Loading queue summary" lines={compact ? 1 : 2} /></div>;
  if (state.status === 'error') return <p className="ql-meta ql-queue-summary-error">Queue information unavailable.</p>;

  const active = activeQueues(state.queues);
  const current = currentQueue(active);
  return <dl className={`ql-queue-summary ${compact ? 'ql-queue-summary-compact' : ''}`} aria-label="Queue summary">
    <div><dt>Current queue</dt><dd>{current?.name ?? 'No active queue'}</dd></div>
    <div><dt>Total queue</dt><dd>{active.length} {active.length === 1 ? 'person' : 'people'}</dd></div>
  </dl>;
}
