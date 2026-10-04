import { after, test } from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';

const vite = await createServer({ configFile: false, optimizeDeps: { noDiscovery: true, include: [] }, server: { middlewareMode: true, ws: false }, appType: 'custom' });
after(() => vite.close());

test('canonical and QR join use cookie credentials, guest-only identity and abort signal', async () => {
  const original = globalThis.fetch;
  const calls = [];
  const id = '11111111-1111-4111-8111-111111111111';
  const signal = new AbortController().signal;
  const guest = { username: 'Guest', phoneNumber: '081234567890' };
  globalThis.fetch = async (url, init) => { calls.push({ url, init }); return Response.json({ id, businessId: id, name: 'A001', state: 'waiting', priority: false }); };
  try {
    const api = await vite.ssrLoadModule('/src/api/queues.ts');
    await api.joinBusinessQueue(id, undefined, { signal });
    await api.joinBusinessQueue(id, guest, { signal });
    await api.registerQueueByQR(id, guest, { signal });
    assert.equal(calls[0].url, `/api/queues/business/${id}/join`);
    assert.equal(calls[0].init.body, undefined);
    assert.equal(calls[1].init.body, JSON.stringify(guest));
    assert.equal(calls[2].url, `/api/queues/qr/${id}`);
    assert.ok(calls.every(({ init }) => init.credentials === 'include' && init.method === 'POST' && init.signal instanceof AbortSignal));
    assert.ok(calls.every(({ init }) => !init.body || !init.body.includes('userId')));
  } finally { globalThis.fetch = original; }
});

test('QR resolution uses the public resolve contract without joining', async () => {
  const original = globalThis.fetch;
  const id = '11111111-1111-4111-8111-111111111111';
  let call;
  globalThis.fetch = async (url, init) => { call = { url, init }; return Response.json({ businessId: id, authenticated: false, requiresGuestForm: true, requiredFields: ['username', 'phoneNumber'] }); };
  try {
    const api = await vite.ssrLoadModule('/src/api/queues.ts');
    assert.equal((await api.resolveQueueQR(id)).requiresGuestForm, true);
    assert.equal(call.url, `/api/queues/qr/${id}/resolve`);
    assert.equal(call.init.method, 'GET');
    assert.equal(call.init.body, undefined);
  } finally { globalThis.fetch = original; }
});

test('join retains typed business and phone errors', async () => {
  const original = globalThis.fetch;
  try {
    const api = await vite.ssrLoadModule('/src/api/queues.ts');
    for (const [status, code] of [[409, 'BUSINESS_UNAVAILABLE'], [409, 'PHONE_ALREADY_REGISTERED'], [409, 'BUSINESS_QUEUE_FULL'], [404, 'BUSINESS_NOT_FOUND'], [400, 'INVALID_PHONE_NUMBER']]) {
      globalThis.fetch = async () => Response.json({ error: { code, message: 'Join rejected' } }, { status });
      await assert.rejects(api.joinBusinessQueue('business', { username: 'Guest', phoneNumber: '123' }), error => error.code === code && error.status === status);
    }
  } finally { globalThis.fetch = original; }
});
