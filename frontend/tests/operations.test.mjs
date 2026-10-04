import { after, test } from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';
const vite = await createServer({ configFile: false, optimizeDeps: { noDiscovery: true, include: [] }, server: { middlewareMode: true, ws: false }, appType: 'custom' });
after(() => vite.close());
const display = await vite.ssrLoadModule('/src/pages/operationsDisplay.ts');
test('operational grouping includes only real waiting/called/processing states', () => {
  const queues = ['waiting', 'called', 'processing', 'done', 'skipped', 'cancelled'].map((state, i) => ({ id: String(i), state }));
  const groups = display.groupQueues(queues);
  assert.deepEqual(Object.values(groups).map(list => list.map(q => q.state)), [['waiting'], ['called'], ['processing']]);
  assert.deepEqual(display.groupQueues([]), { waiting: [], called: [], processing: [] });
});
test('counter state never treats missing queue details as an available desk', () => {
  assert.equal(display.counterStatus({}, []), 'Available');
  assert.equal(display.counterStatus({ currentQueueId: 'q' }, []), 'State pending — refresh');
  assert.equal(display.counterStatus({ currentQueueId: 'q' }, [{ id: 'q', state: 'called' }]), 'Called');
  assert.equal(display.counterStatus({ currentQueueId: 'q' }, [{ id: 'q', state: 'processing' }]), 'In service');
});
test('service time uses server timestamps and handles invalid or future values safely', () => {
  const now = Date.parse('2026-10-02T12:01:05Z');
  assert.equal(display.elapsedServiceTime('2026-10-02T11:00:00Z', now), '01:01:05');
  assert.equal(display.elapsedServiceTime('invalid', now), '00:00:00');
  assert.equal(display.elapsedServiceTime('2026-10-02T13:00:00Z', now), '00:00:00');
  assert.equal(display.averageServiceSeconds([
    { state: 'done', processingAt: '2026-10-02T11:00:00Z', doneAt: '2026-10-02T11:02:00Z' },
    { state: 'done', processingAt: '2026-10-02T11:00:00Z', doneAt: '2026-10-02T11:04:00Z' },
    { state: 'skipped', processingAt: '2026-10-02T11:00:00Z', doneAt: '2026-10-02T11:20:00Z' },
  ]), 180);
  assert.equal(display.averageServiceSeconds([{ state: 'done' }]), null);
});
test('staff listing APIs encode resource IDs and use credentials without browser identity', async () => {
  const original = globalThis.fetch, requests = [];
  globalThis.fetch = async (url, init) => { requests.push({ url, init }); return Response.json([]); };
  try {
    const api = await vite.ssrLoadModule('/src/api/operations.ts');
    await api.getBusinessCounters('business/id'); await api.getBusinessMembers('business/id'); await api.getBusinessQueues('business/id');
    assert.deepEqual(requests.map(r => r.url), ['/api/businesses/business%2Fid/counters', '/api/businesses/business%2Fid/members', '/api/businesses/business%2Fid/queues']);
    assert.ok(requests.every(r => r.init.credentials === 'include' && !r.init.body));
    globalThis.fetch = async () => Response.json({ wrong: true });
    await assert.rejects(api.getBusinessCounters('id'), e => e.code === 'CLIENT_INVALID_RESPONSE');
  } finally { globalThis.fetch = original; }
});
test('stale, persistence and authorization errors offer safe operational guidance', async () => {
  const { APIError } = await vite.ssrLoadModule('/src/api/errors.ts');
  assert.match(display.operationsError(new APIError(409, 'COUNTER_STATE_CHANGED', 'secret')), /state changed/);
  assert.match(display.operationsError(new APIError(404, 'QUEUE_NOT_READY', 'secret')), /persistence is pending/);
  assert.match(display.operationsError(new APIError(403, 'BUSINESS_ACCESS_DENIED', 'secret')), /permission/);
  assert.doesNotMatch(display.operationsError(new APIError(500, 'INTERNAL', 'secret')), /secret/);
});
