import { useCallback } from 'react';
import { Link, useParams } from 'react-router-dom';
import { BusinessNavigation } from '../components/BusinessNavigation';
import { EmptyState, LoadingSkeleton } from '../components/foundation';
import { getBusinessSubscription, getUserSubscription } from '../api/subscriptions';
import type { Subscription } from '../api/types';
import { AccountAccess } from '../components/AccountAccess';
import { Button, Card, PageContainer } from '../components/foundation';
import { useAppState } from '../state/AppState';
import { BusinessSelector } from './ManageBusinesses';
import { canManage } from './accountForms';
import { ResourceFeedback, useAccountResource } from './useAccountResource';

const planName = (value: string) => value.charAt(0).toUpperCase() + value.slice(1);
const date = (value: string) => { const parsed = new Date(value); return Number.isNaN(parsed.getTime()) ? 'Not provided' : parsed.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' }); };
function SubscriptionDates({ subscription }: { subscription: Subscription }) {
  return <dl className="ql-summary"><dt>Status</dt><dd>{planName(subscription.status)}</dd><dt>Started</dt><dd>{date(subscription.startDate)}</dd>{subscription.endDate && <><dt>Ends</dt><dd>{date(subscription.endDate)}</dd></>}</dl>;
}
export function UserSubscriptionSummary({ userId }: { userId: string }) {
  const load = useCallback((signal: AbortSignal) => getUserSubscription(userId, { signal }), [userId]);
  const resource = useAccountResource(load);
  return <Card><h2>Customer subscription</h2><ResourceFeedback {...resource} />{resource.data && <><h3>{planName(resource.data.userPlan.userPlanType)}</h3><p className="ql-muted">{resource.data.userPlan.description}</p><p>Priority slots remaining: {resource.data.userPlan.slots}</p><SubscriptionDates subscription={resource.data.subscription} /></>}</Card>;
}
function BusinessSubscriptionSummary({ businessId }: { businessId: string }) {
  const load = useCallback((signal: AbortSignal) => getBusinessSubscription(businessId, { signal }), [businessId]);
  const resource = useAccountResource(load);
  const info = resource.data;
  return <Card><h2>Business subscription</h2><ResourceFeedback {...resource} />{info && <><h3>{planName(info.businessPlan.businessPlanType)}</h3><p className="ql-muted">{info.businessPlan.description}</p><p>Queue capacity remaining: {info.businessPlan.capacity}</p><dl className="ql-summary"><dt>Analysis</dt><dd>{info.businessPlan.analysis ? 'Included in plan' : 'Not included'}</dd><dt>Insight</dt><dd>{info.businessPlan.insight ? 'Included in plan' : 'Not included'}</dd><dt>Priority support</dt><dd>{info.businessPlan.prioritySupport ? 'Included in plan' : 'Not included'}</dd></dl><SubscriptionDates subscription={info.subscription} /><p className="ql-meta">Plan capabilities shown here reflect your subscription. Analysis and insight pages are coming later.</p></>}</Card>;
}
export default function Plans() {
  const { auth } = useAppState();
  return <PageContainer className="ql-account-page"><h1>Plans &amp; Subscription</h1><AccountAccess>{auth.status === 'authenticated' && <PlansContent key={auth.user.id} userId={auth.user.id} />}</AccountAccess></PageContainer>;
}
function PlansContent({ userId }: { userId: string }) {
  const { businessId } = useParams();
  const { businesses, businessesStatus, refreshBusinesses, selectedBusinessId } = useAppState();
  const managers = businesses.filter(canManage);
  const selected = businessId ? managers.find(item => item.id === businessId) : managers.find(item => item.id === selectedBusinessId) ?? managers[0];
  if (businessId && businessesStatus === 'loading') return <LoadingSkeleton label="Checking business access" />;
  if (businessId && businessesStatus === 'error') return <Card><p>We couldn’t check business access.</p><Button onClick={() => refreshBusinesses()}>Try again</Button></Card>;
  if (businessId && !selected) return <EmptyState title="Permission required" description="Owner or admin access is required to view business plans." />;
  return <div className="ql-stack"><UserSubscriptionSummary userId={userId} />
    <Card><h2>Customer plans</h2><p>Standard · Premium</p><p className="ql-muted">Plan changes and payments are coming later. Upgrades are currently unavailable.</p><Button disabled>Upgrade — coming later</Button></Card>
    {businessId ? <BusinessNavigation businessId={businessId} /> : <BusinessSelector />}{selected && <BusinessSubscriptionSummary key={selected.id} businessId={selected.id} />}
    <Card><h2>Business plans</h2><p>Free · Plus · Pro · Max</p><p className="ql-muted">Choose a business above to see its current subscription. Plan changes and payments are coming later.</p><Button disabled>Change plan — coming later</Button></Card><Link to="/profile">Back to profile</Link>
  </div>;
}
