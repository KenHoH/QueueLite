import { useCallback, useRef, useState, type FormEvent } from 'react';
import { Link } from 'react-router-dom';
import { upsertBusinessMember } from '../api/businesses';
import { createCounter, deleteCounter, updateCounter } from '../api/counters';
import { APIError } from '../api/errors';
import { getBusinessCounters, getBusinessMembers, getBusinessQueues } from '../api/operations';
import type { BusinessMember, BusinessMembership, Counter, Queue } from '../api/types';
import { Button, Card, EmptyState, InlineError, Input } from '../components/foundation';
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from '../components/ui/alert-dialog';
import { Badge } from '../components/ui/badge';
import { Label } from '../components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../components/ui/select';
import { BusinessOperationsPage, OperationsFeedback } from './BusinessOperations';
import { canManage } from './accountForms';
import { counterStatus, currentQueue, operationsError } from './operationsDisplay';
import { useOperations } from './useOperations';

export default function CounterManagement() { return <BusinessOperationsPage title="Counter management">{business => <Management business={business} />}</BusinessOperationsPage>; }
function Management({ business }: { business: BusinessMembership }) {
  const manager = canManage(business);
  const load = useCallback(async (signal: AbortSignal) => {
    const [counters, queues, members] = await Promise.all([getBusinessCounters(business.id, { signal }), getBusinessQueues(business.id, { signal }), manager ? getBusinessMembers(business.id, { signal }) : Promise.resolve([] as BusinessMember[])]);
    return { counters, queues, members };
  }, [business.id, manager]);
  const resource = useOperations(load, false);
  const [name, setName] = useState(''), [busy, setBusy] = useState(false), [error, setError] = useState(''), [uncertain, setUncertain] = useState(false);
  const lock = useRef(false);
  async function create(event: FormEvent) {
    event.preventDefault(); if (lock.current || uncertain) return;
    if (!name.trim()) { setError('Enter a counter name.'); return; }
    lock.current = true; setBusy(true); setError('');
    try { await createCounter({ businessId: business.id, name: name.trim() }); setName(''); resource.retry(); }
    catch (failure) { setError(operationsError(failure)); setUncertain(true); resource.retry(); }
    finally { lock.current = false; setBusy(false); }
  }
  function refresh() { setUncertain(false); setError(''); resource.retry(); }
  return <><p className="ql-muted">{business.name} · {manager ? 'Manage members, counters, and staff assignments.' : 'Your assigned counters. Owners and admins manage assignments.'}</p><OperationsFeedback {...resource} />
    {manager && <MemberManagement business={business} members={resource.data?.members ?? []} ready={!!resource.data} refresh={resource.retry} />}
    {manager && resource.data && <Card><h2>Create counter</h2><form className="ql-stack" onSubmit={create}><Input label="Counter name" value={name} onChange={event => setName(event.target.value)} disabled={busy || uncertain} maxLength={120} />{error && <InlineError>{error}</InlineError>}<Button type="submit" loading={busy} disabled={uncertain}>Create counter</Button></form></Card>}
    {error && !resource.data && <InlineError>{error}</InlineError>}
    {resource.data && <><Button variant="secondary" onClick={refresh} disabled={busy}>Refresh counters</Button>{!resource.data.counters.length ? <EmptyState title="No counters available" description={manager ? 'Create a counter to begin serving customers.' : 'Ask an owner or admin to assign a counter to you.'} /> : <div className="ql-counter-grid">{resource.data.counters.map(counter => <CounterEditor key={counter.id} counter={counter} queues={resource.data!.queues} members={resource.data!.members} manager={manager} refresh={resource.retry} />)}</div>}</>}
  </>;
}
function memberError(error: unknown) {
  if (error instanceof APIError) {
    if (error.code === 'USER_NOT_FOUND') return 'No QueueLite account matches that username or email.';
    if (error.code === 'MEMBER_IDENTIFIER_REQUIRED') return 'Enter a username or email.';
    if (error.code === 'INVALID_MEMBER_ROLE') return 'Choose an available business role.';
    if (['MEMBER_ROLE_DENIED', 'BUSINESS_OWNER_PROTECTED', 'MEMBER_SELF_CHANGE'].includes(error.code)) return 'That business role cannot be changed by your account.';
    if (error.status === 401) return 'Your session has ended. Please sign in again.';
    if (error.status === 403) return 'You do not have permission to assign that role.';
  }
  return 'We couldn’t confirm the member change. Refresh members before trying again.';
}
export function MemberManagement({ business, members, ready, refresh }: { business: BusinessMembership; members: BusinessMember[]; ready: boolean; refresh: () => void }) {
  const [identifier, setIdentifier] = useState(''), [role, setRole] = useState<'admin' | 'counter'>('counter');
  const [busy, setBusy] = useState(false), [error, setError] = useState(''), [message, setMessage] = useState(''), [uncertain, setUncertain] = useState(false);
  const lock = useRef(false);
  async function submit(event: FormEvent) {
    event.preventDefault();
    if (lock.current || uncertain || !ready) return;
    if (!identifier.trim()) { setError('Enter a username or email.'); return; }
    lock.current = true; setBusy(true); setError(''); setMessage('');
    try {
      const member = await upsertBusinessMember(business.id, { identifier: identifier.trim(), role });
      setIdentifier(''); setMessage(`${member.username} now has the ${member.role} role.`); refresh();
    } catch (failure) {
      setError(memberError(failure));
      if (!(failure instanceof APIError) || failure.status === 0 || failure.status >= 500 || failure.kind === 'network' || failure.kind === 'response') setUncertain(true);
    } finally { lock.current = false; setBusy(false); }
  }
  function reload() { setUncertain(false); setError(''); refresh(); }
  return <Card><h2>Business members</h2><p className="ql-muted">Add an existing QueueLite account by exact username or email. Owners can assign admin or counter access; admins can assign counter access.</p>
    <form className="ql-stack" noValidate onSubmit={event => { void submit(event); }}>
      <Input label="Username or email" name="memberIdentifier" autoComplete="off" value={identifier} disabled={busy || uncertain || !ready} onChange={event => setIdentifier(event.target.value)} />
      <div className="ql-field"><Label htmlFor="member-role">Business role</Label><Select value={role} disabled={busy || uncertain || !ready} onValueChange={value => setRole(value as 'admin' | 'counter')}><SelectTrigger id="member-role"><SelectValue /></SelectTrigger><SelectContent><SelectItem value="counter">Counter</SelectItem>{business.role === 'owner' && <SelectItem value="admin">Admin</SelectItem>}</SelectContent></Select></div>
      {error && <InlineError>{error}</InlineError>}{message && <p role="status">{message}</p>}
      <Button type="submit" loading={busy} disabled={uncertain || !ready}>Add or update member</Button>
      {uncertain && <Button variant="secondary" onClick={reload}>Refresh members</Button>}
    </form>
    {ready && <div className="ql-stack"><h3>Current members</h3>{members.length ? <ul>{members.map(member => <li key={member.userId}>{member.username} <Badge variant="secondary">{member.role}</Badge></li>)}</ul> : <p className="ql-muted">No members are available.</p>}</div>}
  </Card>;
}
function CounterEditor({ counter, queues, members, manager, refresh }: { counter: Counter; queues: Queue[]; members: BusinessMember[]; manager: boolean; refresh: () => void }) {
  const [name, setName] = useState(counter.name), [employee, setEmployee] = useState(counter.currentEmployeeId ?? ''), [confirm, setConfirm] = useState(false), [busy, setBusy] = useState(false), [error, setError] = useState(''), [uncertain, setUncertain] = useState(false);
  const lock = useRef(false); const queue = currentQueue(counter, queues);
  const assigned = members.find(member => member.userId === counter.currentEmployeeId);
  async function mutate(action: () => Promise<unknown>) {
    if (lock.current || uncertain) return; lock.current = true; setBusy(true); setError('');
    try { await action(); setConfirm(false); refresh(); }
    catch (failure) { setError(operationsError(failure)); setUncertain(true); }
    finally { lock.current = false; setBusy(false); }
  }
  return <Card><h2>{counter.name}</h2><p>{counterStatus(counter, queues)}</p><p>Assigned employee: {assigned?.username ?? (counter.currentEmployeeId ? 'Assigned member (details unavailable)' : 'Unassigned')}</p>{queue && <p>Current queue: <strong>{queue.name}</strong></p>}
    <p><Link to={`/counter/${counter.id}`}>Open workspace →</Link></p>
    {manager && <form className="ql-stack" onSubmit={event => { event.preventDefault(); if (!name.trim()) { setError('Enter a counter name.'); return; } void mutate(() => updateCounter(counter.id, { name: name.trim(), currentEmployeeId: employee })); }}>
      <Input label={`Name for ${counter.name}`} value={name} disabled={busy || uncertain} onChange={event => setName(event.target.value)} maxLength={120} />
      <div className="ql-field"><Label htmlFor={`employee-${counter.id}`}>Employee for {counter.name}</Label><Select value={employee || 'unassigned'} disabled={busy || uncertain} onValueChange={value => setEmployee(value === 'unassigned' ? '' : value)}><SelectTrigger id={`employee-${counter.id}`}><SelectValue /></SelectTrigger><SelectContent><SelectItem value="unassigned">Unassigned</SelectItem>{counter.currentEmployeeId && !assigned && <SelectItem value={counter.currentEmployeeId}>Current assignment unavailable</SelectItem>}{members.map(member => <SelectItem key={member.userId} value={member.userId}>{member.username} ({member.role})</SelectItem>)}</SelectContent></Select></div>
      <Button type="submit" loading={busy} disabled={uncertain}>Save counter</Button>
      <AlertDialog open={confirm} onOpenChange={setConfirm}><AlertDialogTrigger asChild><Button variant="secondary" disabled={busy || uncertain || !!counter.currentQueueId}>Delete counter</Button></AlertDialogTrigger><AlertDialogContent><AlertDialogHeader><AlertDialogTitle>Delete {counter.name}?</AlertDialogTitle><AlertDialogDescription>This permanently removes the counter. This action cannot be undone.</AlertDialogDescription></AlertDialogHeader><AlertDialogFooter><AlertDialogCancel disabled={busy}>Keep counter</AlertDialogCancel><AlertDialogAction disabled={busy || uncertain || !!counter.currentQueueId} onClick={event => { event.preventDefault(); void mutate(() => deleteCounter(counter.id)); }}>Confirm delete</AlertDialogAction></AlertDialogFooter></AlertDialogContent></AlertDialog>
      {counter.currentQueueId && <p className="ql-meta">Finish the active customer before deleting this counter.</p>}
    </form>}{error && <><InlineError>{error}</InlineError><Button variant="secondary" disabled={busy} onClick={refresh}>Refresh state</Button></>}
  </Card>;
}
