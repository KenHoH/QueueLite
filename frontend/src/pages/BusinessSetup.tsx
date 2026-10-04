import { useCallback, useRef, useState, type FormEvent } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { createBusiness, deleteBusiness, getBusiness, updateBusiness } from '../api/businesses';
import { APIError } from '../api/errors';
import { AccountAccess } from '../components/AccountAccess';
import { BusinessNavigation } from '../components/BusinessNavigation';
import { Button, Card, InlineError, Input, LoadingSkeleton, PageContainer, PageHeader } from '../components/foundation';
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from '../components/ui/alert-dialog';
import { Checkbox } from '../components/ui/checkbox';
import { Label } from '../components/ui/label';
import { Textarea } from '../components/ui/textarea';
import { useAppState } from '../state/AppState';
import { accountError, canManage, emptyBusiness, validateBusiness, type BusinessFields } from './accountForms';
import { isBusinessId } from './businessDisplay';
import { ResourceFeedback, useAccountResource } from './useAccountResource';

export function CreateBusiness() {
  return <PageContainer className="ql-account-page"><PageHeader title="Create business" description="Add public details and operating hours. You can invite staff after setup." /><AccountAccess><BusinessForm initial={emptyBusiness} /></AccountAccess></PageContainer>;
}
export function BusinessSettings() {
  const { businessId } = useParams();
  return <PageContainer className="ql-account-page"><PageHeader eyebrow="Business tools" title="Business settings" action={<Link to="/business/manage">All businesses</Link>} /><AccountAccess>{isBusinessId(businessId) ? <SettingsAccess key={businessId} businessId={businessId!} /> : <Card><InlineError>This business link is invalid.</InlineError></Card>}</AccountAccess></PageContainer>;
}
function SettingsAccess({ businessId }: { businessId: string }) {
  const { businesses, businessesStatus, refreshBusinesses } = useAppState();
  if (businessesStatus === 'loading') return <LoadingSkeleton label="Checking business access" />;
  if (businessesStatus === 'error') return <Card><InlineError>We couldn’t check your business access.</InlineError><Button onClick={() => refreshBusinesses()}>Try again</Button></Card>;
  if (!businesses.some(item => item.id === businessId && canManage(item))) return <Card><h2>Permission required</h2><InlineError>You do not have permission to manage this business. Owner or admin access is required.</InlineError><Link to="/business/manage">Your businesses</Link></Card>;
  return <div className="ql-stack"><BusinessNavigation businessId={businessId} /><SettingsContent businessId={businessId} /></div>;
}
function SettingsContent({ businessId }: { businessId: string }) {
  const load = useCallback((signal: AbortSignal) => getBusiness(businessId, { signal }), [businessId]);
  const resource = useAccountResource(load);
  return <><ResourceFeedback {...resource} />{resource.data && <><BusinessForm businessId={businessId} initial={{ ...resource.data, description: resource.data.description ?? '' }} /><BusinessDangerZone businessId={businessId} businessName={resource.data.name} /></>}</>;
}
function BusinessDangerZone({ businessId, businessName }: { businessId: string; businessName: string }) {
  const { refreshBusinesses, setSelectedBusinessId } = useAppState();
  const navigate = useNavigate();
  const [open, setOpen] = useState(false), [busy, setBusy] = useState(false), [error, setError] = useState('');
  async function remove() {
    if (busy) return;
    setBusy(true); setError('');
    try {
      await deleteBusiness(businessId);
      setSelectedBusinessId(null); refreshBusinesses(); navigate('/business/manage', { replace: true });
    } catch (failure) { setError(accountError(failure)); setOpen(false); }
    finally { setBusy(false); }
  }
  return <section className="ql-danger-zone" aria-labelledby="delete-business-heading"><div><h2 id="delete-business-heading">Delete business</h2><p className="ql-muted">Permanently remove {businessName}. Existing related data may prevent deletion; this cannot be undone.</p></div>
    <AlertDialog open={open} onOpenChange={setOpen}><AlertDialogTrigger asChild><Button variant="danger">Delete business</Button></AlertDialogTrigger><AlertDialogContent><AlertDialogHeader><AlertDialogTitle>Delete {businessName}?</AlertDialogTitle><AlertDialogDescription>This permanently deletes the business. Stop active service and confirm that retained records are no longer needed before continuing.</AlertDialogDescription></AlertDialogHeader><AlertDialogFooter><AlertDialogCancel disabled={busy}>Keep business</AlertDialogCancel><AlertDialogAction disabled={busy} onClick={event => { event.preventDefault(); void remove(); }}>{busy ? 'Deleting…' : 'Permanently delete'}</AlertDialogAction></AlertDialogFooter></AlertDialogContent></AlertDialog>
    {error && <InlineError>{error}</InlineError>}
  </section>;
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
      if (failure instanceof APIError && failure.status === 401) void refreshUser(true).catch(() => {});
    } finally { lock.current = false; setBusy(false); }
  }
  return <Card><form className="ql-stack" noValidate aria-busy={busy} onSubmit={event => { void submit(event); }}>
    <Input label="Business name" required {...bind('name')} /><Input label="Location" required {...bind('location')} />
    <div className="ql-field"><Label htmlFor="business-description">Description (optional)</Label><Textarea id="business-description" name="description" rows={3} value={fields.description} disabled={busy || uncertain} onChange={event => setFields({ ...fields, description: event.target.value })} /></div>
    <div className="ql-form-row"><Input label="Open time" type="time" required {...bind('openTime')} /><Input label="Close time" type="time" required {...bind('closeTime')} /></div>
    <p className="ql-meta">Daily hours. Overnight hours are supported.</p><Input label="Business email" type="email" autoComplete="email" required {...bind('email')} /><Input label="Business phone number" type="tel" autoComplete="tel" required {...bind('phoneNumber')} />
    {businessId && <div className="ql-field"><div className="ql-checkbox"><Checkbox id="business-operational" checked={fields.operational} disabled={busy} onCheckedChange={checked => setFields({ ...fields, operational: checked === true })} /><Label htmlFor="business-operational">Operational</Label></div><span className="ql-meta">When off, the business is unavailable for new queues. Hours are shown separately.</span></div>}
    {error && <InlineError>{error}</InlineError>}{message && <p role="status">{message}</p>}{uncertain && <Link to="/business/manage">Check your businesses</Link>}
    <Button type="submit" loading={busy} disabled={uncertain}>{businessId ? 'Save business settings' : 'Create business'}</Button>
    {businessId && <Link to={`/business/${businessId}/plans`} onClick={() => setSelectedBusinessId(businessId)}>View business subscription</Link>}
  </form></Card>;
}
