import { useEffect, useRef, useState, type FormEvent } from 'react';
import { Link, useParams } from 'react-router-dom';
import { getBusiness } from '../api/businesses';
import { APIError } from '../api/errors';
import { joinBusinessQueue } from '../api/queues';
import type { Business, GuestQueueRequest, Queue } from '../api/types';
import { Button, Card, EmptyState, InlineError, Input, LoadingSkeleton, PageContainer, StatusBadge } from '../components/foundation';
import { useAppState } from '../state/AppState';
import { isBusinessId } from './businessDisplay';
import { isAmbiguousJoin, joinErrorMessage, validateGuestJoin } from './queueDisplay';

export default function JoinQueue() {
  const { businessId = '' } = useParams();
  return <JoinBusiness key={businessId} businessId={businessId} />;
}
function JoinBusiness({ businessId }: { businessId: string }) {
  const { auth, setActiveTicket } = useAppState();
  const [business, setBusiness] = useState<Business | null>(null);
  const [load, setLoad] = useState<'loading' | 'ready' | 'not-found' | 'error'>('loading');
  const [attempt, setAttempt] = useState(0);
  const [fields, setFields] = useState<GuestQueueRequest>({ username: '', phoneNumber: '' });
  const [errors, setErrors] = useState<Partial<Record<keyof GuestQueueRequest, string>>>({});
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [uncertain, setUncertain] = useState(false);
  const [ticket, setTicket] = useState<Queue | null>(null);
  const submitting = useRef(false);
  const mutation = useRef<AbortController | null>(null);
  const errorRef = useRef<HTMLDivElement>(null);
  const successRef = useRef<HTMLHeadingElement>(null);
  useEffect(() => () => mutation.current?.abort(), []);
  useEffect(() => { if (ticket) successRef.current?.focus(); }, [ticket]);
  useEffect(() => {
    if (!isBusinessId(businessId)) { setLoad('not-found'); return; }
    const controller = new AbortController();
    setLoad('loading');
    void getBusiness(businessId, { signal: controller.signal }).then(value => {
      if (!controller.signal.aborted) { setBusiness(value); setLoad('ready'); }
    }).catch(failure => {
      if (!controller.signal.aborted) setLoad(failure instanceof APIError && failure.status === 404 ? 'not-found' : 'error');
    });
    return () => controller.abort();
  }, [businessId, attempt]);
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (submitting.current || uncertain || ticket || !business?.operational || !['guest', 'authenticated'].includes(auth.status)) return;
    const validation = auth.status === 'guest' ? validateGuestJoin(fields) : {};
    setErrors(validation); setError('');
    const first = Object.keys(validation)[0];
    if (first) { (event.currentTarget.elements.namedItem(first) as HTMLInputElement)?.focus(); return; }
    submitting.current = true; setBusy(true);
    const controller = new AbortController(); mutation.current = controller;
    try {
      const queue = await joinBusinessQueue(businessId, auth.status === 'authenticated' ? undefined : { username: fields.username.trim(), phoneNumber: fields.phoneNumber.trim() }, { signal: controller.signal });
      if (!controller.signal.aborted) {
        setTicket(queue); setActiveTicket({ queue, joinedAs: auth.status === 'authenticated' ? 'user' : 'guest' });
      }
    } catch (failure) {
      if (!controller.signal.aborted) {
        setError(joinErrorMessage(failure)); setUncertain(isAmbiguousJoin(failure));
        requestAnimationFrame(() => errorRef.current?.focus());
      }
    } finally { if (!controller.signal.aborted) { submitting.current = false; setBusy(false); } }
  }
  const bind = (name: keyof GuestQueueRequest) => ({ name, value: fields[name], disabled: busy, error: errors[name], onChange: (event: React.ChangeEvent<HTMLInputElement>) => setFields(previous => ({ ...previous, [name]: event.target.value })) });
  return <PageContainer className="ql-ticket-page"><Link className="ql-back-link" to={`/business/${businessId}`}>Back to business</Link>
    {load === 'loading' ? <Card><LoadingSkeleton label="Loading business" /></Card>
      : load === 'not-found' ? <EmptyState title="Business not found." description="Choose a business to join its queue." action={<Link to="/#businesses">Browse businesses</Link>} />
      : load === 'error' ? <Card><InlineError>We couldn’t load this business.</InlineError><Button onClick={() => setAttempt(value => value + 1)}>Try again</Button></Card>
      : ticket ? <Card className="ql-ticket-card"><p className="ql-eyebrow">YOUR QUEUE</p><h1 tabIndex={-1} ref={successRef}>Your place is saved.</h1><p>{business?.name}</p><p className="ql-queue-number">{ticket.name}</p><StatusBadge state="waiting" /><p className="ql-muted">Keep this ticket handy for your visit.{auth.status === 'guest' ? ' Reopen it in this browser so your guest credentials remain available.' : ''}</p><Button asChild><Link to={`/queue/${ticket.id}`}>View my queue</Link></Button></Card>
      : <Card><p className="ql-eyebrow">QUEUE ACCESS · {business?.name}</p><h1>Join queue</h1><p className="ql-muted">Confirm your identity below to save your place from this page or a business QR code.</p>
        {!business?.operational ? <p className="ql-muted">This business is currently unavailable. Joining is paused.</p>
          : auth.status === 'loading' ? <LoadingSkeleton label="Checking your account" />
          : auth.status === 'error' ? <InlineError>Please retry the session check above before joining.</InlineError>
          : <form className="ql-stack" onSubmit={submit} noValidate>
            {auth.status === 'authenticated' ? <div className="ql-identity"><p className="ql-meta">Joining with your account</p><p><strong>{auth.user.username}</strong><br /><span>{auth.user.phonenumber}</span></p></div>
              : <><Input label="Name" autoComplete="name" {...bind('username')} /><Input label="Phone number" type="tel" autoComplete="tel" placeholder="0812 3456 7890" hint="An Indonesian mobile number, starting with 08 or +62." {...bind('phoneNumber')} /><p className="ql-meta">Keep this ticket in the same browser to view and manage your queue.</p></>}
            <div tabIndex={-1} ref={errorRef}>{error && <InlineError>{error}</InlineError>}</div>
            <Button type="submit" loading={busy} disabled={uncertain}>Join queue</Button>
            {(error || uncertain) && auth.status === 'authenticated' && <Link to="/my-queues">Check My Queues</Link>}
            {uncertain && auth.status === 'guest' && <p className="ql-meta">Do not submit again yet. Keep this browser open and ask the business to confirm whether your ticket was accepted.</p>}
          </form>}
      </Card>}
  </PageContainer>;
}
