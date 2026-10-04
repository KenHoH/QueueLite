import { useCallback, useEffect, useRef, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { callNextQueue, getCounter, processCalledQueue, removeQueueFromCounter, skipQueue } from '../api/counters';
import { getBusinessQueues } from '../api/operations';
import { AccountAccess } from '../components/AccountAccess';
import { Button, Card, InlineError, PageContainer, StatusBadge } from '../components/foundation';
import { currentQueue, elapsedServiceTime, operationsError } from './operationsDisplay';
import { OperationsFeedback } from './BusinessOperations';
import { PriorityBadge } from './Dashboard';
import { useOperations } from './useOperations';
import { isBusinessId } from './businessDisplay';

export default function CounterWorkspace() {
  const { counterId = '' } = useParams();
  return <PageContainer className="ql-counter-workspace-page"><h1>Counter workspace</h1><AccountAccess>{isBusinessId(counterId) ? <Workspace key={counterId} counterId={counterId} /> : <Card><InlineError>This counter link is invalid.</InlineError></Card>}</AccountAccess></PageContainer>;
}

function ServiceTimer({ startedAt }: { startedAt?: string }) {
  const [now, setNow] = useState(Date.now());
  useEffect(() => {
    if (!startedAt) return;
    setNow(Date.now());
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, [startedAt]);
  return <output className="ql-service-timer" aria-label="Time taken">{startedAt ? elapsedServiceTime(startedAt, now) : 'Timing unavailable'}</output>;
}

function Workspace({ counterId }: { counterId: string }) {
  const load = useCallback(async (signal: AbortSignal) => {
    const counter = await getCounter(counterId, { signal });
    const queues = await getBusinessQueues(counter.businessId, { signal });
    return { counter, queues };
  }, [counterId]);
  const resource = useOperations(load, false);
  const [busy, setBusy] = useState(false), [error, setError] = useState(''), [notice, setNotice] = useState(''), [uncertain, setUncertain] = useState(false);
  const lock = useRef(false);
  const data = resource.data, queue = data && currentQueue(data.counter, data.queues);

  async function act(action: 'call' | 'process' | 'complete' | 'skip') {
    if (lock.current || !data || uncertain) return;
    lock.current = true; setBusy(true); setError(''); setNotice('');
    try {
      if (action === 'call') {
        const result = await callNextQueue(counterId, data.counter.businessId);
        if ('queue' in result && result.queue === null) setNotice('No waiting customer available.');
      } else if (queue) {
        if (action === 'process') await processCalledQueue(counterId, queue.id);
        else if (action === 'complete') await removeQueueFromCounter(counterId, queue.id);
        else { await skipQueue(counterId, queue.id); setNotice('Customer skipped. The scheduler selected the next available customer.'); }
      }
      resource.retry();
    } catch (failure) {
      setError(operationsError(failure)); setUncertain(true); resource.retry();
    } finally { lock.current = false; setBusy(false); }
  }

  function refresh() { setError(''); setNotice(''); setUncertain(false); resource.retry(); }
  const active = queue && (queue.state === 'called' || queue.state === 'processing');
  const waiting = data?.queues.filter(item => item.state === 'waiting').length ?? 0;

  return <div className="ql-stack">
    <OperationsFeedback {...resource} />
    {error && <Card><InlineError>{error}</InlineError><Button variant="secondary" disabled={busy} onClick={refresh}>Refresh state</Button></Card>}
    {notice && <p role="status">{notice}</p>}
    {data && <>
      <Link className="ql-back-link" to={`/business/${data.counter.businessId}/dashboard`}>← Business dashboard</Link>
      <Card className="ql-workspace-header"><div><p className="ql-eyebrow">Active service counter</p><h2>{data.counter.name}</h2><p className="ql-muted">Waiting customers: {waiting}</p></div><Button variant="secondary" disabled={busy} onClick={refresh}>Refresh workspace</Button></Card>
      <div className="ql-workspace-grid">
        <Card className="ql-workspace-current" aria-labelledby="current-customer-heading">
          <h2 id="current-customer-heading">Customer being handled</h2>
          {active ? <>
            <div className="ql-workspace-badges"><StatusBadge state={queue.state} /><PriorityBadge queue={queue} /></div>
            <p className="ql-queue-number">{queue.name}</p>
            {queue.state === 'processing' ? <div className="ql-time-panel"><span className="ql-meta">Time taken</span><ServiceTimer startedAt={queue.processingAt} /></div> : <p className="ql-muted">Service time starts after you begin service.</p>}
            <div className="ql-actions">{queue.state === 'called'
              ? <Button loading={busy} disabled={uncertain} onClick={() => void act('process')}>Start service</Button>
              : <Button loading={busy} disabled={uncertain} onClick={() => void act('complete')}>Complete</Button>}
              <Button variant="secondary" disabled={busy || uncertain} onClick={() => void act('skip')}>Skip</Button>
            </div>
          </> : data.counter.currentQueueId ? <><h3>Customer state pending</h3><p className="ql-muted">The assigned queue is not available in an active state. Refresh before operating this counter.</p></> : <><p className="ql-workspace-empty">No active customer</p><Button loading={busy} disabled={uncertain} onClick={() => void act('call')}>Call next</Button></>}
        </Card>
        <Card className="ql-workspace-next" aria-labelledby="next-customer-heading">
          <h2 id="next-customer-heading">Next customer</h2>
          <p className="ql-workspace-next-value">Selected by scheduler</p>
          <p className="ql-muted">The next ticket is selected securely when you call or skip, including priority and fairness rules.</p>
          <p className="ql-meta">{waiting} {waiting === 1 ? 'customer' : 'customers'} waiting</p>
        </Card>
      </div>
    </>}
  </div>;
}
