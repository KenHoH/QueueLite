import { useEffect, useRef, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { updateQueueState } from '../api/queues';
import { CounterBoard } from '../components/CounterBoard';
import { Button, Card, InlineError, LoadingSkeleton, PageContainer, StatusBadge } from '../components/foundation';
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from '../components/ui/alert-dialog';
import { useAppState } from '../state/AppState';
import { cancellationErrorMessage, queueHeadings, ticketErrorMessage } from './queueDisplay';
import { useCustomerQueueStatus } from './useCustomerQueueStatus';

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
  const { data, error, persistencePending, refresh } = useCustomerQueueStatus(queueId, busy || cancelled);
  const mutation = useRef<AbortController | null>(null);
  const submitting = useRef(false);

  useEffect(() => () => mutation.current?.abort(), []);

  async function cancel() {
    if (submitting.current || !confirm || !data || data.queue.state !== 'waiting') return;
    submitting.current = true;
    setBusy(true);
    setCancelError('');
    const controller = new AbortController();
    mutation.current = controller;
    try {
      await updateQueueState(queueId, 'cancelled', { signal: controller.signal });
      if (!controller.signal.aborted) { setCancelled(true); setConfirm(false); }
    } catch (failure) {
      if (!controller.signal.aborted) setCancelError(cancellationErrorMessage(failure));
    } finally {
      if (!controller.signal.aborted) { submitting.current = false; setBusy(false); }
    }
  }

  const queue = data?.queue;
  const state = cancelled ? 'cancelled' : queue?.state;
  const back = auth.status === 'authenticated' ? '/my-queues' : '/#businesses';
  return <PageContainer className="ql-ticket-page">
    <Link className="ql-back-link" to={back}>{auth.status === 'authenticated' ? 'Back to My Queues' : 'Back to businesses'}</Link>
    {persistencePending && !data ? <Card><p className="ql-eyebrow">TICKET CONFIRMATION</p><h1>Saving your ticket</h1><p>Your place was accepted and is still being confirmed. Keep this page open—we’ll retry automatically.</p><LoadingSkeleton label="Confirming your ticket" lines={2} /><Button variant="secondary" onClick={refresh}>Check now</Button></Card>
      : !data && !error ? <Card><LoadingSkeleton label="Loading your queue" lines={8} /></Card> : <>
      {persistencePending && data && <Card role="status"><p>Your latest queue update is still being confirmed. Showing the last known ticket state.</p></Card>}
      {data && queue && state && <>
        <Card className={`ql-ticket-card ${state === 'called' ? 'ql-ticket-called' : ''}`}>
          <p className="ql-eyebrow">YOUR QUEUE</p>
          <div role="status" aria-live="polite"><h1>{queueHeadings[state]}</h1><StatusBadge state={state} /></div>
          <p>{data.business.name}</p>
          <p className="ql-queue-number">{queue.name}</p>
          {state === 'waiting' && data.customerPosition && <div className="ql-position-summary">
            <p><strong>{data.customerPosition.ahead === 0 ? 'You are next.' : `${data.customerPosition.ahead} ${data.customerPosition.ahead === 1 ? 'queue' : 'queues'} ahead of you.`}</strong></p>
            <p className="ql-muted">Position {data.customerPosition.position} of {data.totalWaiting} waiting</p>
            {data.estimatedWaitMinutes !== undefined && <p>Estimated wait: <strong>{data.estimatedWaitMinutes} minutes</strong></p>}
          </div>}
          {state === 'called' && <p>Head to your assigned counter. The team is ready for you.</p>}
          {state === 'processing' && <p>You are currently being served.</p>}
          {state === 'skipped' && <p className="ql-muted">Please ask the team about the next step.</p>}
          <Link to={`/business/${queue.businessId}`}>View business details</Link>
          {cancelError && <InlineError>{cancelError}</InlineError>}
          {state === 'waiting' && !error && <div className="ql-ticket-actions"><AlertDialog open={confirm} onOpenChange={open => { setConfirm(open); if (!open) setCancelError(''); }}><AlertDialogTrigger asChild><Button variant="secondary">Leave queue</Button></AlertDialogTrigger><AlertDialogContent><AlertDialogHeader><AlertDialogTitle>Leave this queue?</AlertDialogTitle><AlertDialogDescription>Your place will be released. You can join again later.</AlertDialogDescription></AlertDialogHeader><AlertDialogFooter><AlertDialogCancel disabled={busy}>Keep my place</AlertDialogCancel><AlertDialogAction disabled={busy} onClick={event => { event.preventDefault(); void cancel(); }}>{busy ? 'Leaving…' : 'Confirm leave'}</AlertDialogAction></AlertDialogFooter></AlertDialogContent></AlertDialog></div>}
        </Card>

        <section aria-labelledby="counter-board-heading">
          <div className="ql-section-heading"><div><h2 id="counter-board-heading">Service counters</h2><p className="ql-muted">{data.activeCounters} active of {data.totalCounters} total counters</p></div></div>
          <CounterBoard counters={data.counters} />
        </section>

        <Card className="ql-next-queue-card">
          <p className="ql-eyebrow">NEXT CUSTOMER</p>
          <h2>{data.nextQueue?.queueName ?? 'No waiting customer'}</h2>
          <p className="ql-muted">{data.totalWaiting} {data.totalWaiting === 1 ? 'customer' : 'customers'} waiting</p>
        </Card>
      </>}
      {error && <Card><InlineError>{ticketErrorMessage(error)}</InlineError><Button variant="secondary" onClick={refresh}>Refresh ticket</Button></Card>}
    </>}
  </PageContainer>;
}
