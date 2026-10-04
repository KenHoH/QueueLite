import { after, test } from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';
const vite = await createServer({ configFile: false, optimizeDeps: { noDiscovery: true, include: [] }, server: { middlewareMode: true, ws: false }, appType: 'custom' });
after(() => vite.close());
const display = await vite.ssrLoadModule('/src/pages/queueDisplay.ts');
const { APIError } = await vite.ssrLoadModule('/src/api/errors.ts');
test('guest name and Indonesian phone validation matches the join contract', () => {
  assert.deepEqual(Object.keys(display.validateGuestJoin({ username: ' ', phoneNumber: '' })), ['username', 'phoneNumber']);
  for (const phoneNumber of ['0812 3456 7890', '+62 812-3456-7890', '6281234567890']) assert.deepEqual(display.validateGuestJoin({ username: 'Ayu', phoneNumber }), {});
  assert.ok(display.validateGuestJoin({ username: 'Ayu', phoneNumber: '123' }).phoneNumber);
});
test('known join errors have calm copy; ambiguous failures require checking before a new join', () => {
  for (const code of ['BUSINESS_NOT_FOUND', 'BUSINESS_UNAVAILABLE', 'ACTIVE_QUEUE_EXISTS', 'BUSINESS_QUEUE_FULL', 'PHONE_ALREADY_REGISTERED', 'INVALID_PHONE_NUMBER', 'USERNAME_REQUIRED', 'PHONE_NUMBER_REQUIRED', 'INVALID_FORMAT']) {
    const error = new APIError(400, code, 'sensitive raw details');
    assert.equal(display.isAmbiguousJoin(error), false);
    assert.ok(!display.joinErrorMessage(error).includes('sensitive'));
    assert.ok(!display.joinErrorMessage(error).includes('couldn’t join'));
  }
  for (const error of [new APIError(0, 'CLIENT_NETWORK_ERROR', 'raw'), new APIError(500, 'REGISTER_QUEUE_ERROR', 'raw'), new APIError(201, 'CLIENT_INVALID_RESPONSE', 'raw', 'response')]) {
    assert.equal(display.isAmbiguousJoin(error), true);
    assert.match(display.joinErrorMessage(error), /before joining again/);
  }
});
test('all wire states have distinct copy and terminal states are inactive', () => {
  for (const state of ['waiting', 'called', 'processing']) assert.equal(display.isActiveQueue(state), true);
  for (const state of ['done', 'skipped', 'cancelled']) assert.equal(display.isActiveQueue(state), false);
  assert.equal(Object.keys(display.queueHeadings).length, 6);
  const persistencePending = new APIError(409, 'QUEUE_PERSISTENCE_PENDING', 'raw');
  assert.equal(display.isQueuePersistencePending(persistencePending), true);
  assert.equal(display.isQueuePersistencePending(new APIError(404, 'QUEUE_NOT_READY', 'raw')), true);
  assert.equal(display.isQueuePersistencePending(new APIError(404, 'QUEUE_NOT_FOUND', 'raw')), false);
  assert.match(display.cancellationErrorMessage(persistencePending), /still being saved/);
  assert.match(display.ticketErrorMessage(persistencePending), /still being confirmed/);
  assert.match(display.ticketErrorMessage(new APIError(403, 'QUEUE_ACCESS_DENIED', 'raw')), /session/);
});
test('My Queues and cancellation use credentials and the owned state transition', async () => {
  const original = globalThis.fetch, calls = [];
  globalThis.fetch = async (url, init) => { calls.push({ url, init }); return Response.json(init.method === 'GET' ? [] : { message: 'updated' }); };
  try {
    const api = await vite.ssrLoadModule('/src/api/queues.ts');
    await api.getMyQueues(); await api.updateQueueState('ticket', 'cancelled');
    assert.equal(calls[0].url, '/api/queues/me');
    assert.equal(calls[0].init.method, 'GET');
    assert.equal(calls[1].url, '/api/queues/ticket/state');
    assert.equal(calls[1].init.method, 'PATCH');
    assert.equal(calls[1].init.body, '{"state":"cancelled"}');
    assert.ok(calls.every(call => call.init.credentials === 'include'));
  } finally { globalThis.fetch = original; }
});
