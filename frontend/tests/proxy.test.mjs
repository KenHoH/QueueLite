import { test } from 'node:test';
import assert from 'node:assert/strict';
import { createServer as createHTTPServer } from 'node:http';
import { createServer } from 'vite';

test('actual Vite config starts, serves deep links and forwards paths/cookies', { timeout: 20000 }, async () => {
  // Local HTTP fixture only: verifies transport; does not simulate application data.
  const upstream = createHTTPServer((req, res) => {
    res.setHeader('Content-Type', 'application/json');
    res.setHeader('Set-Cookie', 'queueToken=fixture; Path=/; HttpOnly; Max-Age=86400');
    res.end(JSON.stringify({ path: req.url, cookie: req.headers.cookie }));
  });
  await new Promise(resolve => upstream.listen(0, '127.0.0.1', resolve));
  const previousTarget = process.env.API_PROXY_TARGET;
  process.env.API_PROXY_TARGET = `http://127.0.0.1:${upstream.address().port}`;
  let vite;
  try {
    vite = await createServer({ optimizeDeps: { noDiscovery: true, include: [] }, server: { ws: false, port: 0, host: '127.0.0.1', open: false } });
    await vite.listen();
    const origin = `http://127.0.0.1:${vite.httpServer.address().port}`;
    const result = await fetch(`${origin}/api/queues/qr/test/resolve?test=1`, { signal: AbortSignal.timeout(5000), headers: { Cookie: 'token=fixture-user' } });
    assert.deepEqual(await result.json(), { path: '/queues/qr/test/resolve?test=1', cookie: 'token=fixture-user' });
    assert.match(result.headers.get('set-cookie'), /queueToken=fixture; Path=\/; HttpOnly/);
    for (const route of ['/', '/login', '/register', '/profile', '/plans', '/business/manage', '/business/11111111-1111-4111-8111-111111111111', '/business/11111111-1111-4111-8111-111111111111/join', '/business/create', '/business/11111111-1111-4111-8111-111111111111/settings', '/queue/example', '/counter/example']) {
      const response = await fetch(`${origin}${route}`);
      assert.equal(response.status, 200);
      assert.match(await response.text(), /src\/main.tsx/);
    }
    const module = await fetch(`${origin}/src/main.tsx`);
    assert.equal(module.status, 200);
    assert.match(await module.text(), /AppStateProvider/);
    const envResponse = await fetch(`${origin}/.env.example`);
    assert.equal(envResponse.status, 403);
  } finally {
    await vite?.close();
    upstream.closeAllConnections();
    await new Promise(resolve => upstream.close(resolve));
    if (previousTarget === undefined) delete process.env.API_PROXY_TARGET;
    else process.env.API_PROXY_TARGET = previousTarget;
  }
});




