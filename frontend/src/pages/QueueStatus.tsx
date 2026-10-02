import { useEffect, useRef, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { getBusiness } from '../api/businesses';
import { getCounter } from '../api/counters';
import { updateQueueState } from '../api/queues';
import { Button, Card, InlineError, LoadingSkeleton, PageContainer, StatusBadge } from '../components/foundation';
import { useAppState } from '../state/AppState';
import { cancellationErrorMessage, isActiveQueue, queueHeadings, ticketErrorMessage } from './queueDisplay';
import { useQueueTicket } from './useQueueTicket';

export default function QueueStatus() {
  const { queueId = '' } = useParams();
  return <Ticket key={queueId} queueId={queueId} />;
}
function Ticket({ queueId }: { queueId: string }) {
  const { auth } = useAppState();
  const [busy, setBusy] = useState(false);
  const [cancelled, setCancelled] = useState(false);
  const [confirm, setConfirm] = useState(false);
  const [cancelError, setCancelError] = useState('');
  const { queue, error, refresh } = useQueueTicket(queueId, busy || cancelled);
  const [business, setBusiness] = useState('');
  const [counter, setCounter] = useState('');
  const mutation = useRef<AbortController | null>(null);
  const submitting = useRef(false);
  const confirmRef = useRef<HTMLDivElement>(null);
  useEffect(() => () => mutation.current?.abort(), []);
  useEffect(() => { if (confirm) confirmRef.current?.focus(); }, [confirm]);
  useEffect(() => {
    if (!queue?.businessId) return;
    const controller = new AbortController();
    void getBusiness(queue.businessId, { signal: controller.signal }).then(value => { if (!controller.signal.aborted) setBusiness(value.name); }).catch(() => {});
    return () => controller.abort();
  }, [queue?.businessId]);
  useEffect(() => {
    setCounter('');
    if (!queue?.calledByCounterId) return;
    const controller = new AbortController();
    void getCounter(queue.calledByCounterId, { signal: controller.signal }).then(value => { if (!controller.signal.aborted && value.businessId === queue.businessId) setCounter(value.name); }).catch(() => {});
    return () => controller.abort();
  }, [queue?.calledByCounterId, queue?.businessId]);
  async function cancel() {
    if (submitting.current || !confirm || !queue || queue.state !== 'waiting') return;
    submitting.current = true; setBusy(true); setCancelError('');
    const controller = new AbortController(); mutation.current = controller;
    try {
      await updateQueueState(queueId, 'cancelled', { signal: controller.signal });
      if (!controller.signal.aborted) { setCancelled(true); setConfirm(false); }
    } catch (failure) { if (!controller.signal.aborted) setCancelError(cancellationErrorMessage(failure)); }
    finally { if (!controller.signal.aborted) { submitting.current = false; setBusy(false); } }
  }
  const state = cancelled ? 'cancelled' : queue?.state;
  return <PageContainer className="ql-ticket-page"><Link className="ql-back-link" to={auth.status === 'authenticated' ? '/my-queues' : '/#businesses'}>{auth.status === 'authenticated' ? 'Back to My Queues' : 'Back to businesses'}</Link>
    <Card className={`ql-ticket-card ${state === 'called' ? 'ql-ticket-called' : ''}`}>
      {!queue && !error ? <LoadingSkeleton label="Loading your queue" lines={6} /> : <>
        {queue && state && <><p className="ql-eyebrow">YOUR QUEUE</p><div role="status" aria-live="polite"><h1>{queueHeadings[state]}</h1><StatusBadge state={state} /></div><p>{business || 'Business details unavailable'}</p><p className="ql-queue-number">{queue.name}</p>
          {queue.calledByCounterId && isActiveQueue(state) && <p className="ql-counter-name">{counter ? `Counter: ${counter}` : 'A counter has been assigned. Ask the team where to go.'}</p>}
          {state === 'waiting' && <p className="ql-muted">Keep your ticket handy. This page updates while your queue is active.</p>}
          {state === 'called' && <p>Head to your assigned counter. The team is ready for you.</p>}
          {state === 'skipped' && <p className="ql-muted">Please ask the team about the next step.</p>}
          <Link to={`/business/${queue.businessId}`}>View business details</Link>
        </>}
        {error && <><InlineError>{ticketErrorMessage(error)}</InlineError><Button variant="secondary" onClick={refresh}>Refresh ticket</Button></>}
        {cancelError && <InlineError>{cancelError}</InlineError>}
        {queue && state === 'waiting' && !error && <div className="ql-ticket-actions">{confirm ? <div ref={confirmRef} tabIndex={-1} role="group" aria-label="Confirm leaving queue"><h2>Leave this queue?</h2><p>Your place will be released. You can join again later.</p><div className="ql-actions"><Button variant="secondary" disabled={busy} onClick={() => { setConfirm(false); setCancelError(''); }}>Keep my place</Button><Button variant="danger" loading={busy} onClick={() => { void cancel(); }}>Confirm leave</Button></div></div> : <Button variant="secondary" onClick={() => setConfirm(true)}>Leave queue</Button>}</div>}
      </>}
    </Card>
  </PageContainer>;
}
