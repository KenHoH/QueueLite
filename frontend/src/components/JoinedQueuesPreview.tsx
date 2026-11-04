import { useEffect, useState } from 'react';
import { ArrowRight, Ticket } from 'lucide-react';
import { Link } from 'react-router-dom';
import { getBusiness } from '../api/businesses';
import { APIError } from '../api/errors';
import { getMyQueues, getGuestQueues } from '../api/queues';
import type { Queue } from '../api/types';
import { Button, Card, EmptyState, InlineError, LoadingSkeleton, StatusBadge } from './foundation';
import { useAppState } from '../state/AppState';
import { isActiveQueue } from '../pages/queueDisplay';

type PreviewState = 'loading' | 'ready' | 'error' | 'unauthorized';
export function JoinedQueuesPreview() {
  const { auth, refreshUser } = useAppState();
  const [queues, setQueues] = useState<Queue[]>([]);
  const [names, setNames] = useState<Record<string, string>>({});
  const [status, setStatus] = useState<PreviewState>('loading');
  const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    if (auth.status !== 'authenticated' && auth.status !== 'guest') return;
    const controller = new AbortController();
    setStatus('loading');
    const fetchQueues = auth.status === 'authenticated' ? getMyQueues : getGuestQueues;
    void fetchQueues({ signal: controller.signal }).then(async list => {
      const active = list.filter(queue => isActiveQueue(queue.state));
      const entries = await Promise.all([...new Set(active.map(queue => queue.businessId))].map(async id => {
        try { const business = await getBusiness(id, { signal: controller.signal }); return [id, business.name] as const; }
        catch { return [id, 'Business details unavailable'] as const; }
      }));
      if (!controller.signal.aborted) { setQueues(active); setNames(Object.fromEntries(entries)); setStatus('ready'); }
    }).catch(error => {
      if (controller.signal.aborted) return;
      const unauthorized = error instanceof APIError && error.status === 401;
      setStatus(unauthorized ? 'unauthorized' : 'error');
      if (unauthorized && auth.status === 'authenticated') void refreshUser(true).catch(() => {});
    });
    return () => controller.abort();
  }, [auth.status, auth.status === 'authenticated' ? auth.user.id : '', attempt, refreshUser]);
  const guest = auth.status === 'guest';
  const displayed = queues;
  return <section className="ql-joined-queues" aria-labelledby="joined-queues-heading">
    <div className="ql-section-heading"><div><h2 id="joined-queues-heading">Joined queues {status === 'ready' && <span className="ql-count">{displayed.length}</span>}</h2><p className="ql-muted">A little freedom while you wait.</p></div>{status === 'ready' && <Button variant="secondary" onClick={() => setAttempt(value => value + 1)}>Refresh queues</Button>}</div>
    {auth.status === 'loading' || status === 'loading' && auth.status !== 'error' ? <Card><LoadingSkeleton label="Loading joined queues" lines={2} /></Card>
      : auth.status === 'error' ? <Card><InlineError>Please retry the session check above to view your queues.</InlineError></Card>
      : status === 'unauthorized' && !guest ? <Card><InlineError>Your session has ended. Please sign in to view your queues.</InlineError><Link to="/login?returnTo=%2F">Sign in</Link></Card>
      : status === 'error' ? <Card><InlineError>We couldn’t load your queues.</InlineError><Button onClick={() => setAttempt(value => value + 1)}>Try again</Button></Card>
      : !displayed.length ? <EmptyState compact title={guest ? 'No joined queues yet.' : 'You don’t have any active queues.'} description="Find a business and save your place. Recently joined queues may take a moment to appear here." />
      : <div className="ql-active-queue-list">{displayed.map(queue => <Card asChild className="ql-active-queue" key={queue.id}><Link to={`/queue/${queue.id}`}><span className="ql-business-icon" aria-hidden="true"><Ticket size={24} /></span><div className="ql-active-business"><StatusBadge state={queue.state} /><h3>{names[queue.businessId] ?? 'Your queue'}</h3><p className="ql-meta">Your place, on your time.</p></div><div className="ql-active-number"><span className="ql-eyebrow">Your number</span><strong>{queue.name}</strong></div><span className="ql-active-link">View queue <ArrowRight size={17} aria-hidden="true" /></span></Link></Card>)}</div>}
  </section>;
}
