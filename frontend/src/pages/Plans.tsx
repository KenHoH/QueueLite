import { useCallback, useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { BusinessNavigation } from '../components/BusinessNavigation';
import { EmptyState, InlineError, LoadingSkeleton } from '../components/foundation';
import { getBusinessSubscription, getUserSubscription, updateUserSubscriptionType } from '../api/subscriptions';
import type { Subscription, UserSubscriptionInfo } from '../api/types';
import type { PlanComparison } from './subscriptionPlans';
import { businessPlanComparison, canShowBusinessPlans, userPlanComparison } from './subscriptionPlans';
import { AccountAccess } from '../components/AccountAccess';
import { Button, Card, PageContainer, PageHeader } from '../components/foundation';
import { useAppState } from '../state/AppState';
import { BusinessSelector } from './ManageBusinesses';
import { accountError, canManage } from './accountForms';
import { ResourceFeedback, useAccountResource } from './useAccountResource';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../components/ui/table';

const planName = (value: string) => value.charAt(0).toUpperCase() + value.slice(1);
const date = (value: string) => { const parsed = new Date(value); return Number.isNaN(parsed.getTime()) ? 'Not provided' : parsed.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' }); };
function SubscriptionDates({ subscription }: { subscription: Subscription }) {
  return <dl className="ql-summary"><dt>Status</dt><dd>{planName(subscription.status)}</dd><dt>Started</dt><dd>{date(subscription.startDate)}</dd>{subscription.endDate && <><dt>Ends</dt><dd>{date(subscription.endDate)}</dd></>}</dl>;
}
function PlanValue({ value }: { value: string }) {
  if (value === '✓') return <><span aria-hidden="true">✓</span><span className="ql-sr-only">Included</span></>;
  if (value === '—') return <><span aria-hidden="true">—</span><span className="ql-sr-only">Not included</span></>;
  return value;
}
function PlanComparisonTable({ id, title, comparison }: { id: string; title: string; comparison: PlanComparison }) {
  return <section className="ql-stack" aria-labelledby={`${id}-heading`}><div><p className="ql-eyebrow">PLAN COMPARISON</p><h2 id={`${id}-heading`}>{title}</h2></div><div className="ql-plan-table-wrap" role="region" aria-label={`${title} comparison`} tabIndex={0}><Table className="ql-plan-table"><caption className="ql-sr-only">{title} feature comparison</caption><TableHeader><TableRow><TableHead scope="col">Feature</TableHead>{comparison.plans.map(plan => <TableHead scope="col" key={plan}>{plan}</TableHead>)}</TableRow></TableHeader><TableBody>{comparison.features.map(feature => <TableRow key={feature.name}><TableHead scope="row">{feature.name}</TableHead>{feature.values.map((value, index) => <TableCell key={comparison.plans[index]}><PlanValue value={value} /></TableCell>)}</TableRow>)}</TableBody></Table></div>
    {/* Phones get one stacked card per plan instead of a sideways-scrolling table (CSS swaps them at 640px). */}
    <div className="ql-plan-cards">{comparison.plans.map((plan, index) => <Card key={plan} className="ql-plan-card"><h3>{plan}</h3><p className="ql-plan-price">{comparison.features.find(feature => feature.name === 'Price')?.values[index]}</p><dl>{comparison.features.filter(feature => feature.name !== 'Price').map(feature => <div key={feature.name}><dt>{feature.name}</dt><dd><PlanValue value={feature.values[index]} /></dd></div>)}</dl></Card>)}</div></section>;
}
export function UserSubscriptionSummary({ userId, onLoaded }: { userId: string; onLoaded?: (info: UserSubscriptionInfo) => void }) {
  const load = useCallback((signal: AbortSignal) => getUserSubscription(userId, { signal }), [userId]);
  const resource = useAccountResource(load);

  useEffect(() => {
    if (resource.data && onLoaded) {
      onLoaded(resource.data);
    }
  }, [resource.data, onLoaded]);

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
  return <PageContainer className="ql-account-page ql-plans-page"><PageHeader title="Plans & Subscription" description="Review current allowances and compare the available plan catalog." /><AccountAccess>{auth.status === 'authenticated' && <PlansContent key={auth.user.id} userId={auth.user.id} />}</AccountAccess></PageContainer>;
}
function PlansContent({ userId }: { userId: string }) {
  const { businessId } = useParams();
  const { businesses, businessesStatus, refreshBusinesses, selectedBusinessId } = useAppState();
  const [currentPlan, setCurrentPlan] = useState<string>('standard');
  const [version, setVersion] = useState(0);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  const managers = businesses.filter(canManage);
  const showBusinessPlans = canShowBusinessPlans(businesses);
  const selected = businessId ? managers.find(item => item.id === businessId) : managers.find(item => item.id === selectedBusinessId) ?? managers[0];
  if (businessId && businessesStatus === 'loading') return <LoadingSkeleton label="Checking business access" />;
  if (businessId && businessesStatus === 'error') return <Card><p>We couldn’t check business access.</p><Button onClick={() => refreshBusinesses()}>Try again</Button></Card>;
  if (businessId && !selected) return <EmptyState title="Permission required" description="Owner or admin access is required to view business plans." />;

  const handleUpgrade = async () => {
    setBusy(true);
    setError('');
    setSuccess('');
    try {
      await updateUserSubscriptionType(userId, 'premium');
      setSuccess('Your subscription has been successfully upgraded to Premium with priority slots.');
      setVersion(v => v + 1);
    } catch (failure) {
      setError(accountError(failure));
    } finally {
      setBusy(false);
    }
  };

  const isPremium = currentPlan === 'premium';

  return <div className="ql-stack"><UserSubscriptionSummary key={`${userId}-${version}`} userId={userId} onLoaded={(info) => setCurrentPlan(info.userPlan.userPlanType)} />
    <PlanComparisonTable id="user-plans" title="Customer plans" comparison={userPlanComparison} />
    <Card>
      <p className="ql-muted">
        Upgrade to the Premium customer plan to receive 4 monthly priority slots for express queueing.
      </p>
      {error && <InlineError>{error}</InlineError>}
      {success && <p role="status" className="ql-success-note">{success}</p>}
      <Button
        disabled={busy || isPremium}
        onClick={() => void handleUpgrade()}
      >
        {busy ? 'Upgrading…' : isPremium ? 'Current plan: Premium' : 'Upgrade to Premium'}
      </Button>
    </Card>
    {businessesStatus === 'loading' ? <Card><LoadingSkeleton label="Checking managed businesses" /></Card>
      : businessesStatus === 'error' ? <Card><p>We couldn’t check your managed businesses.</p><Button onClick={() => refreshBusinesses()}>Try again</Button></Card>
      : showBusinessPlans && <>{businessId ? <BusinessNavigation businessId={businessId} /> : <BusinessSelector />}{selected && <BusinessSubscriptionSummary key={selected.id} businessId={selected.id} />}
        <PlanComparisonTable id="business-plans" title="Business plans" comparison={businessPlanComparison} />
        <Card><p className="ql-muted">Business plan comparisons are informational. Listed integrations, analytics, insights, billing, and plan changes may not yet be available.</p><Button disabled>Change plan — coming later</Button></Card></>}
    <Link to="/profile">Back to profile</Link>
  </div>;
}
