import type { CustomerCounterStatus } from '../api/types';
import { Card, EmptyState } from './foundation';
import { LiveValue } from './LiveMotion';
import { Badge } from './ui/badge';

export function CounterBoard({ counters, highlightCounterId }: { counters: CustomerCounterStatus[]; highlightCounterId?: string }) {
  if (!counters.length) return <EmptyState title="No counters available" description="The business has not opened a service counter yet." />;
  return <div className="ql-customer-counter-grid">
    {counters.map(counter => {
      const mine = highlightCounterId !== undefined && counter.id === highlightCounterId;
      return <Card className={`ql-customer-counter ${mine ? 'ql-customer-counter-mine' : ''}`} key={counter.id}>
        <div className="ql-customer-counter-top">
          <h3>{counter.name}</h3>
          {mine && <span className="ql-counter-your-tag">Your counter</span>}
        </div>
        <p className="ql-queue-number"><LiveValue value={counter.currentQueueName ?? 'Available'} /></p>
        <Badge variant={counter.state === 'idle' ? 'secondary' : 'default'}>{counter.state === 'processing' ? 'In service' : counter.state === 'called' ? 'Called' : 'Available'}</Badge>
      </Card>;
    })}
  </div>;
}
