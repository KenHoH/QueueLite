import { before, after, test } from 'node:test';
import assert from 'node:assert/strict';
import { mkdir } from 'node:fs/promises';
import { createServer } from 'vite';
import { chromium } from 'playwright';
const user = { id: '22222222-2222-4222-8222-222222222222', username: 'Staff', phonenumber: '081234567890' };
const business = { id: '11111111-1111-4111-8111-111111111111', name: 'Northside Barbers', location: 'Jakarta', operational: true, openTime: '09:00', closeTime: '17:00', email: 'desk@example.test', phoneNumber: '081234567890', role: 'owner' };
const other = { ...business, id: '33333333-3333-4333-8333-333333333333', name: 'GreenCare', role: 'admin' };
const counter = { id: '44444444-4444-4444-8444-444444444444', businessId: business.id, name: 'Desk A', currentEmployeeId: user.id };
const queue = { id: '55555555-5555-4555-8555-555555555555', businessId: business.id, name: 'Q001', state: 'waiting', priority: true };
const visible = locator => locator.waitFor({ state: 'visible' });
const pause = ms => new Promise(resolve => setTimeout(resolve, ms));
let vite, browser, origin;
before(async () => {
  vite = await createServer({ server: { host: '127.0.0.1', port: 0, open: false, ws: false } }); await vite.listen(); origin = `http://127.0.0.1:${vite.httpServer.address().port}`;
  browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_CHANNEL ? { channel: process.env.PLAYWRIGHT_CHANNEL } : {}) });
  await mkdir('output/playwright', { recursive: true });
});
after(async () => { await browser?.close(); await vite?.close(); });
async function scenario(t, options = {}) {
  const page = await browser.newPage({ viewport: { width: 1280, height: 900 } }); page.setDefaultTimeout(8000);
  const counters = structuredClone(options.counters ?? [counter]), queues = structuredClone(options.queues ?? [queue]), requests = [], errors = [];
  const memberships = options.businesses ?? [business, other];
  page.on('pageerror', error => errors.push(error.message));
  t.after(async () => { try { assert.deepEqual(errors, []); } finally { await page.close(); } });
  await page.route(`${origin}/api/**`, async route => {
    const request = route.request(), path = new URL(request.url()).pathname, method = request.method(); requests.push({ path, method, body: request.postDataJSON() });
    let response = await options.respond?.(path, request);
    if (!response) {
      if (path === '/api/users/me') response = options.guest ? { status: 401, json: {} } : { json: user };
      else if (path === '/api/businesses/mine') response = { json: memberships };
      else if (/\/businesses\/[^/]+\/counters$/.test(path)) response = { json: counters.filter(c => c.businessId === path.split('/')[3]) };
      else if (/\/businesses\/[^/]+\/queues$/.test(path)) response = { json: queues.filter(q => q.businessId === path.split('/')[3]) };
      else if (path.endsWith('/members')) response = { json: [{ userId: user.id, username: user.username, role: 'counter' }, { userId: '66666666-6666-4666-8666-666666666666', username: 'Ayu', role: 'admin' }] };
      else if (path.startsWith('/api/subscriptions/businesses/')) response = { json: { subscription: { type: 'business', status: 'active', startDate: '2026-09-01T00:00:00Z' }, businessPlan: { capacity: 43, businessPlanType: 'free' } } };
      else if (path.startsWith('/api/subscriptions/users/')) response = { json: { subscription: { type: 'user', status: 'active', startDate: '2026-09-01T00:00:00Z' }, userPlan: { userPlanType: 'standard', slots: 0 } } };
      else if (path.startsWith('/api/businesses/')) response = { json: memberships.find(b => path.endsWith(b.id)) ?? business };
      else if (path === '/api/counters/' && method === 'POST') { const body = request.postDataJSON(), created = { ...body, id: '77777777-7777-4777-8777-777777777777' }; counters.push(created); response = { status: 201, json: created }; }
      else if (path.startsWith('/api/counters/')) {
        const c = counters.find(c => c.id === path.split('/')[3]);
        if (path.endsWith('/call-next')) { const q = queues.find(q => q.state === 'waiting'); if (q) { q.state = 'called'; q.calledByCounterId = c.id; c.currentQueueId = q.id; response = { json: { queueId: q.id, queueName: q.name } }; } else response = { json: { queue: null } }; }
        else if (path.endsWith('/process')) { queues.find(q => q.id === c.currentQueueId).state = 'processing'; response = { json: { queueId: c.currentQueueId, state: 'processing' } }; }
        else if (path.endsWith('/skip') || (path.includes('/queues/') && method === 'DELETE')) { const q = queues.find(q => q.id === c.currentQueueId); q.state = path.endsWith('/skip') ? 'skipped' : 'done'; delete c.currentQueueId; response = { json: path.endsWith('/skip') ? { queue: null } : { message: 'queue removed' } }; }
        else if (method === 'PUT') { const body = request.postDataJSON(); Object.assign(c, body); if (!body.currentEmployeeId) delete c.currentEmployeeId; response = { json: { message: 'counter updated' } }; }
        else if (method === 'DELETE') { counters.splice(counters.indexOf(c), 1); response = { json: { message: 'counter deleted' } }; }
        else response = c ? { json: c } : { status: 404, json: {} };
      } else throw new Error(`Unexpected ${method} ${path}`);
    }
    try { await route.fulfill(response); } catch (error) { if (!/closed|cancel|abort|Invalid Interception/i.test(error.message)) throw error; }
  });
  return { page, requests, counters, queues };
}
test('dashboard loads real groups, counters, capacity and explicit navigation', async t => {
  const queues = [queue, { ...queue, id: 'q2', name: 'Q002', state: 'called', priority: false, calledByCounterId: counter.id }, { ...queue, id: 'q3', name: 'Q003', state: 'processing', priority: false }, { ...queue, id: 'q4', name: 'Q004', state: 'done' }];
  const { page, requests } = await scenario(t, { queues, counters: [{ ...counter, currentQueueId: 'q2' }] }); await page.goto(`${origin}/business/${business.id}/dashboard`);
  for (const [state, name] of [['waiting', 'Q001'], ['called', 'Q002'], ['processing', 'Q003']]) await visible(page.getByRole('region', { name: `${state} queues` }).getByText(name, { exact: true }));
  assert.equal(await page.getByText('Q004', { exact: true }).count(), 0); await visible(page.getByText('Queue capacity remaining: 43')); await visible(page.getByRole('link', { name: 'Open workspace →' }));
  assert.equal(await page.getByRole('link', { name: 'Counters', exact: true }).getAttribute('href'), `/business/${business.id}/counters`);
  assert.ok(requests.some(r => r.path === `/api/businesses/${business.id}/queues`));
  await page.screenshot({ path: 'output/playwright/operations-dashboard-desktop.png', fullPage: true });
});
test('dashboard empty and error/retry states are honest', async t => {
  let fail = true; const { page } = await scenario(t, { counters: [], queues: [], respond: path => path.endsWith('/queues') && fail ? { status: 500, json: {} } : undefined });
  await page.goto(`${origin}/business/${business.id}/dashboard`); await visible(page.getByText('We couldn’t confirm the operation. Refresh state before trying again.')); fail = false;
  await page.getByRole('button', { name: 'Refresh state' }).click(); await visible(page.getByText('No waiting queues.')); await visible(page.getByRole('heading', { name: 'No counters available' }));
});
for (const route of ['dashboard', 'counters']) test(`business ${route} denies unrelated membership`, async t => {
  const { page, requests } = await scenario(t, { businesses: [other] }); await page.goto(`${origin}/business/${business.id}/${route}`); await visible(page.getByRole('heading', { name: 'Permission required' }));
  assert.ok(!requests.some(r => r.path.includes(`/${business.id}/`)));
});
test('revoked server authorization removes dashboard operational data', async t => {
  const { page } = await scenario(t, { respond: path => path.endsWith('/queues') ? { status: 403, json: {} } : undefined }); await page.goto(`${origin}/business/${business.id}/dashboard`);
  await visible(page.getByRole('heading', { name: 'Permission required' })); assert.equal(await page.getByRole('link', { name: 'Open workspace →' }).count(), 0);
});
test('management creates, renames, assigns, clears assignment and confirms deletion', async t => {
  const { page, requests, counters } = await scenario(t); await page.goto(`${origin}/business/${business.id}/counters`);
  await page.getByLabel('Counter name', { exact: true }).fill('Desk B'); await page.getByRole('button', { name: 'Create counter', exact: true }).click(); await visible(page.getByRole('heading', { name: 'Desk B', exact: true }));
  const card = page.locator('.ql-card').filter({ has: page.getByRole('heading', { name: 'Desk B', exact: true }) });
  await card.getByLabel('Name for Desk B').fill('Desk C'); await card.getByLabel('Employee for Desk B').selectOption(user.id); await card.getByRole('button', { name: 'Save counter' }).click(); await visible(page.getByRole('heading', { name: 'Desk C', exact: true }));
  assert.equal(counters.find(c => c.name === 'Desk C').currentEmployeeId, user.id);
  const renamed = page.locator('.ql-card').filter({ has: page.getByRole('heading', { name: 'Desk C', exact: true }) });
  await renamed.getByLabel('Employee for Desk C').selectOption(''); await renamed.getByRole('button', { name: 'Save counter' }).click(); await visible(renamed.getByText('Assigned employee: Unassigned'));
  await renamed.getByRole('button', { name: 'Delete counter', exact: true }).click(); await renamed.getByRole('button', { name: 'Confirm delete' }).click(); await page.getByRole('heading', { name: 'Desk C', exact: true }).waitFor({ state: 'hidden' });
  assert.ok(requests.find(r => r.method === 'POST').body.businessId === business.id);
  assert.ok(requests.filter(r => r.method !== 'GET').every(r => !('userId' in (r.body ?? {}))));
  await page.screenshot({ path: 'output/playwright/operations-counters-desktop.png', fullPage: true });
});
test('staff see assigned desks without management or member/subscription reads', async t => {
  const { page, requests } = await scenario(t, { businesses: [{ ...business, role: 'counter' }] }); await page.goto(`${origin}/business/${business.id}/counters`); await visible(page.getByRole('heading', { name: 'Desk A', exact: true }));
  assert.equal(await page.getByRole('button', { name: /Create counter|Save counter|Delete counter/ }).count(), 0); assert.ok(requests.every(r => !r.path.endsWith('/members')));
  await page.getByRole('link', { name: 'Dashboard', exact: true }).click(); await visible(page.getByRole('heading', { name: business.name })); assert.ok(requests.every(r => !r.path.startsWith('/api/subscriptions/')));
});
test('workspace calls next, shows priority, starts service, completes and refreshes', async t => {
  const { page, requests, queues, counters } = await scenario(t); await page.goto(`${origin}/counter/${counter.id}`); await page.getByRole('button', { name: 'Call next', exact: true }).click();
  await visible(page.getByText('Q001', { exact: true })); await visible(page.getByText('Priority', { exact: true })); await page.getByRole('button', { name: 'Start service' }).click(); await visible(page.getByRole('button', { name: 'Complete', exact: true }));
  await page.getByRole('button', { name: 'Complete', exact: true }).click(); await visible(page.getByRole('button', { name: 'Call next', exact: true })); assert.equal(queues[0].state, 'done'); assert.equal(counters[0].currentQueueId, undefined);
  assert.ok(requests.filter(r => r.method === 'GET' && r.path === `/api/counters/${counter.id}`).length >= 4);
});
test('workspace handles no waiting customers and skip semantics', async t => {
  const { page, queues } = await scenario(t, { counters: [{ ...counter, currentQueueId: queue.id }], queues: [{ ...queue, state: 'called', calledByCounterId: counter.id }] }); await page.goto(`${origin}/counter/${counter.id}`);
  await page.getByRole('button', { name: 'Skip', exact: true }).click(); await visible(page.getByRole('button', { name: 'Call next', exact: true })); assert.equal(queues[0].state, 'skipped');
  await page.getByRole('button', { name: 'Call next', exact: true }).click(); await visible(page.getByText('No waiting customer available.'));
});
test('workspace prevents duplicate submissions while an action is pending', async t => {
  let release; const held = new Promise(resolve => { release = resolve; });
  const { page, requests } = await scenario(t, { respond: async (path, request) => { if (path.endsWith('/call-next') && request.method() === 'POST') await held; } }); await page.goto(`${origin}/counter/${counter.id}`);
  const call = page.getByRole('button', { name: 'Call next', exact: true }); await call.click(); assert.equal(await call.isDisabled(), true); await call.evaluate(button => { button.click(); button.click(); }); await pause(50);
  assert.equal(requests.filter(r => r.method === 'POST').length, 1); release(); await visible(page.getByRole('button', { name: 'Start service' }));
});
for (const [status, code, message] of [[409, 'COUNTER_STATE_CHANGED', 'Counter state changed. Refresh state before trying again.'], [404, 'QUEUE_NOT_READY', 'Queue persistence is pending. Refresh state before trying again.'], [500, 'INTERNAL', 'We couldn’t confirm the operation. Refresh state before trying again.']]) test(`workspace handles ${code} without automatic mutation retry`, async t => {
  let fail = true; const { page, requests } = await scenario(t, { respond: (path, request) => path.endsWith('/call-next') && request.method() === 'POST' && fail ? { status, json: { error: { code, message: 'private details' } } } : undefined }); await page.goto(`${origin}/counter/${counter.id}`); await page.getByRole('button', { name: 'Call next', exact: true }).click(); await visible(page.getByText(message));
  await visible(page.getByRole('button', { name: 'Call next', exact: true })); assert.equal(await page.getByRole('button', { name: 'Call next', exact: true }).isDisabled(), true); assert.equal(requests.filter(r => r.method === 'POST').length, 1);
  fail = false; await page.getByRole('button', { name: 'Refresh state', exact: true }).click(); await page.getByRole('button', { name: 'Call next', exact: true }).click(); await visible(page.getByRole('button', { name: 'Start service' }));
});
test('workspace waits for missing assigned queue details', async t => {
  const { page } = await scenario(t, { counters: [{ ...counter, currentQueueId: queue.id }], queues: [] }); await page.goto(`${origin}/counter/${counter.id}`); await visible(page.getByRole('heading', { name: 'Customer state pending' })); assert.equal(await page.getByRole('button', { name: /Call next|Complete|Start service|Skip/ }).count(), 0);
});
test('dashboard loading, visible polling, hidden pause and unmount cleanup', async t => {
  let release; const held = new Promise(resolve => { release = resolve; }); let initial = true;
  const { page, requests } = await scenario(t, { respond: async path => { if (path.endsWith('/queues') && initial) await held; } });
  await page.clock.install(); await page.goto(`${origin}/business/${business.id}/dashboard`); await visible(page.getByText('Loading business operations', { exact: true }));
  initial = false; release(); await visible(page.getByRole('region', { name: 'waiting queues' }).getByText('Q001', { exact: true }));
  const count = () => requests.filter(r => r.path.endsWith('/queues')).length;
  const before = count(); await page.clock.fastForward(10001); await pause(100); assert.equal(count(), before + 1);
  await page.evaluate(() => { Object.defineProperty(document, 'visibilityState', { configurable: true, get: () => 'hidden' }); document.dispatchEvent(new Event('visibilitychange')); });
  const hidden = count(); await page.clock.fastForward(20001); await pause(100); assert.equal(count(), hidden);
  await page.evaluate(() => { Object.defineProperty(document, 'visibilityState', { configurable: true, get: () => 'visible' }); document.dispatchEvent(new Event('visibilitychange')); }); await pause(100); assert.equal(count(), hidden + 1);
  await page.getByRole('link', { name: 'Your businesses', exact: false }).click(); await visible(page.getByRole('heading', { name: 'Manage business', exact: true })); const gone = count(); await page.clock.fastForward(20001); await pause(100); assert.equal(count(), gone);
});
test('counter management validates names and prevents duplicate create submissions', async t => {
  let release; const held = new Promise(resolve => { release = resolve; });
  const { page, requests } = await scenario(t, { respond: async (path, request) => { if (path === '/api/counters/' && request.method() === 'POST') await held; } }); await page.goto(`${origin}/business/${business.id}/counters`);
  await page.getByRole('button', { name: 'Create counter', exact: true }).click(); await visible(page.getByText('Enter a counter name.')); assert.equal(requests.filter(r => r.method === 'POST').length, 0);
  await page.getByLabel('Counter name', { exact: true }).fill('Desk B'); await page.getByRole('button', { name: 'Create counter', exact: true }).click(); assert.equal(await page.getByRole('button', { name: 'Create counter', exact: true }).isDisabled(), true);
  await page.locator('form').first().evaluate(form => { form.requestSubmit(); form.requestSubmit(); }); await pause(50); assert.equal(requests.filter(r => r.method === 'POST').length, 1); release(); await visible(page.getByRole('heading', { name: 'Desk B', exact: true }));
});
test('counter management handles revoked mutation authorization and requires refresh', async t => {
  const { page, requests } = await scenario(t, { respond: (path, request) => request.method() === 'PUT' ? { status: 403, json: {} } : undefined }); await page.goto(`${origin}/business/${business.id}/counters`);
  await page.getByRole('button', { name: 'Save counter' }).click(); await visible(page.getByText('You do not have permission to access this business or counter.')); assert.equal(await page.getByRole('button', { name: 'Save counter' }).isDisabled(), true);
  await page.getByRole('button', { name: 'Refresh state' }).click(); await visible(page.getByRole('button', { name: 'Save counter' })); assert.equal(requests.filter(r => r.method === 'PUT').length, 1);
});
test('workspace uses protected counter reads and fails closed on 403', async t => {
  const { page, requests } = await scenario(t, { respond: path => path === `/api/counters/${counter.id}` ? { status: 403, json: {} } : undefined }); await page.goto(`${origin}/counter/${counter.id}`); await visible(page.getByRole('heading', { name: 'Permission required' })); assert.equal(await page.getByRole('button', { name: 'Call next', exact: true }).count(), 0); assert.ok(!requests.some(r => r.path.endsWith('/queues')));
});
test('business links preserve multi-business selection and plans use explicit IDs', async t => {
  const { page, requests } = await scenario(t); await page.goto(`${origin}/business/manage`); await page.getByLabel('Select business').selectOption(other.id); await page.getByRole('link', { name: 'Dashboard', exact: true }).click();
  await visible(page.getByRole('heading', { name: other.name })); assert.match(page.url(), new RegExp(`${other.id}/dashboard$`)); await page.getByRole('link', { name: 'Plans', exact: true }).click(); await visible(page.getByText('Queue capacity remaining: 43'));
  assert.match(page.url(), new RegExp(`${other.id}/plans$`)); assert.ok(requests.some(r => r.path === `/api/subscriptions/businesses/${other.id}`));
});
for (const path of [`/business/${business.id}/dashboard`, `/business/${business.id}/counters`, `/counter/${counter.id}`]) test(`guest sees sign-in on ${path}`, async t => {
  const { page, requests } = await scenario(t, { guest: true }); await page.goto(origin + path); await visible(page.getByRole('heading', { name: 'Sign in to continue.' })); assert.ok(requests.every(r => r.path === '/api/users/me'));
});
test('staff screens fit mobile and expose clear workspace actions', async t => {
  const { page } = await scenario(t, { counters: [{ ...counter, currentQueueId: queue.id }], queues: [{ ...queue, state: 'called', calledByCounterId: counter.id }] }); await page.setViewportSize({ width: 390, height: 844 });
  for (const [path, screenshot] of [[`/business/${business.id}/dashboard`, 'dashboard'], [`/business/${business.id}/counters`, 'counters'], [`/counter/${counter.id}`, 'workspace']]) {
    await page.goto(origin + path); await visible(page.getByRole('heading', { name: path.includes('/counter/') ? 'Desk A' : business.name, exact: true }).or(page.getByRole('heading', { name: 'Create counter', exact: true })).first()); await pause(100);
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth), true); await page.screenshot({ path: `output/playwright/operations-${screenshot}-mobile.png`, fullPage: true });
  }
});
