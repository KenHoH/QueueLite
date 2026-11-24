import { ArrowLeft, ArrowRight, Clock3, MapPin, ShieldCheck, Store } from 'lucide-react';
import { motion } from 'motion/react';
import { useEffect, useRef, useState, type FormEvent } from 'react';
import { Link, useParams } from 'react-router-dom';
import { getBusiness } from '../api/businesses';
import { APIError } from '../api/errors';
import { joinBusinessQueue, getGuestQueues } from '../api/queues';
import type { Business, GuestQueueRequest, Queue } from '../api/types';
import { Button, Card, EmptyState, InlineError, Input, LoadingSkeleton, PageContainer, StatusBadge } from '../components/foundation';
import { AnimatedCheckIcon } from '../components/LiveMotion';
import { useToast } from '../components/Toast';
import { useAppState } from '../state/AppState';
import { businessHours, isBusinessId } from './businessDisplay';
import { BusinessAvailability } from '../components/BusinessCard';
import { isAmbiguousJoin, joinErrorMessage, validateGuestJoin } from './queueDisplay';

function isValidIndonesianMobile(value: string) {
  const normalized = value.trim().replace(/[\s-]/g, '').replace(/^\+62/, '62').replace(/^08/, '628');
  return /^628[0-9]{8,11}$/.test(normalized);
}

export default function JoinQueue() {
  const { businessId = '' } = useParams();
  return <JoinBusiness key={businessId} businessId={businessId} />;
}
function JoinBusiness({ businessId }: { businessId: string }) {
  const { auth, setActiveTicket } = useAppState();
  const toast = useToast();
  const [business, setBusiness] = useState<Business | null>(null);
  const [load, setLoad] = useState<'loading' | 'ready' | 'not-found' | 'error'>('loading');
  const [attempt, setAttempt] = useState(0);
  const [fields, setFields] = useState<GuestQueueRequest>({ username: '', phoneNumber: '' });
  const [errors, setErrors] = useState<Partial<Record<keyof GuestQueueRequest, string>>>({});
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [uncertain, setUncertain] = useState(false);
  const [ticket, setTicket] = useState<Queue | null>(null);
  const [shaking, setShaking] = useState(false);
  const shakeTimer = useRef<number | null>(null);
  const submitting = useRef(false);
  const mutation = useRef<AbortController | null>(null);
  const errorRef = useRef<HTMLDivElement>(null);
  const successRef = useRef<HTMLHeadingElement>(null);
  useEffect(() => () => {
    mutation.current?.abort();
    if (shakeTimer.current !== null) window.clearTimeout(shakeTimer.current);
  }, []);
  function triggerShake() {
    if (shakeTimer.current !== null) window.clearTimeout(shakeTimer.current);
    setShaking(false);
    requestAnimationFrame(() => {
      setShaking(true);
      shakeTimer.current = window.setTimeout(() => setShaking(false), 320);
    });
  }
  useEffect(() => { if (ticket) { window.scrollTo({ top: 0, behavior: 'auto' }); successRef.current?.focus({ preventScroll: true }); } }, [ticket]);
  useEffect(() => {
    if (!isBusinessId(businessId)) { setLoad('not-found'); return; }
    const controller = new AbortController();
    setLoad('loading');
    void getBusiness(businessId, { signal: controller.signal }).then(async value => {
      if (controller.signal.aborted) return;
      setBusiness(value);
      setLoad('ready');
      if (auth.status === 'guest') {
        try {
          const guestQueues = await getGuestQueues({ signal: controller.signal });
          const existing = guestQueues.find(q => q.businessId === businessId && !['done', 'cancelled', 'skipped'].includes(q.state));
          if (existing && !controller.signal.aborted) {
            setTicket(existing);
            setActiveTicket({ queue: existing, joinedAs: 'guest' });
          }
        } catch {
          // best-effort active ticket detection
        }
      }
    }).catch(failure => {
      if (!controller.signal.aborted) setLoad(failure instanceof APIError && failure.status === 404 ? 'not-found' : 'error');
    });
    return () => controller.abort();
  }, [businessId, attempt, auth.status, setActiveTicket]);
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (submitting.current || uncertain || ticket || !business?.operational || !['guest', 'authenticated'].includes(auth.status)) return;
    const validation = auth.status === 'guest' ? validateGuestJoin(fields) : {};
    setErrors(validation); setError('');
    const first = Object.keys(validation)[0];
    if (first) {
      triggerShake();
      (event.currentTarget.elements.namedItem(first) as HTMLInputElement)?.focus();
      return;
    }
    submitting.current = true; setBusy(true);
    const controller = new AbortController(); mutation.current = controller;
    try {
      const queue = await joinBusinessQueue(businessId, auth.status === 'authenticated' ? undefined : { username: fields.username.trim(), phoneNumber: fields.phoneNumber.trim() }, { signal: controller.signal });
      if (!controller.signal.aborted) {
        setTicket(queue); setActiveTicket({ queue, joinedAs: auth.status === 'authenticated' ? 'user' : 'guest' });
        toast.success(`Ticket ${queue.name} issued.`);
      }
    } catch (failure) {
      if (!controller.signal.aborted) {
        setError(joinErrorMessage(failure)); setUncertain(isAmbiguousJoin(failure));
        triggerShake();
        requestAnimationFrame(() => errorRef.current?.focus());
      }
    } finally { if (!controller.signal.aborted) { submitting.current = false; setBusy(false); } }
  }
  const bind = (name: keyof GuestQueueRequest) => ({
    name,
    value: fields[name],
    disabled: busy,
    error: errors[name],
    valid: name === 'username' ? fields.username.trim().length > 0 : isValidIndonesianMobile(fields.phoneNumber),
    onChange: (event: React.ChangeEvent<HTMLInputElement>) => {
      const next = event.target.value;
      setFields(previous => ({ ...previous, [name]: next }));
      if (errors[name]) setErrors(previous => ({ ...previous, [name]: undefined }));
    },
  });
  return <PageContainer className={`ql-ticket-page ql-join-page ${ticket ? 'ql-join-success-page' : ''}`}><Link className="ql-back-link" to={`/business/${businessId}`}><ArrowLeft size={16} aria-hidden="true" />Back to business</Link>
    {load === 'loading' ? <Card><LoadingSkeleton label="Loading business" /></Card>
      : load === 'not-found' ? <EmptyState title="Business not found." description="Choose a business to join its queue." action={<Link to="/#businesses">Browse businesses</Link>} />
      : load === 'error' ? <Card><InlineError>We couldn’t load this business.</InlineError><Button onClick={() => setAttempt(value => value + 1)}>Try again</Button></Card>
      : ticket ? <section className="ql-join-success">
        <motion.span className="ql-success-icon" aria-hidden="true" initial={{ scale: 0.86 }} animate={{ scale: 1 }} transition={{ type: 'spring', stiffness: 320, damping: 22 }}><AnimatedCheckIcon size={30} /></motion.span>
        <p className="ql-eyebrow">A little less waiting starts now</p>
        <h1 tabIndex={-1} ref={successRef}>Your place is saved.</h1>
        <p className="ql-muted">Your ticket at {business?.name} is ready.</p>
        <motion.div className="ql-ticket-stub-motion" initial={{ y: 12, scale: 0.98 }} animate={{ y: 0, scale: 1 }} transition={{ duration: 0.34, ease: [0.16, 1, 0.3, 1] }}>
          <Card className="ql-ticket-card ql-ticket-stub">
            <p className="ql-eyebrow">Your queue number</p>
            <motion.p className="ql-queue-number" initial={{ scale: 0.94 }} animate={{ scale: 1 }} transition={{ duration: 0.28, ease: [0.16, 1, 0.3, 1] }}>{ticket.name}</motion.p>
            <StatusBadge state={ticket.state} />
            <p className="ql-meta">Open your ticket for the latest queue position.</p>
          </Card>
        </motion.div>
        <Button asChild><Link to={`/queue/${ticket.id}`}>View my queue <ArrowRight size={17} aria-hidden="true" /></Link></Button>
        {auth.status === 'guest' && <p className="ql-meta">Reopen your ticket in this browser to keep access to your place.</p>}
      </section>
      : <div className="ql-join-content"><div className="ql-join-heading"><p className="ql-eyebrow">Your next stop, without the wait</p><h1>Join queue</h1><p className="ql-muted">A few details, and you’re good to go.</p></div><div className="ql-join-layout"><Card className="ql-join-form"><h2>Your details</h2><p className="ql-muted">So the team knows how to welcome you.</p>
        {!business?.operational ? <p className="ql-muted">This business is currently unavailable. Joining is paused.</p>
          : auth.status === 'loading' ? <LoadingSkeleton label="Checking your account" />
          : auth.status === 'error' ? <InlineError>Please retry the session check above before joining.</InlineError>
          : <form className={`ql-stack ${shaking ? 'ql-shake' : ''}`} onSubmit={submit} noValidate>
            {auth.status === 'authenticated' ? <div className="ql-identity"><p className="ql-meta">Joining with your account</p><p><strong>{auth.user.username}</strong><br /><span>{auth.user.phonenumber}</span></p></div>
              : <><Input label="Name" autoComplete="name" {...bind('username')} /><Input label="Phone number" type="tel" autoComplete="tel" placeholder="0812 3456 7890" hint="An Indonesian mobile number, starting with 08 or +62." {...bind('phoneNumber')} /><p className="ql-meta">Keep this ticket in the same browser to view and manage your queue.</p></>}
            <div tabIndex={-1} ref={errorRef}>{error && <InlineError>{error}</InlineError>}</div>
            <Button type="submit" loading={busy} disabled={uncertain}>Join queue</Button>
            {(error || uncertain) && auth.status === 'authenticated' && <Link to="/my-queues">Check My Queues</Link>}
            {uncertain && auth.status === 'guest' && <p className="ql-meta">Do not submit again yet. Keep this browser open and ask the business to confirm whether your ticket was accepted.</p>}
          </form>}
      </Card><aside className="ql-join-aside"><Card><span className="ql-business-icon" aria-hidden="true"><Store size={26} /></span><h2>{business?.name}</h2><p className="ql-business-location"><MapPin size={16} aria-hidden="true" />{business?.location || 'Location not provided'}</p>{business && <><BusinessAvailability business={business} /><p className="ql-soft-note"><Clock3 size={18} aria-hidden="true" />{businessHours(business)}</p></>}</Card><p className="ql-meta ql-security-note"><ShieldCheck size={18} aria-hidden="true" />Your place is linked to your account or this browser.</p></aside></div></div>}
  </PageContainer>;
}
