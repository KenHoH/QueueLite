import { useCallback } from 'react';
import { Link } from 'react-router-dom';
import { getBusinessCounters } from '../api/operations';
import type { BusinessMembership } from '../api/types';
import { Button, Card, EmptyState, InlineError, LoadingSkeleton, PageContainer } from '../components/foundation';
import { useAppState } from '../state/AppState';
import { canManage } from './accountForms';
import { useOperations } from './useOperations';

export default function StaffCounters() {
  const { businesses, businessesStatus, refreshBusinesses } = useAppState();
  if (businessesStatus === 'loading') return <PageContainer><h1>My service counters</h1><Card><LoadingSkeleton label="Loading assigned counters" /></Card></PageContainer>;
  if (businessesStatus === 'error') return <PageContainer><h1>My service counters</h1><Card><InlineError>We couldn’t load your business access.</InlineError><Button onClick={() => refreshBusinesses()}>Try again</Button></Card></PageContainer>;
  return <CounterAssignments businesses={businesses} />;
}

function CounterAssignments({ businesses }: { businesses: BusinessMembership[] }) {
  const load = useCallback(async (signal: AbortSignal) => Promise.all(businesses.map(async business => ({ business, counters: await getBusinessCounters(business.id, { signal }) }))), [businesses]);
  const resource = useOperations(load, false);
  return <PageContainer className="ql-account-page"><div className="ql-section-heading"><div><h1>My service counters</h1><p className="ql-muted">Open an assigned counter workspace or choose a business to manage.</p></div>{businesses.length > 0 && <Button variant="secondary" onClick={resource.retry}>Refresh counters</Button>}</div>
    {!businesses.length ? <EmptyState title="No business access yet" description="Ask a business owner or admin to add your account, or create your own business." action={<Link to="/business/create">Create a business</Link>} />
      : resource.loading ? <Card><LoadingSkeleton label="Loading assigned counters" lines={5} /></Card>
      : resource.error ? <Card><InlineError>We couldn’t load your counters.</InlineError><Button variant="secondary" onClick={resource.retry}>Try again</Button></Card>
      : <div className="ql-stack">{resource.data?.map(({ business, counters }) => <Card key={business.id}><div className="ql-section-heading"><div><h2>{business.name}</h2><p className="ql-muted">{business.role === 'counter' ? 'Counters assigned to you' : `Business access: ${business.role}`}</p></div><Link to={`/business/${business.id}/counters`}>{canManage(business) ? 'Manage counters' : 'View counters'} →</Link></div>
        {!counters.length ? <p className="ql-muted">{business.role === 'counter' ? 'No counter is assigned to you.' : 'No counters have been created.'}</p>
          : <div className="ql-counter-grid">{counters.map(counter => <Card key={counter.id} className="ql-counter-service-card"><h3>{counter.name}</h3><p className="ql-muted">{counter.currentQueueId ? 'Customer assigned' : 'Ready for the next customer'}</p><Link to={`/counter/${counter.id}`}>Open workspace →</Link></Card>)}</div>}
      </Card>)}</div>}
  </PageContainer>;
}
