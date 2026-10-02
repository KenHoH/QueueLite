import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import type { Business, BusinessMembership, Counter, Queue, QueueState } from '../api/types';
import { getBusinessSubscription } from '../api/subscriptions';
import { Button, Card, EmptyState, StatusBadge } from '../components/foundation';
import { BusinessOperationsPage, OperationsFeedback, useBusinessOperations } from './BusinessOperations';
import { canManage } from './accountForms';
import { averageServiceSeconds, counterStatus, currentQueue, formatDuration } from './operationsDisplay';
import { useOperations } from './useOperations';

const activeStates: QueueState[] = ['waiting', 'called', 'processing'];
type QueueFilter = 'current' | QueueState;
const filters: Array<{ value: QueueFilter; label: string }> = [
  { value: 'current', label: 'All current' },
  { value: 'waiting', label: 'Waiting' },
  { value: 'called', label: 'Called' },
  { value: 'processing', label: 'In service' },
  { value: 'done', label: 'Completed' },
  { value: 'cancelled', label: 'Cancelled' },
  { value: 'skipped', label: 'Skipped' },
];
const pageSizes = [10, 25, 50] as const;

export function PriorityBadge({ queue }: { queue: Queue }) {
  return queue.priority ? <span className="ql-badge ql-priority">Priority</span> : null;
}

function ServicesDashboardHeader({ business, refresh }: { business: Business; refresh: () => void }) {
  return <Card className="ql-services-header">
    <div>
      <p className="ql-eyebrow">Services overview</p>
      <h2>{business.name}</h2>
      <p className="ql-muted">{business.operational ? 'Operational' : 'Not operational'} · Updates every 10 seconds while this tab is visible.</p>
    </div>
    <Button variant="secondary" onClick={refresh}>Refresh</Button>
  </Card>;
}

function QueueFilterTabs({ value, queues, onChange }: { value: QueueFilter; queues: Queue[]; onChange: (value: QueueFilter) => void }) {
  const count = (filter: QueueFilter) => queues.filter(queue => filter === 'current' ? activeStates.includes(queue.state) : queue.state === filter).length;
  return <div className="ql-queue-tabs" role="tablist" aria-label="Filter customer queues">
    {filters.map(filter => <button key={filter.value} type="button" role="tab" aria-selected={value === filter.value} onClick={() => onChange(filter.value)}>
      {filter.label} <span>{count(filter.value)}</span>
    </button>)}
  </div>;
}

function CurrentQueueList({ queues, counters }: { queues: Queue[]; counters: Counter[] }) {
  if (!queues.length) return <EmptyState title="No queues in this view" description="No waiting queues. Choose another status or wait for a customer to join." />;
  return <div className="ql-queue-table-wrap">
    <table className="ql-queue-table">
      <thead><tr><th>Customer</th><th>Status</th><th>Counter</th><th>Queue type</th></tr></thead>
      <tbody>{queues.map(queue => {
        const counter = counters.find(item => item.currentQueueId === queue.id || item.id === queue.calledByCounterId);
        return <tr key={queue.id} className={queue.priority ? 'ql-priority-row' : undefined}>
          <th scope="row" data-label="Customer"><span role="region" aria-label={`${queue.state} queues`}>{queue.name}</span></th>
          <td data-label="Status"><StatusBadge state={queue.state} /></td>
          <td data-label="Counter">{counter?.name ?? '—'}</td>
          <td data-label="Queue type"><PriorityBadge queue={queue} />{!queue.priority && <span className="ql-muted">Standard</span>}</td>
        </tr>;
      })}</tbody>
    </table>
  </div>;
}

function QueueBoard({ queues, counters }: { queues: Queue[]; counters: Counter[] }) {
  const [filter, setFilter] = useState<QueueFilter>('current');
  const [pageSize, setPageSize] = useState<(typeof pageSizes)[number]>(10);
  const [page, setPage] = useState(1);
  const filtered = useMemo(() => queues.filter(queue => filter === 'current' ? activeStates.includes(queue.state) : queue.state === filter), [filter, queues]);
  const pageCount = Math.max(1, Math.ceil(filtered.length / pageSize));
  const currentPage = Math.min(page, pageCount);
  const displayed = filtered.slice((currentPage - 1) * pageSize, currentPage * pageSize);
  useEffect(() => { setPage(value => Math.min(value, pageCount)); }, [pageCount]);
  function changeFilter(next: QueueFilter) { setFilter(next); setPage(1); }
  function changePageSize(next: number) { setPageSize(next as (typeof pageSizes)[number]); setPage(1); }
  return <section className="ql-stack" aria-labelledby="current-queue-heading">
    <div className="ql-section-heading"><div><h2 id="current-queue-heading">Customer queues</h2><p className="ql-muted">Priority customers are highlighted.</p></div></div>
    <QueueFilterTabs value={filter} queues={queues} onChange={changeFilter} />
    <div className="ql-queue-toolbar">
      <label>Rows per page <select className="ql-input" value={pageSize} onChange={event => changePageSize(Number(event.target.value))}>{pageSizes.map(size => <option key={size}>{size}</option>)}</select></label>
      <span className="ql-meta">{filtered.length ? `${(currentPage - 1) * pageSize + 1}–${Math.min(currentPage * pageSize, filtered.length)} of ${filtered.length}` : '0 customers'}</span>
    </div>
    <CurrentQueueList queues={displayed} counters={counters} />
    {pageCount > 1 && <nav className="ql-pagination" aria-label="Customer queue pages">
      <Button variant="secondary" disabled={currentPage === 1} onClick={() => setPage(value => Math.max(1, value - 1))}>Previous</Button>
      <span>Page {currentPage} of {pageCount}</span>
      <Button variant="secondary" disabled={currentPage === pageCount} onClick={() => setPage(value => Math.min(pageCount, value + 1))}>Next</Button>
    </nav>}
  </section>;
}

export function CounterOverview({ counters, queues }: { counters: Counter[]; queues: Queue[] }) {
  if (!counters.length) return <EmptyState title="No counters available" description="Owners and admins can create counters. Staff see only assigned counters." />;
  return <div className="ql-counter-grid">{counters.map(counter => {
    const queue = currentQueue(counter, queues);
    return <Card key={counter.id} className="ql-counter-service-card">
      <div className="ql-counter-card-heading"><h3>{counter.name}</h3><span className={`ql-counter-state ${queue ? 'ql-counter-state-busy' : ''}`}>{counterStatus(counter, queues)}</span></div>
      <p className="ql-meta">Customer being handled</p>
      {queue ? <p className="ql-counter-customer"><strong>{queue.name}</strong> <PriorityBadge queue={queue} /></p> : <p className="ql-muted">No active customer</p>}
      <Link to={`/counter/${counter.id}`}>Open workspace →</Link>
    </Card>;
  })}</div>;
}

function QueueMetricsSummary({ queues }: { queues: Queue[] }) {
  const count = (state: QueueState) => queues.filter(queue => queue.state === state).length;
  const average = averageServiceSeconds(queues);
  return <section aria-labelledby="queue-metrics-heading"><h2 id="queue-metrics-heading">Queue totals</h2><div className="ql-metrics-grid">
    <Card><span className="ql-meta">Skipped</span><strong>{count('skipped')}</strong><span>customers</span></Card>
    <Card><span className="ql-meta">Cancelled</span><strong>{count('cancelled')}</strong><span>customers</span></Card>
    <Card><span className="ql-meta">Finished</span><strong>{count('done')}</strong><span>customers</span></Card>
    <Card className={average === null ? 'ql-metric-unavailable' : ''}><span className="ql-meta">Average service time</span><strong>{average === null ? '—' : formatDuration(average)}</strong><span>{average === null ? 'No completed timing data' : 'processing to completion'}</span></Card>
  </div></section>;
}

export default function Dashboard() {
  return <BusinessOperationsPage title="Business dashboard">{business => <DashboardContent business={business} />}</BusinessOperationsPage>;
}

function Capacity({ businessId }: { businessId: string }) {
  const load = useCallback((signal: AbortSignal) => getBusinessSubscription(businessId, { signal }), [businessId]);
  const resource = useOperations(load);
  return <Card><h2>Daily queue capacity</h2><OperationsFeedback {...resource} />{resource.data && <p>Queue capacity remaining: {resource.data.businessPlan.capacity}</p>}</Card>;
}

function DashboardContent({ business }: { business: BusinessMembership }) {
  const resource = useBusinessOperations(business);
  return <><OperationsFeedback {...resource} />{resource.data && <>
    <ServicesDashboardHeader business={resource.data.business} refresh={resource.retry} />
    <QueueBoard queues={resource.data.queues} counters={resource.data.counters} />
    <section aria-label="Counters"><div className="ql-section-heading"><div><h2>Service counters · {resource.data.counters.length}</h2><p className="ql-muted">Current customer at every available counter.</p></div></div><CounterOverview counters={resource.data.counters} queues={resource.data.queues} /></section>
    <QueueMetricsSummary queues={resource.data.queues} />
    {canManage(business) && <Capacity businessId={business.id} />}
  </>}</>;
}
