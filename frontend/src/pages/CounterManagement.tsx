import { useCallback, useRef, useState, type FormEvent } from 'react';
import { Link } from 'react-router-dom';
import { createCounter, deleteCounter, updateCounter } from '../api/counters';
import { getBusinessCounters, getBusinessMembers, getBusinessQueues } from '../api/operations';
import type { BusinessMember, BusinessMembership, Counter, Queue } from '../api/types';
import { Button, Card, EmptyState, InlineError, Input } from '../components/foundation';
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
  return <><p className="ql-muted">{business.name} · {manager ? 'Manage counters and assign existing business members.' : 'Your assigned counters. Owners and admins manage assignments.'}</p><OperationsFeedback {...resource} />
    {manager && resource.data && <Card><h2>Create counter</h2><form className="ql-stack" onSubmit={create}><Input label="Counter name" value={name} onChange={event => setName(event.target.value)} disabled={busy || uncertain} maxLength={120} />{error && <InlineError>{error}</InlineError>}<Button type="submit" loading={busy} disabled={uncertain}>Create counter</Button></form></Card>}
    {error && !resource.data && <InlineError>{error}</InlineError>}
    {resource.data && <><Button variant="secondary" onClick={refresh} disabled={busy}>Refresh counters</Button>{!resource.data.counters.length ? <EmptyState title="No counters available" description={manager ? 'Create a counter to begin serving customers.' : 'Ask an owner or admin to assign a counter to you.'} /> : <div className="ql-counter-grid">{resource.data.counters.map(counter => <CounterEditor key={counter.id} counter={counter} queues={resource.data!.queues} members={resource.data!.members} manager={manager} refresh={resource.retry} />)}</div>}</>}
  </>;
}
function CounterEditor({ counter, queues, members, manager, refresh }: { counter: Counter; queues: Queue[]; members: BusinessMember[]; manager: boolean; refresh: () => void }) {
  const [name, setName] = useState(counter.name), [employee, setEmployee] = useState(counter.currentEmployeeId ?? ''), [confirm, setConfirm] = useState(false), [busy, setBusy] = useState(false), [error, setError] = useState(''), [uncertain, setUncertain] = useState(false);
  const lock = useRef(false); const queue = currentQueue(counter, queues);
  const assigned = members.find(member => member.userId === counter.currentEmployeeId);
  async function mutate(action: () => Promise<unknown>) {
    if (lock.current || uncertain) return; lock.current = true; setBusy(true); setError('');
    try { await action(); refresh(); }
    catch (failure) { setError(operationsError(failure)); setUncertain(true); }
    finally { lock.current = false; setBusy(false); }
  }
  return <Card><h2>{counter.name}</h2><p>{counterStatus(counter, queues)}</p><p>Assigned employee: {assigned?.username ?? (counter.currentEmployeeId ? 'Assigned member (details unavailable)' : 'Unassigned')}</p>{queue && <p>Current queue: <strong>{queue.name}</strong></p>}
    <p><Link to={`/counter/${counter.id}`}>Open workspace →</Link></p>
    {manager && <form className="ql-stack" onSubmit={event => { event.preventDefault(); if (!name.trim()) { setError('Enter a counter name.'); return; } void mutate(() => updateCounter(counter.id, { name: name.trim(), currentEmployeeId: employee })); }}>
      <Input label={`Name for ${counter.name}`} value={name} disabled={busy || uncertain} onChange={event => setName(event.target.value)} maxLength={120} />
      <div className="ql-field"><label htmlFor={`employee-${counter.id}`}>Employee for {counter.name}</label><select className="ql-input" id={`employee-${counter.id}`} value={employee} disabled={busy || uncertain} onChange={event => setEmployee(event.target.value)}><option value="">Unassigned</option>{counter.currentEmployeeId && !assigned && <option value={counter.currentEmployeeId}>Current assignment unavailable</option>}{members.map(member => <option key={member.userId} value={member.userId}>{member.username} ({member.role})</option>)}</select></div>
      <Button type="submit" loading={busy} disabled={uncertain}>Save counter</Button>
      {confirm ? <div className="ql-stack"><p>Delete {counter.name}?</p><Button variant="danger" disabled={busy || uncertain || !!counter.currentQueueId} onClick={() => void mutate(() => deleteCounter(counter.id))}>Confirm delete</Button><Button variant="secondary" disabled={busy} onClick={() => setConfirm(false)}>Keep counter</Button></div> : <Button variant="secondary" disabled={busy || uncertain || !!counter.currentQueueId} onClick={() => setConfirm(true)}>Delete counter</Button>}
      {counter.currentQueueId && <p className="ql-meta">Finish the active customer before deleting this counter.</p>}
    </form>}{error && <><InlineError>{error}</InlineError><Button variant="secondary" disabled={busy} onClick={refresh}>Refresh state</Button></>}
  </Card>;
}
