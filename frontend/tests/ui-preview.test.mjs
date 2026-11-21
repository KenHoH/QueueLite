import { test } from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';

test('UI preview serves isolated sample sessions and working guest/counter flows without an upstream', { timeout: 30000 }, async () => {
  const vite = await createServer({ mode: 'ui-preview', optimizeDeps: { noDiscovery: true, include: [] }, server: { host: '127.0.0.1', port: 0, ws: false } });
  try {
    await vite.listen();
    const origin = `http://127.0.0.1:${vite.httpServer.address().port}`;
    let cookie;
    const request = async (path, value, method = value ? 'POST' : 'GET') => {
      const res = await fetch(origin + path, { method, headers: { ...(cookie ? { Cookie: cookie } : {}), ...(value ? { 'Content-Type': 'application/json' } : {}) }, ...(value ? { body: JSON.stringify(value) } : {}) });
      cookie = res.headers.get('set-cookie')?.split(';')[0] ?? cookie;
      return { status: res.status, data: await res.json() };
    };
    const html = await fetch(origin).then(r => r.text());
    assert.match(html, /__ui-preview\/toolbar.js/);
    assert.equal(vite.config.server.proxy, undefined);
    const { data: config } = await request('/__ui-preview/pages');
    assert.ok(config.pages.length > 20);
    assert.equal((await request('/api/users/me')).data.username, 'Matthew');
    const { data: page } = await request('/__ui-preview/select', { page: 2 });
    assert.equal((await request('/api/users/me')).status, 401);
    // A separate browser gets independent owner data.
    assert.equal((await fetch(origin + '/api/users/me')).status, 200);
    const business = page.path.split('/')[2];
    const { data: q, status } = await request(`/api/queues/business/${business}/join`, { username: 'Preview guest', phoneNumber: '081234567890' });
    assert.equal(status, 201);
    const { data: snapshot } = await request(`/api/queues/${q.id}/customer-status`);
    assert.equal(snapshot.queue.id, q.id);
    assert.equal(snapshot.counters.length, 2);
    assert.equal((await request('/api/queues/guest/me')).data.length, 1);
    assert.equal((await request(`/api/queues/business/${business}/join`, {})).status, 409);
    await request('/__ui-preview/reset', {});
    const counterPath = config.pages.find(p => p[0] === 'Counter workspace')[1];
    const counter = (await request('/api' + counterPath.replace('/counter/', '/counters/'))).data;
    await request(`/api/counters/${counter.id}/queues/${counter.currentQueueId}`, undefined, 'DELETE');
    const next = (await request(`/api/counters/${counter.id}/business/${counter.businessId}/call-next`, {})).data;
    assert.ok(next.queueId);
    assert.equal((await request(`/api/counters/${counter.id}/queues/${next.queueId}/process`, {})).data.state, 'processing');
    assert.equal((await request('/api/unsupported')).status, 501);
  } finally { await vite.close(); }
});
