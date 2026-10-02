import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { getBusiness } from '../api/businesses';
import { APIError } from '../api/errors';
import { getMyQueues } from '../api/queues';
import type { Queue } from '../api/types';
import { Button, Card, EmptyState, InlineError, LoadingSkeleton, StatusBadge } from './foundation';
import { useAppState } from '../state/AppState';

type PreviewState = 'idle' | 'loading' | 'ready' | 'error' | 'unauthorized';

export function JoinedQueuesPreview() {
  const { auth, activeTicket, refreshUser } = useAppState();
  const [queues, setQueues] = useState<Queue[]>([]);
  const [names, setNames] = useState<Record<string, string>>({});
  const [status, setStatus] = useState<PreviewState>('idle');
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    if (auth.status !== 'authenticated') { setStatus('idle'); setQueues([]); setNames({}); return; }
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
  }, [auth.status, auth.status === 'authenticated' ? auth.user.id : '', attempt, refreshUser]);

  if (auth.status === 'loading') return <section className="ql-joined-queues" aria-labelledby="joined-queues-heading"><Card><LoadingSkeleton label="Checking your queues" lines={2} /></Card></section>;

  if (auth.status !== 'authenticated') {
    return <section className="ql-joined-queues" aria-labelledby="joined-queues-heading">
      <div className="ql-section-heading"><div><h2 id="joined-queues-heading">Joined queues</h2><p className="ql-muted">Queues you have joined appear here.</p></div></div>
      {activeTicket ? <div className="ql-business-grid"><QueueCard queue={activeTicket.queue} businessName="Your guest queue" /></div>
        : <EmptyState title="No joined queues yet." description="Sign in to view your queues across devices, or join a business as a guest from this browser." action={<Link className="ql-button ql-button-primary" to="/login?returnTo=%2F">Sign in</Link>} />}
    </section>;
  }

  return <section className="ql-joined-queues" aria-labelledby="joined-queues-heading">
    <div className="ql-section-heading"><div><h2 id="joined-queues-heading">Joined queues</h2><p className="ql-muted">Businesses or services where you have an active visit.</p></div>
      {status === 'ready' && <Button variant="secondary" onClick={() => setAttempt(value => value + 1)}>Refresh queues</Button>}
    </div>
    {status === 'loading' ? <Card><LoadingSkeleton label="Loading joined queues" lines={4} /></Card>
      : status === 'unauthorized' ? <Card><InlineError>Your session has ended. Please sign in to view your queues.</InlineError><Link to="/login?returnTo=%2F">Sign in</Link></Card>
      : status === 'error' ? <Card><InlineError>We couldn’t load your queues.</InlineError><Button onClick={() => setAttempt(value => value + 1)}>Try again</Button></Card>
      : queues.length === 0 ? <EmptyState title="You don’t have any active queues." description="Find a business and save your place. Recently joined queues may take a moment to appear here." />
      : <div className="ql-business-grid">{queues.map(queue => <QueueCard key={queue.id} queue={queue} businessName={names[queue.businessId] ?? 'Business details unavailable'} />)}</div>}
  </section>;
}

function QueueCard({ queue, businessName }: { queue: Queue; businessName: string }) {
  return <Link className="ql-card ql-queue-card" to={`/queue/${queue.id}`}><h3>{businessName}</h3><p className="ql-queue-number">{queue.name}</p><StatusBadge state={queue.state} /><span className="ql-meta">View queue →</span></Link>;
}
