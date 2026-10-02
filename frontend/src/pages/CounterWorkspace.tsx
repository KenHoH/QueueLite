import { useCallback, useRef, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { callNextQueue, getCounter, processCalledQueue, removeQueueFromCounter, skipQueue } from '../api/counters';
import { getBusinessQueues } from '../api/operations';
import { AccountAccess } from '../components/AccountAccess';
import { Button, Card, InlineError, PageContainer, StatusBadge } from '../components/foundation';
import { currentQueue, operationsError } from './operationsDisplay';
import { OperationsFeedback } from './BusinessOperations';
import { PriorityBadge } from './Dashboard';
import { useOperations } from './useOperations';
import { isBusinessId } from './businessDisplay';

export default function CounterWorkspace() {
  const { counterId = '' } = useParams();
  return <PageContainer className="ql-ticket-page"><h1>Counter workspace</h1><AccountAccess>{isBusinessId(counterId) ? <Workspace key={counterId} counterId={counterId} /> : <Card><InlineError>This counter link is invalid.</InlineError></Card>}</AccountAccess></PageContainer>;
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
        else { await skipQueue(counterId, queue.id); setNotice('Customer skipped. The backend calls the next available customer.'); }
      }
      resource.retry();
    } catch (failure) {
      setError(operationsError(failure)); setUncertain(true); resource.retry();
    } finally { lock.current = false; setBusy(false); }
  }
  function refresh() { setError(''); setNotice(''); setUncertain(false); resource.retry(); }
  const active = queue && (queue.state === 'called' || queue.state === 'processing');
  return <div className="ql-stack"><OperationsFeedback {...resource} />{error && <Card><InlineError>{error}</InlineError><Button variant="secondary" disabled={busy} onClick={refresh}>Refresh state</Button></Card>}{notice && <p role="status">{notice}</p>}
    {data && <><Link to={`/business/${data.counter.businessId}/dashboard`}>← Business dashboard</Link><Card className="ql-ticket-card"><h2>{data.counter.name}</h2>
      {active ? <><StatusBadge state={queue.state} /> <PriorityBadge queue={queue} /><p className="ql-meta">Current queue</p><p className="ql-queue-number">{queue.name}</p><div className="ql-actions">{queue.state === 'called' ? <Button loading={busy} disabled={uncertain} onClick={() => void act('process')}>Start service</Button> : <Button loading={busy} disabled={uncertain} onClick={() => void act('complete')}>Complete</Button>}<Button variant="secondary" disabled={busy || uncertain} onClick={() => void act('skip')}>Skip</Button></div></> : data.counter.currentQueueId ? <><h3>Customer state pending</h3><p className="ql-muted">The assigned queue is not available in an active state. Refresh before operating this counter.</p></> : <><h3>No active customer</h3><Button loading={busy} disabled={uncertain} onClick={() => void act('call')}>Call next</Button></>}
      <p className="ql-meta">Waiting customers: {data.queues.filter(item => item.state === 'waiting').length}</p><Button variant="secondary" disabled={busy} onClick={refresh}>Refresh workspace</Button>
    </Card></>}
  </div>;
}
