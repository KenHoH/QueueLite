import { after, test } from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';

// Vite transforms the same TS/env modules used by the browser; no extra test dependency.
const vite = await createServer({ configFile: false, optimizeDeps: { noDiscovery: true, include: [] }, server: { middlewareMode: true, ws: false }, appType: 'custom' });
after(() => vite.close());
const { createAPIClient } = await vite.ssrLoadModule('/src/api/client.ts');
const { APIError } = await vite.ssrLoadModule('/src/api/errors.ts');

test('sends cookie credentials, JSON body and encoded cursor queries', async () => {
  let captured;
  const request = createAPIClient('/api/', async (url, init) => {
    captured = { url, init };
    return Response.json({ data: [], nextCursor: null });
  });
  const result = await request('/businesses/', { method: 'POST', body: { name: 'A' }, query: { cursorCreatedAt: '2026-09-29T10:00:00+07:00', limit: 20, absent: undefined } });
  assert.deepEqual(result, { data: [], nextCursor: null });
  assert.equal(captured.init.credentials, 'include');
  assert.equal(captured.init.headers['Content-Type'], 'application/json');
  assert.equal(captured.init.body, '{"name":"A"}');
  const url = new URL(captured.url, 'http://localhost');
  assert.equal(url.pathname, '/api/businesses/');
  assert.equal(url.searchParams.get('cursorCreatedAt'), '2026-09-29T10:00:00+07:00');
  assert.equal(url.searchParams.has('absent'), false);
});

test('GET has no JSON content type or body; preserves empty-counter response', async () => {
  const request = createAPIClient('/api', async (_, init) => {
    assert.equal(init.body, undefined);
    assert.equal(init.headers['Content-Type'], undefined);
    return Response.json({ queue: null });
  });
  assert.deepEqual(await request('/counters/example'), { queue: null });
});

test('keeps domain codes and status even when a conflict-like error is 400', async () => {
  for (const [status, code, kind] of [[400, 'ACTIVE_QUEUE_EXISTS', 'validation'], [409, 'PHONE_ALREADY_REGISTERED', 'conflict'], [400, 'BUSINESS_QUEUE_FULL', 'validation'], [400, 'INVALID_PRIORITY_QUOTA', 'validation']]) {
    const request = createAPIClient('/api', async () => Response.json({ error: { code, message: 'backend detail' } }, { status }));
    await assert.rejects(request('/queues/'), e => e instanceof APIError && e.status === status && e.code === code && e.kind === kind);
  }
});

test('handles plain-text middleware errors and proxy HTML without leaking markup', async () => {
  for (const [status, body, kind] of [[401, 'unauthorized\n', 'unauthorized'], [502, '<html>upstream unavailable</html>', 'server']]) {
    const request = createAPIClient('/api', async () => new Response(body, { status }));
    await assert.rejects(request('/queues/id'), e => e.kind === kind && e.code === `CLIENT_HTTP_${status}` && !e.message.includes('<html>'));
  }
});

test('treats malformed successful HTML as response error', async () => {
  const request = createAPIClient('/api', async () => new Response('<html>SPA fallback</html>'));
  await assert.rejects(request('/users/id'), e => e.kind === 'response');
});

test('normalizes network errors without retrying mutations', async () => {
  let calls = 0;
  const request = createAPIClient('/api', async () => { calls++; throw new TypeError('Failed to fetch'); });
  await assert.rejects(request('/queues/', { method: 'POST' }), e => e.status === 0 && e.kind === 'network');
  assert.equal(calls, 1);
});

test('passes AbortSignal and preserves cancellations', async () => {
  const controller = new AbortController();
  controller.abort();
  const request = createAPIClient('/api', async (_, init) => { assert.equal(init.signal, controller.signal); throw controller.signal.reason; });
  await assert.rejects(request('/businesses/', { signal: controller.signal }), e => e.name === 'AbortError');
});

test('does not turn subscription business rejection into success or an HTTP error', async () => {
  const request = createAPIClient('/api', async () => Response.json({ success: false, message: 'no user priority slots available' }));
  assert.equal((await request('/subscriptions/users/id/use', { method: 'POST', body: { businessId: 'id' } })).success, false);
});



