import { CheckCheck, CircleX, Clock3, SkipForward } from 'lucide-react';
import { greeting } from './businessDisplay';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import type { Business, BusinessMembership, Counter, Queue, QueueState } from '../api/types';
import { getBusinessSubscription } from '../api/subscriptions';
import { Button, Card, EmptyState, StatusBadge } from '../components/foundation';
import { Badge } from '../components/ui/badge';
import { Label } from '../components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../components/ui/select';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../components/ui/table';
import { Tabs, TabsList, TabsTrigger } from '../components/ui/tabs';
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
  return queue.priority ? <Badge className="ql-priority">Priority</Badge> : null;
}

function ServicesDashboardHeader({ business, refresh }: { business: Business; refresh: () => void }) {
  return <div className="ql-services-header">
    <div>
      <h2>{business.name}</h2>
      <p className="ql-muted">Here’s how things are moving at your business.</p>
    </div>
    <Button variant="secondary" onClick={refresh}>Refresh</Button>
  </div>;
}

function QueueFilterTabs({ value, queues, onChange }: { value: QueueFilter; queues: Queue[]; onChange: (value: QueueFilter) => void }) {
  const count = (filter: QueueFilter) => queues.filter(queue => filter === 'current' ? activeStates.includes(queue.state) : queue.state === filter).length;
  return <Tabs value={value} onValueChange={next => onChange(next as QueueFilter)}><TabsList className="ql-queue-tabs" aria-label="Filter customer queues">
    {filters.map(filter => <TabsTrigger key={filter.value} value={filter.value}>{filter.label} <span>{count(filter.value)}</span></TabsTrigger>)}
  </TabsList></Tabs>;
}

function CurrentQueueList({ queues, counters }: { queues: Queue[]; counters: Counter[] }) {
  if (!queues.length) return <EmptyState title="No queues in this view" description="No waiting queues. Choose another status or wait for a customer to join." />;
  return <div className="ql-queue-table-wrap">
    <Table className="ql-queue-table">
      <TableHeader><TableRow><TableHead>Queue number</TableHead><TableHead>Status</TableHead><TableHead>Counter</TableHead><TableHead>Queue type</TableHead></TableRow></TableHeader>
      <TableBody>{queues.map(queue => {
        const counter = counters.find(item => item.currentQueueId === queue.id || item.id === queue.calledByCounterId);
        return <TableRow key={queue.id} className={queue.priority ? 'ql-priority-row' : undefined}>
          <TableHead scope="row" data-label="Queue number">{queue.name}</TableHead>
          <TableCell data-label="Status"><StatusBadge state={queue.state} /></TableCell>
          <TableCell data-label="Counter">{counter?.name ?? '—'}</TableCell>
          <TableCell data-label="Queue type"><PriorityBadge queue={queue} />{!queue.priority && <span className="ql-muted">Standard</span>}</TableCell>
        </TableRow>;
      })}</TableBody>
    </Table>
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
    <div className="ql-section-heading"><div><h2 id="current-queue-heading">Queue overview</h2><p className="ql-muted">Priority customers are highlighted.</p></div></div>
    <QueueFilterTabs value={filter} queues={queues} onChange={changeFilter} />
    <div className="ql-queue-toolbar">
      <div className="flex items-center gap-2"><Label htmlFor="queue-page-size">Rows per page</Label><Select value={String(pageSize)} onValueChange={value => changePageSize(Number(value))}><SelectTrigger id="queue-page-size" className="w-20"><SelectValue /></SelectTrigger><SelectContent>{pageSizes.map(size => <SelectItem key={size} value={String(size)}>{size}</SelectItem>)}</SelectContent></Select></div>
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
      <div className="ql-counter-card-heading"><h3>{counter.name}</h3><Badge variant={queue ? 'secondary' : 'default'}>{counterStatus(counter, queues)}</Badge></div>

      {queue ? <p className="ql-counter-customer"><strong>{queue.name}</strong> <PriorityBadge queue={queue} /></p> : <p className="ql-counter-customer">Available</p>}
      <p className="ql-meta">{queue ? 'Currently serving' : 'Ready for the next customer'}</p><Link to={`/counter/${counter.id}`}>Open workspace →</Link>
    </Card>;
  })}</div>;
}

function QueueMetricsSummary({ queues }: { queues: Queue[] }) {
  const count = (state: QueueState) => queues.filter(queue => queue.state === state).length;
  const average = averageServiceSeconds(queues);
  return <section aria-labelledby="queue-metrics-heading"><h2 id="queue-metrics-heading" className="ql-sr-only">Queue totals</h2><div className="ql-metrics-grid">
    <Card><span className="ql-metric-label">Customers served <CheckCheck size={18} aria-hidden="true" /></span><strong>{count('done')}</strong><span>Recorded completed visits</span></Card>
    <Card><span className="ql-metric-label">Skipped <SkipForward size={18} aria-hidden="true" /></span><strong>{count('skipped')}</strong><span>Customers not present</span></Card>
    <Card><span className="ql-metric-label">Cancelled <CircleX size={18} aria-hidden="true" /></span><strong>{count('cancelled')}</strong><span>Recorded cancelled visits</span></Card>
    <Card><span className="ql-metric-label">Average service time <Clock3 size={18} aria-hidden="true" /></span><strong>{average === null ? '—' : formatDuration(average)}</strong><span>{average === null ? 'No completed timing data' : 'From recorded completed visits'}</span></Card>
  </div></section>;
}

export default function Dashboard() {
  return <BusinessOperationsPage title={greeting()} eyebrow="Your business, at a glance">{business => <DashboardContent business={business} />}</BusinessOperationsPage>;
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
    <QueueMetricsSummary queues={resource.data.queues} />
    <section aria-label="Counters"><div className="ql-section-heading"><div><h2>Service counters · {resource.data.counters.length}</h2><p className="ql-muted">Current customer at every available counter.</p></div></div><CounterOverview counters={resource.data.counters} queues={resource.data.queues} /></section>
    <QueueBoard queues={resource.data.queues} counters={resource.data.counters} />
    {canManage(business) && <Capacity businessId={business.id} />}
  </>}</>;
}
