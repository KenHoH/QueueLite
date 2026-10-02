import type { CustomerCounterStatus } from '../api/types';
import { Card, EmptyState } from './foundation';

export function CounterBoard({ counters }: { counters: CustomerCounterStatus[] }) {
  if (!counters.length) return <EmptyState title="No counters available" description="The business has not opened a service counter yet." />;
  return <div className="ql-customer-counter-grid">
    {counters.map(counter => <Card className="ql-customer-counter" key={counter.id}>
      <h3>{counter.name}</h3>
      <p className="ql-queue-number">{counter.currentQueueName ?? 'Idle'}</p>
      <span className="ql-meta">{counter.state === 'processing' ? 'In service' : counter.state === 'called' ? 'Called' : 'Available'}</span>
    </Card>)}
  </div>;
}
