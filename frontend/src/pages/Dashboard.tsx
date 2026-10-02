import { useCallback } from 'react';
import { Link } from 'react-router-dom';
import type { BusinessMembership, Counter, Queue } from '../api/types';
import { getBusinessSubscription } from '../api/subscriptions';
import { Button, Card, EmptyState, StatusBadge } from '../components/foundation';
import { BusinessOperationsPage, OperationsFeedback, useBusinessOperations } from './BusinessOperations';
import { canManage } from './accountForms';
import { counterStatus, currentQueue, groupQueues } from './operationsDisplay';
import { useOperations } from './useOperations';

export function PriorityBadge({ queue }: { queue: Queue }) { return queue.priority ? <span className="ql-badge ql-priority">Priority</span> : null; }
export function CounterOverview({ counters, queues }: { counters: Counter[]; queues: Queue[] }) {
  if (!counters.length) return <EmptyState title="No counters available" description="Owners and admins can create counters. Staff see only assigned counters." />;
  return <div className="ql-counter-grid">{counters.map(counter => { const queue = currentQueue(counter, queues); return <Card key={counter.id}><h3>{counter.name}</h3><p>{counterStatus(counter, queues)}</p>{queue && <p>Current queue: <strong>{queue.name}</strong> <PriorityBadge queue={queue} /></p>}<Link to={`/counter/${counter.id}`}>Open workspace →</Link></Card>; })}</div>;
}
export default function Dashboard() { return <BusinessOperationsPage title="Business dashboard">{business => <DashboardContent business={business} />}</BusinessOperationsPage>; }
function Capacity({ businessId }: { businessId: string }) {
  const load = useCallback((signal: AbortSignal) => getBusinessSubscription(businessId, { signal }), [businessId]);
  const resource = useOperations(load);
  return <Card><h2>Daily queue capacity</h2><OperationsFeedback {...resource} />{resource.data && <p>Queue capacity remaining: {resource.data.businessPlan.capacity}</p>}</Card>;
}
function DashboardContent({ business }: { business: BusinessMembership }) {
  const resource = useBusinessOperations(business);
  const groups = resource.data ? groupQueues(resource.data.queues) : null;
  return <><OperationsFeedback {...resource} />{resource.data && groups && <>
    <Card><div className="ql-section-heading"><div><h2>{resource.data.business.name}</h2><p>{resource.data.business.operational ? 'Operational' : 'Not operational'}</p></div><Button variant="secondary" onClick={resource.retry}>Refresh</Button></div><p className="ql-meta">Operations refresh every 10 seconds while this tab is visible.</p></Card>
    <div className="ql-operation-grid">{(['waiting', 'called', 'processing'] as const).map(state => <section aria-label={`${state} queues`} key={state}><Card><h2>{state === 'processing' ? 'Processing' : state === 'called' ? 'Called' : 'Waiting'} <span className="ql-muted">{groups[state].length}</span></h2>{groups[state].length ? <ul className="ql-operational-list">{groups[state].map(queue => <li key={queue.id}><strong>{queue.name}</strong> <StatusBadge state={queue.state} /> <PriorityBadge queue={queue} /></li>)}</ul> : <p className="ql-muted">No {state} queues.</p>}</Card></section>)}</div>
    <section aria-label="Counters"><h2>Counters · {resource.data.counters.length}</h2><CounterOverview counters={resource.data.counters} queues={resource.data.queues} /></section>
    {canManage(business) && <Capacity businessId={business.id} />}
  </>}</>;
}
