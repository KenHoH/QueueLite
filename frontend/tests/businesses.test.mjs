import { after, test } from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';
const vite = await createServer({ configFile: false, optimizeDeps: { noDiscovery: true, include: [] }, server: { middlewareMode: true, ws: false }, appType: 'custom' });
after(() => vite.close());
const display = await vite.ssrLoadModule('/src/pages/businessDisplay.ts');

test('availability reflects backend truth; hours do not assume a timezone', () => {
  const business = { operational: true, openTime: '22:00', closeTime: '06:00' };
  assert.equal(display.businessStatus(business), 'Operational');
  assert.equal(display.businessStatus({ ...business, operational: false }), 'Unavailable');
  assert.equal(display.businessHours(business), '22:00 – 06:00');
  assert.equal(display.businessHours({}), 'Hours not provided');
});
test('greetings use the observed username and local hour; invalid IDs fail gracefully', () => {
  assert.equal(display.greeting('Ayu', 9), 'Good morning, Ayu.');
  assert.equal(display.greeting('Ayu', 15), 'Good afternoon, Ayu.');
  assert.equal(display.greeting('Ayu', 20), 'Good evening, Ayu.');
  assert.equal(display.isBusinessId('11111111-1111-4111-8111-111111111111'), true);
  for (const id of [undefined, '', 'bad/id', 'not-a-uuid']) assert.equal(display.isBusinessId(id), false);
});
test('business services send real paths, named cursor parameters, encoded search and cancellation', async () => {
  const original = globalThis.fetch;
  const calls = [];
  const signal = new AbortController().signal;
  globalThis.fetch = async (url, init) => { calls.push({ url, init }); return Response.json({ data: [], nextCursor: null }); };
  try {
    const api = await vite.ssrLoadModule('/src/api/businesses.ts');
    await api.getBusinesses({ limit: 12, cursorCreatedAt: '2026-09-30T09:00:00+07:00', cursorID: 'id' }, { signal });
    await api.searchBusinesses('A & B', { signal });
    await api.getBusiness('11111111-1111-4111-8111-111111111111', { signal });
    const url = new URL(calls[0].url, 'http://localhost');
    assert.equal(url.pathname, '/api/businesses/');
    assert.equal(url.searchParams.get('limit'), '12');
    assert.equal(url.searchParams.get('cursorCreatedAt'), '2026-09-30T09:00:00+07:00');
    assert.equal(url.searchParams.get('cursorID'), 'id');
    assert.equal(new URL(calls[1].url, 'http://localhost').searchParams.get('name'), 'A & B');
    assert.equal(calls[2].url, '/api/businesses/11111111-1111-4111-8111-111111111111');
    assert.ok(calls.every(call => call.init.signal === signal && call.init.credentials === 'include' && call.init.method === 'GET'));
  } finally { globalThis.fetch = original; }
});
