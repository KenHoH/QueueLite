import { useCallback, useRef, useState, type FormEvent } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { createBusiness, getBusiness, updateBusiness } from '../api/businesses';
import { APIError } from '../api/errors';
import { AccountAccess } from '../components/AccountAccess';
import { Button, Card, InlineError, Input, LoadingSkeleton, PageContainer } from '../components/foundation';
import { useAppState } from '../state/AppState';
import { accountError, canManage, emptyBusiness, validateBusiness, type BusinessFields } from './accountForms';
import { isBusinessId } from './businessDisplay';
import { ResourceFeedback, useAccountResource } from './useAccountResource';

export function CreateBusiness() {
  return <PageContainer className="ql-account-page"><h1>Create business</h1><p className="ql-muted">Set up your business with one QueueLite account.</p><AccountAccess><BusinessForm initial={emptyBusiness} /></AccountAccess></PageContainer>;
}
export function BusinessSettings() {
  const { businessId } = useParams();
  return <PageContainer className="ql-account-page"><Link className="ql-back-link" to="/business/manage">← Your businesses</Link><h1>Business settings</h1><AccountAccess>{isBusinessId(businessId) ? <SettingsAccess key={businessId} businessId={businessId!} /> : <Card><InlineError>This business link is invalid.</InlineError></Card>}</AccountAccess></PageContainer>;
}
function SettingsAccess({ businessId }: { businessId: string }) {
  const { businesses, businessesStatus, refreshBusinesses } = useAppState();
  if (businessesStatus === 'loading') return <LoadingSkeleton label="Checking business access" />;
  if (businessesStatus === 'error') return <Card><InlineError>We couldn’t check your business access.</InlineError><Button onClick={() => refreshBusinesses()}>Try again</Button></Card>;
  if (!businesses.some(item => item.id === businessId && canManage(item))) return <Card><h2>Permission required</h2><InlineError>You do not have permission to manage this business. Owner or admin access is required.</InlineError><Link to="/business/manage">Your businesses</Link></Card>;
  return <SettingsContent businessId={businessId} />;
}
function SettingsContent({ businessId }: { businessId: string }) {
  const load = useCallback((signal: AbortSignal) => getBusiness(businessId, { signal }), [businessId]);
  const resource = useAccountResource(load);
  return <><ResourceFeedback {...resource} />{resource.data && <BusinessForm businessId={businessId} initial={{ ...resource.data, description: resource.data.description ?? '' }} />}</>;
}
function BusinessForm({ businessId, initial }: { businessId?: string; initial: BusinessFields }) {
  const { refreshBusinesses, refreshUser, setSelectedBusinessId } = useAppState();
  const navigate = useNavigate();
  const [fields, setFields] = useState(initial);
  const [errors, setErrors] = useState<ReturnType<typeof validateBusiness>>({});
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [message, setMessage] = useState('');
  const [uncertain, setUncertain] = useState(false);
  const lock = useRef(false);
  const bind = (name: Exclude<keyof BusinessFields, 'operational'>) => ({ name, value: fields[name], disabled: busy || uncertain, error: errors[name], onChange: (event: React.ChangeEvent<HTMLInputElement>) => setFields(previous => ({ ...previous, [name]: event.target.value })) });
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (lock.current || uncertain) return;
    const validation = validateBusiness(fields); setErrors(validation); setError(''); setMessage('');
    const first = Object.keys(validation)[0];
    if (first) { (event.currentTarget.elements.namedItem(first) as HTMLInputElement)?.focus(); return; }
    lock.current = true; setBusy(true);
    const body = { name: fields.name.trim(), location: fields.location.trim(), description: fields.description.trim(), openTime: fields.openTime, closeTime: fields.closeTime, email: fields.email.trim(), phoneNumber: fields.phoneNumber.trim() };
    try {
      if (businessId) {
        await updateBusiness(businessId, { ...body, operational: fields.operational });
        setMessage('Business settings saved.');
        refreshBusinesses(true);
      } else {
        const business = await createBusiness(body);
        setSelectedBusinessId(business.id);
        refreshBusinesses();
        navigate(`/business/${business.id}/settings`, { replace: true });
      }
    } catch (failure) {
      if (!businessId && failure instanceof APIError && (failure.status === 0 || failure.status >= 500 || failure.kind === 'response')) {
        setUncertain(true); refreshBusinesses();
        setError('We couldn’t confirm whether setup completed. Check your businesses before creating another business.');
      } else setError(accountError(failure));
      if (failure instanceof APIError && failure.status === 401) void refreshUser().catch(() => {});
    } finally { lock.current = false; setBusy(false); }
  }
  return <Card><form className="ql-stack" noValidate aria-busy={busy} onSubmit={event => { void submit(event); }}>
    <Input label="Business name" required {...bind('name')} /><Input label="Location" required {...bind('location')} />
    <div className="ql-field"><label htmlFor="business-description">Description (optional)</label><textarea id="business-description" name="description" className="ql-input" rows={3} value={fields.description} disabled={busy || uncertain} onChange={event => setFields({ ...fields, description: event.target.value })} /></div>
    <div className="ql-form-row"><Input label="Open time" type="time" required {...bind('openTime')} /><Input label="Close time" type="time" required {...bind('closeTime')} /></div>
    <p className="ql-meta">Daily hours. Overnight hours are supported.</p><Input label="Business email" type="email" autoComplete="email" required {...bind('email')} /><Input label="Business phone number" type="tel" autoComplete="tel" required {...bind('phoneNumber')} />
    {businessId && <div className="ql-field"><label className="ql-checkbox"><input type="checkbox" checked={fields.operational} disabled={busy} onChange={event => setFields({ ...fields, operational: event.target.checked })} />Operational</label><span className="ql-meta">When off, the business is unavailable for new queues. Hours are shown separately.</span></div>}
    {error && <InlineError>{error}</InlineError>}{message && <p role="status">{message}</p>}{uncertain && <Link to="/business/manage">Check your businesses</Link>}
    <Button type="submit" loading={busy} disabled={uncertain}>{businessId ? 'Save business settings' : 'Create business'}</Button>
    {businessId && <Link to="/plans" onClick={() => setSelectedBusinessId(businessId)}>View business subscription</Link>}
  </form></Card>;
}
