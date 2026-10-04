import { useCallback, type ReactNode } from 'react';
import { Link, useParams } from 'react-router-dom';
import type { BusinessMembership } from '../api/types';
import { APIError } from '../api/errors';
import { AccountAccess } from '../components/AccountAccess';
import { BusinessNavigation } from '../components/BusinessNavigation';
import { Button, Card, EmptyState, InlineError, LoadingSkeleton, PageContainer, PageHeader } from '../components/foundation';
import { useAppState } from '../state/AppState';
import { canManage } from './accountForms';
import { operationsError } from './operationsDisplay';
import { useOperations } from './useOperations';
import { getBusiness } from '../api/businesses';
import { getBusinessCounters, getBusinessQueues } from '../api/operations';

export function OperationsFeedback({ loading, error, retry }: { loading: boolean; error: unknown; retry: () => void }) {
  if (loading) return <LoadingSkeleton label="Loading business operations" />;
  if (!error) return null;
  return <Card>{error instanceof APIError && error.status === 403 && <h2>Permission required</h2>}<InlineError>{operationsError(error)}</InlineError>{error instanceof APIError && error.status === 401 ? <Link to="/login">Sign in</Link> : <Button variant="secondary" onClick={retry}>Refresh state</Button>}</Card>;
}
export function BusinessOperationsPage({ title, children }: { title: string; children: (business: BusinessMembership) => ReactNode }) {
  const { businessId = '' } = useParams();
  return <PageContainer className="ql-operations-page"><PageHeader eyebrow="Business tools" title={title} action={<Link to="/business/manage">All businesses</Link>} /><AccountAccess><Membership key={businessId} businessId={businessId}>{children}</Membership></AccountAccess></PageContainer>;
}
function Membership({ businessId, children }: { businessId: string; children: (business: BusinessMembership) => ReactNode }) {
  const { businesses, businessesStatus, refreshBusinesses } = useAppState();
  if (businessesStatus === 'loading') return <LoadingSkeleton label="Checking business access" />;
  if (businessesStatus === 'error') return <Card><InlineError>We couldn’t check business membership.</InlineError><Button onClick={() => refreshBusinesses()}>Try again</Button></Card>;
  const business = businesses.find(item => item.id === businessId && ['owner', 'admin', 'counter'].includes(item.role));
  if (!business) return <EmptyState title="Permission required" description="Business membership is required to view operations." />;
  return <div className="ql-stack"><BusinessNavigation businessId={businessId} manager={canManage(business)} />{children(business)}</div>;
}
export function useBusinessOperations(business: BusinessMembership) {
  const load = useCallback(async (signal: AbortSignal) => {
    const [details, counters, queues] = await Promise.all([getBusiness(business.id, { signal }), getBusinessCounters(business.id, { signal }), getBusinessQueues(business.id, { signal })]);
    return { business: details, counters, queues };
  }, [business.id]);
  return useOperations(load);
}
