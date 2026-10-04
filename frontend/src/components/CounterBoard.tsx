import type { CustomerCounterStatus } from '../api/types';
import { Card, EmptyState } from './foundation';
import { Badge } from './ui/badge';

export function CounterBoard({ counters }: { counters: CustomerCounterStatus[] }) {
  if (!counters.length) return <EmptyState title="No counters available" description="The business has not opened a service counter yet." />;
  return <div className="ql-customer-counter-grid">
    {counters.map(counter => <Card className="ql-customer-counter" key={counter.id}>
      <h3>{counter.name}</h3>
      <p className="ql-queue-number">{counter.currentQueueName ?? 'Idle'}</p>
      <Badge variant={counter.state === 'idle' ? 'secondary' : 'default'}>{counter.state === 'processing' ? 'In service' : counter.state === 'called' ? 'Called' : 'Available'}</Badge>
    </Card>)}
  </div>;
}
