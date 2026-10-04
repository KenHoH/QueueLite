import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { getBusiness } from '../api/businesses';
import { APIError } from '../api/errors';
import { getMyQueues } from '../api/queues';
import type { Queue } from '../api/types';
import { Button, Card, EmptyState, InlineError, LoadingSkeleton, PageContainer, StatusBadge } from '../components/foundation';
import { useAppState } from '../state/AppState';

export default function MyQueues() {
  const { auth } = useAppState();
  return <PageContainer><h1>My Queues</h1>{auth.status === 'authenticated' ? <QueueList key={auth.user.id} />
    : auth.status === 'loading' ? <Card><LoadingSkeleton label="Checking your account" /></Card>
    : auth.status === 'error' ? <Card><InlineError>Please retry the session check above to view your queues.</InlineError></Card>
    : <EmptyState title="Sign in to view your queues." description="My Queues is available for customers with an account. Guest tickets can be opened in the browser where you joined." action={<Button asChild><Link to="/login?returnTo=%2Fmy-queues">Sign in</Link></Button>} />}</PageContainer>;
}
function QueueList() {
  const { refreshUser } = useAppState();
  const [queues, setQueues] = useState<Queue[]>([]);
  const [names, setNames] = useState<Record<string, string>>({});
  const [status, setStatus] = useState<'loading' | 'ready' | 'error' | 'unauthorized'>('loading');
  const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    setStatus('loading');
    void getMyQueues({ signal: controller.signal }).then(async list => {
      if (controller.signal.aborted) return;
      const entries = await Promise.all([...new Set(list.map(queue => queue.businessId))].map(async id => {
        try { const business = await getBusiness(id, { signal: controller.signal }); return [id, business.name] as const; }
        catch { return [id, 'Business details unavailable'] as const; }
      }));
      if (!controller.signal.aborted) { setQueues(list); setNames(Object.fromEntries(entries)); setStatus('ready'); }
    }).catch(error => {
      if (controller.signal.aborted) return;
      const unauthorized = error instanceof APIError && error.status === 401;
      setStatus(unauthorized ? 'unauthorized' : 'error');
      if (unauthorized) void refreshUser(true).catch(() => {});
    });
    return () => controller.abort();
  }, [attempt, refreshUser]);
  if (status === 'loading') return <Card><LoadingSkeleton label="Loading your queues" lines={5} /></Card>;
  if (status === 'unauthorized') return <Card><InlineError>Your session has ended. Please sign in to view your queues.</InlineError><Link to="/login?returnTo=%2Fmy-queues">Sign in</Link></Card>;
  if (status === 'error') return <Card><InlineError>We couldn’t load your queues.</InlineError><Button onClick={() => setAttempt(value => value + 1)}>Try again</Button></Card>;
  return <><div className="ql-section-heading"><p className="ql-muted">Your active visits</p><Button variant="secondary" onClick={() => setAttempt(value => value + 1)}>Refresh queues</Button></div>
    {queues.length === 0 ? <EmptyState title="You don’t have any active queues." description="Find a business and save your place. Recently joined queues may take a moment to appear here." action={<Link to="/#businesses">Browse businesses</Link>} />
      : <div className="ql-business-grid">{queues.map(queue => <Card asChild className="ql-queue-card" key={queue.id}><Link to={`/queue/${queue.id}`}><h2>{names[queue.businessId]}</h2><p className="ql-queue-number">{queue.name}</p><StatusBadge state={queue.state} /><span className="ql-meta">View queue →</span></Link></Card>)}</div>}
  </>;
}
