import { ArrowLeft, Clock3, Info, Ticket as TicketIcon, Volume2, VolumeX } from 'lucide-react';
import { useEffect, useRef, useState, type PointerEvent } from 'react';
import { Link, useParams } from 'react-router-dom';
import { updateQueueState } from '../api/queues';
import type { QueueState } from '../api/types';
import { CounterBoard } from '../components/CounterBoard';
import { Button, Card, InlineError, LoadingSkeleton, PageContainer, StatusBadge } from '../components/foundation';
import { LiveValue, QueueProgressTrack } from '../components/LiveMotion';
import { useToast } from '../components/Toast';
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from '../components/ui/alert-dialog';
import { useAppState } from '../state/AppState';
import { useSound } from '../state/SoundContext';
import { cancellationErrorMessage, isActiveQueue, queueHeadings, ticketErrorMessage } from './queueDisplay';
import { useCustomerQueueStatus } from './useCustomerQueueStatus';

export default function QueueStatus() {
  const { queueId = '' } = useParams();
  return <Ticket key={queueId} queueId={queueId} />;
}

function Ticket({ queueId }: { queueId: string }) {
  const { auth } = useAppState();
  const toast = useToast();
  const sound = useSound();
  const [busy, setBusy] = useState(false);
  const [cancelled, setCancelled] = useState(false);
  const [confirm, setConfirm] = useState(false);
  const [cancelError, setCancelError] = useState('');
  const { data, error, persistencePending, streamStatus, refresh } = useCustomerQueueStatus(queueId, busy || cancelled);
  const mutation = useRef<AbortController | null>(null);
  const submitting = useRef(false);
  const previousState = useRef<QueueState | undefined>(undefined);

  useEffect(() => () => mutation.current?.abort(), []);

  const queue = data?.queue;
  const state = cancelled ? 'cancelled' : queue?.state;
  const myCounter = queue ? data?.counters.find(counter => counter.currentQueueId === queue.id) : undefined;
  const back = auth.status === 'authenticated' ? '/my-queues' : '/#businesses';

  // Trigger optional chime & toast only on live state transition into "called".
  useEffect(() => {
    const prev = previousState.current;
    previousState.current = state;
    if (prev && prev !== 'called' && state === 'called') {
      sound.turnChime();
      toast.success(myCounter ? `It’s your turn! Head to ${myCounter.name}.` : 'It’s your turn! Head to your counter.');
    }
  }, [state, sound, myCounter, toast]);

  // Update browser tab title when called so background tabs surface the turn immediately.
  useEffect(() => {
    if (state === 'called') {
      document.title = myCounter ? `Your turn (${myCounter.name}) · QueueLite` : 'It’s your turn! · QueueLite';
      return () => { document.title = 'Queue ticket · QueueLite'; };
    }
    if (state) document.title = 'Queue ticket · QueueLite';
  }, [state, myCounter]);

  function toggleSound() {
    const next = sound.toggle('chime');
    toast.info(next ? 'Sound alert on. You’ll hear a chime when called.' : 'Sound alert muted.');
  }

  function onTicketPointerMove(event: PointerEvent<HTMLDivElement>) {
    if (event.pointerType !== 'mouse') return;
    const rect = event.currentTarget.getBoundingClientRect();
    const x = (event.clientX - rect.left) / rect.width;
    const y = (event.clientY - rect.top) / rect.height;
    event.currentTarget.style.setProperty('--ql-rx', `${((0.5 - y) * 6).toFixed(2)}deg`);
    event.currentTarget.style.setProperty('--ql-ry', `${((x - 0.5) * 6).toFixed(2)}deg`);
    event.currentTarget.style.setProperty('--ql-tx', `${Math.round(x * 100)}%`);
    event.currentTarget.style.setProperty('--ql-ty', `${Math.round(y * 100)}%`);
  }

  function onTicketPointerLeave(event: PointerEvent<HTMLDivElement>) {
    event.currentTarget.style.setProperty('--ql-rx', '0deg');
    event.currentTarget.style.setProperty('--ql-ry', '0deg');
  }

  async function cancel() {
    if (submitting.current || !confirm || !data || data.queue.state !== 'waiting') return;
    submitting.current = true;
    setBusy(true);
    setCancelError('');
    const controller = new AbortController();
    mutation.current = controller;
    try {
      await updateQueueState(queueId, 'cancelled', { signal: controller.signal });
      if (!controller.signal.aborted) {
        setCancelled(true);
        setConfirm(false);
        sound.pop();
        toast.info('You left the queue.');
      }
    } catch (failure) {
      if (!controller.signal.aborted) {
        sound.softError();
        setCancelError(cancellationErrorMessage(failure));
      }
    } finally {
      if (!controller.signal.aborted) { submitting.current = false; setBusy(false); }
    }
  }

  const showLiveControls = state && isActiveQueue(state);
  const positionHeadline = data?.customerPosition
    ? data.customerPosition.ahead === 0
      ? 'You are next.'
      : `${data.customerPosition.ahead} ${data.customerPosition.ahead === 1 ? 'queue' : 'queues'} ahead of you.`
    : '';

  return <PageContainer className="ql-ticket-page ql-status-page">
    <Link className="ql-back-link" to={back}><ArrowLeft size={16} aria-hidden="true" />{auth.status === 'authenticated' ? 'Back to My Queues' : 'Back to businesses'}</Link>
    {persistencePending && !data ? <Card><p className="ql-eyebrow">TICKET CONFIRMATION</p><h1>Saving your ticket</h1><p>Your place was accepted and is still being confirmed. Keep this page open—we’ll retry automatically.</p><LoadingSkeleton label="Confirming your ticket" lines={2} /><Button variant="secondary" onClick={refresh}>Check now</Button></Card>
      : !data && !error ? <Card><LoadingSkeleton label="Loading your queue" lines={8} /></Card> : <>
      {persistencePending && data && <Card role="status"><p>Your latest queue update is still being confirmed. Showing the last known ticket state.</p></Card>}
      {data && queue && state && <>
        <div className="ql-status-heading">
          <div>
            <p className="ql-eyebrow">A little time for yourself</p>
            <h1>{queueHeadings[state]}</h1>
            <p className="ql-muted">{data.business.name}</p>
          </div>
          <div className="ql-status-controls">
            {showLiveControls && streamStatus !== 'closed' && (
              <span className={`ql-live-pill ql-live-pill-${streamStatus}`} role="status">
                <span className="ql-live-pill-dot" aria-hidden="true" />
                {streamStatus === 'reconnecting' ? 'Reconnecting…' : 'Live updates'}
              </span>
            )}
            {(state === 'waiting' || state === 'called') && (
              <button
                type="button"
                className={`ql-sound-toggle ${sound.enabled ? 'is-on' : ''}`}
                aria-pressed={sound.enabled}
                onClick={toggleSound}
              >
                {sound.enabled ? <Volume2 size={16} aria-hidden="true" /> : <VolumeX size={16} aria-hidden="true" />}
                <span>{sound.enabled ? 'Sound on' : 'Sound alert'}</span>
              </button>
            )}
            <div role="status" aria-live="polite"><StatusBadge state={state} /></div>
          </div>
        </div>
        <div className="ql-status-layout"><Card
          className={`ql-ticket-card ql-ticket-stub ql-ticket-pass ${state === 'called' ? 'ql-ticket-called' : ''}`}
          onPointerMove={onTicketPointerMove}
          onPointerLeave={onTicketPointerLeave}
        >
          <span className={`ql-ticket-icon ${state === 'called' ? 'ql-sonar-rings' : ''}`} aria-hidden="true"><TicketIcon size={25} /></span>
          <p className="ql-eyebrow">YOUR QUEUE</p>
          <p className="ql-queue-number">{queue.name}</p>
          {state === 'waiting' && data.customerPosition && <div className="ql-position-summary">
            <p><strong><LiveValue value={positionHeadline} /></strong></p>
            <p className="ql-muted">Position {data.customerPosition.position} of {data.totalWaiting} waiting</p>
            <QueueProgressTrack ahead={data.customerPosition.ahead} totalWaiting={data.totalWaiting} />
            {data.estimatedWaitMinutes !== undefined && <p className="ql-wait-estimate"><Clock3 size={19} aria-hidden="true" /><span>Estimated waiting time<strong><LiveValue value={`${data.estimatedWaitMinutes} minutes`} /></strong></span></p>}
          </div>}
          {state === 'called' && <p className="ql-ticket-callout"><strong>{myCounter ? `Please go to ${myCounter.name}.` : 'Head to your assigned counter.'}</strong> The team is ready for you.</p>}
          {state === 'processing' && <p>You are currently being served.</p>}
          {state === 'skipped' && <p className="ql-muted">Please ask the team about the next step.</p>}
          {cancelError && !confirm && <InlineError>{cancelError}</InlineError>}
          {state === 'waiting' && !error && <div className="ql-ticket-actions"><AlertDialog open={confirm} onOpenChange={open => { setConfirm(open); if (!open) setCancelError(''); }}><AlertDialogTrigger asChild><Button variant="secondary">Leave queue</Button></AlertDialogTrigger><AlertDialogContent><AlertDialogHeader><AlertDialogTitle>Leave this queue?</AlertDialogTitle><AlertDialogDescription>Your place will be released. You can join again later.</AlertDialogDescription></AlertDialogHeader>{cancelError && <InlineError>{cancelError}</InlineError>}<AlertDialogFooter><AlertDialogCancel disabled={busy}>Keep my place</AlertDialogCancel><AlertDialogAction disabled={busy} onClick={event => { event.preventDefault(); void cancel(); }}>{busy ? 'Leaving…' : 'Confirm leave'}</AlertDialogAction></AlertDialogFooter></AlertDialogContent></AlertDialog></div>}
        </Card>

        <div className="ql-status-board"><section aria-labelledby="counter-board-heading">
          <div className="ql-section-heading"><div><h2 id="counter-board-heading">Service counters</h2><p className="ql-muted">{data.activeCounters} active of {data.totalCounters} total counters</p></div></div>
          <CounterBoard counters={data.counters} highlightCounterId={state === 'called' || state === 'processing' ? myCounter?.id : undefined} />
        </section>

        <Card className="ql-next-queue-card">
          <p className="ql-eyebrow">NEXT CUSTOMER</p>
          <h2><LiveValue value={data.nextQueue?.queueName ?? 'No waiting customer'} /></h2>
          <p className="ql-muted">{data.totalWaiting} {data.totalWaiting === 1 ? 'customer' : 'customers'} waiting</p>
        </Card><p className="ql-soft-note"><Info size={18} aria-hidden="true" />Queue position and estimated waiting time may change based on service conditions. Priority customers may be called sooner.</p><div className="ql-wait-copy"><h3>Make the wait your own.</h3><p className="ql-muted">Take a short break and keep this ticket handy. Head back when your turn is close.</p><Link to={`/business/${queue.businessId}`}>View business details →</Link></div></div></div>
      </>}
      {error && <Card><InlineError>{ticketErrorMessage(error)}</InlineError><Button variant="secondary" onClick={refresh}>Refresh ticket</Button></Card>}
    </>}
  </PageContainer>;
}
