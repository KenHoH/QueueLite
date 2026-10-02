import { after, before, test } from 'node:test';
import assert from 'node:assert/strict';
import { mkdir } from 'node:fs/promises';
import { createServer } from 'vite';
import { chromium } from 'playwright';

const business = { id: '11111111-1111-4111-8111-111111111111', name: 'Northside Barbers', location: 'Pluit, Jakarta', operational: true, openTime: '09:00', closeTime: '21:00', email: 'hello@example.test', phoneNumber: '6281234567890' };
const user = { id: '22222222-2222-4222-8222-222222222222', username: 'Ayu', phonenumber: '081234567890' };
const counter = { id: '33333333-3333-4333-8333-333333333333', businessId: business.id, name: 'Front desk' };
const ticket = { id: '44444444-4444-4444-8444-444444444444', businessId: business.id, userId: user.id, name: 'A024', state: 'waiting', priority: false };
const pause = ms => new Promise(resolve => setTimeout(resolve, ms));
const visible = locator => locator.waitFor({ state: 'visible' });
const absent = locator => locator.waitFor({ state: 'hidden' });
let vite, browser, origin;
before(async () => {
  vite = await createServer({ server: { host: '127.0.0.1', port: 0, open: false, ws: false } });
  await vite.listen(); origin = `http://127.0.0.1:${vite.httpServer.address().port}`;
  browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_CHANNEL ? { channel: process.env.PLAYWRIGHT_CHANNEL } : {}) });
  await mkdir('output/playwright', { recursive: true });
});
after(async () => { await browser?.close(); await vite?.close(); });
async function scenario(t, options = {}) {
  const page = await browser.newPage({ viewport: { width: 1280, height: 900 } });
  page.setDefaultTimeout(8000);
  const requests = [], errors = [];
  page.on('pageerror', error => errors.push(error.message));
  page.on('console', message => {
    if (message.type() === 'error' && !(message.text().startsWith('Failed to load resource:') && message.location().url.startsWith(`${origin}/api/`))) errors.push(message.text());
  });
  t.after(async () => { try { assert.deepEqual(errors, [], 'application errors'); } finally { await page.close(); } });
  await page.route(`${origin}/api/**`, async route => {
    const request = route.request(), path = new URL(request.url()).pathname;
    requests.push({ path, method: request.method(), body: request.postData(), headers: request.headers() });
    let response = await options.respond?.(path, request);
    if (response?.abort) return route.abort('failed');
    if (!response) {
      if (path === '/api/users/me') response = options.user ? { json: options.user } : { status: 401, json: {} };
      else if (path === '/api/businesses/mine') response = { json: [] };
      else if (path === '/api/queues/me') response = { json: options.queues ?? [ticket] };
      else if (path.endsWith('/join')) response = { status: 201, json: ticket, headers: { 'set-cookie': 'queueToken_test=ticket-cookie; Path=/; HttpOnly; SameSite=Lax' } };
      else if (path.startsWith('/api/businesses/')) response = { json: business };
      else if (path.startsWith('/api/counters/')) response = { json: counter };
      else if (path.endsWith('/state')) response = { json: { state: options.state ?? 'waiting' } };
      else if (path.startsWith('/api/queues/')) response = { json: { ...ticket, ...(options.state ? { state: options.state } : {}), ...(options.called ? { calledByCounterId: counter.id } : {}) } };
      else throw new Error(`Unexpected fixture request ${request.method()} ${path}`);
    }
    try { await route.fulfill(response); }
    catch (error) { if (!/closed|cancel|abort|Invalid Interception/i.test(error.message)) throw error; }
  });
  return { page, requests };
}
test('account join confirms identity, sends no body, prevents duplicate submits and shows backend number', async t => {
  const { page, requests } = await scenario(t, { user, respond: async path => { if (path.endsWith('/join')) { await pause(200); return { status: 201, json: ticket }; } } });
  await page.goto(`${origin}/business/${business.id}/join`);
  await visible(page.getByText('Joining with your account'));
  await visible(page.getByText(user.phonenumber, { exact: true }));
  await absent(page.getByRole('textbox'));
  await page.getByRole('button', { name: 'Join queue' }).click();
  assert.equal(await page.getByRole('button', { name: 'Join queue' }).isDisabled(), true);
  await visible(page.getByRole('heading', { name: 'Your place is saved.' }));
  await visible(page.getByText('A024', { exact: true })); await visible(page.getByText('Waiting', { exact: true }));
  const joins = requests.filter(r => r.path.endsWith('/join'));
  assert.equal(joins.length, 1); assert.equal(joins[0].body, null);
  assert.equal(await page.getByRole('link', { name: 'View my queue' }).getAttribute('href'), `/queue/${ticket.id}`);
  await page.screenshot({ path: 'output/playwright/customer-success-desktop.png', fullPage: true });
});
test('guest validation, trimmed identity-only body, success and immediate cookie-owned ticket read', async t => {
  const { page, requests } = await scenario(t);
  await page.goto(`${origin}/business/${business.id}/join`);
  const submit = page.getByRole('button', { name: 'Join queue' }); await visible(submit);
  await submit.click(); await visible(page.getByText('Enter your name.', { exact: true }));
  assert.equal(requests.filter(r => r.method === 'POST').length, 0);
  await page.getByRole('textbox', { name: 'Name', exact: true }).fill(' Guest ');
  await page.getByRole('textbox', { name: 'Phone number', exact: true }).fill('123');
  await submit.click(); await visible(page.getByText(/Enter an Indonesian mobile number/));
  await page.getByRole('textbox', { name: 'Phone number', exact: true }).fill('0812 3456 7890');
  await page.screenshot({ path: 'output/playwright/customer-join-desktop.png', fullPage: true });
  await submit.click(); await visible(page.getByRole('heading', { name: 'Your place is saved.' }));
  assert.deepEqual(JSON.parse(requests.find(r => r.method === 'POST').body), { username: 'Guest', phoneNumber: '0812 3456 7890' });
  await page.getByRole('link', { name: 'View my queue' }).click();
  await visible(page.getByRole('heading', { name: 'You’re in the queue.' }));
  const read = requests.find(r => r.path === `/api/queues/${ticket.id}`);
  assert.match(read.headers.cookie, /queueToken_test=ticket-cookie/);
  assert.equal(requests.filter(r => r.path === '/api/queues/me').length, 0);
});
test('business missing or unavailable blocks joining before rendering inputs', async t => {
  let missing = true;
  const { page, requests } = await scenario(t, { respond: path => path.startsWith('/api/businesses/') ? missing ? { status: 404, json: { error: { code: 'BUSINESS_NOT_FOUND', message: 'raw' } } } : { json: { ...business, operational: false } } : undefined });
  await page.goto(`${origin}/business/${business.id}/join`); await visible(page.getByRole('heading', { name: 'Business not found.' }));
  await absent(page.getByRole('textbox')); missing = false;
  await page.reload(); await visible(page.getByText('This business is currently unavailable. Joining is paused.'));
  await absent(page.getByRole('button', { name: 'Join queue' }));
  assert.equal(requests.filter(r => r.method === 'POST').length, 0);
});
for (const [code, copy] of Object.entries({ BUSINESS_NOT_FOUND: 'This business could not be found.', BUSINESS_UNAVAILABLE: 'This business is currently unavailable. Please try another time.', ACTIVE_QUEUE_EXISTS: 'You already have an active queue at this business.', BUSINESS_QUEUE_FULL: 'This business has reached its queue limit. Please try another time.', PHONE_ALREADY_REGISTERED: 'This phone number already has an active queue at this business.', INVALID_PHONE_NUMBER: 'Enter a valid Indonesian mobile number.', USERNAME_REQUIRED: 'Enter your name.', PHONE_NUMBER_REQUIRED: 'Enter your phone number.', INVALID_FORMAT: 'Please check your details and try again.' })) {
  test(`join maps ${code} without exposing backend message`, async t => {
    const { page } = await scenario(t, { user, respond: path => path.endsWith('/join') ? { status: 409, json: { error: { code, message: 'raw internal backend message' } } } : undefined });
    await page.goto(`${origin}/business/${business.id}/join`); await visible(page.getByText('Joining with your account'));
    await page.getByRole('button', { name: 'Join queue' }).click(); await visible(page.getByText(copy, { exact: true }));
    await absent(page.getByText('raw internal backend message'));
  });
}
test('ambiguous join failure is never retried automatically or by repeated submit', async t => {
  const { page, requests } = await scenario(t, { user, respond: path => path.endsWith('/join') ? { abort: true } : undefined });
  await page.clock.install(); await page.goto(`${origin}/business/${business.id}/join`); await visible(page.getByText('Joining with your account'));
  await page.getByRole('button', { name: 'Join queue' }).click();
  await visible(page.getByText(/We couldn’t confirm whether your place was saved/));
  await page.clock.runFor(15000);
  assert.equal(requests.filter(r => r.method === 'POST').length, 1);
  assert.equal(await page.getByRole('button', { name: 'Join queue' }).isDisabled(), true);
  await visible(page.getByRole('link', { name: 'Check My Queues' }));
});
const states = { waiting: 'You’re in the queue.', called: 'It’s your turn.', processing: 'You’re being served.', done: 'Your visit is complete.', skipped: 'Your turn was skipped.', cancelled: 'Your queue is cancelled.' };
for (const [state, heading] of Object.entries(states)) {
  test(`ticket renders ${state} and ${['done', 'skipped', 'cancelled'].includes(state) ? 'stops' : 'continues'} polling`, async t => {
    const { page, requests } = await scenario(t, { state, called: state === 'called' || state === 'processing' });
    await page.clock.install(); await page.goto(`${origin}/queue/${ticket.id}`);
    await visible(page.getByRole('heading', { name: heading })); await visible(page.getByText(business.name, { exact: true }));
    if (state === 'called' || state === 'processing') await visible(page.getByText('Counter: Front desk', { exact: true }));
    const before = requests.filter(r => r.path.endsWith('/state')).length;
    const nextPoll = ['done', 'skipped', 'cancelled'].includes(state) ? null : page.waitForResponse(response => new URL(response.url()).pathname.endsWith('/state'));
    await page.clock.runFor(5500);
    if (['done', 'skipped', 'cancelled'].includes(state)) {
      assert.equal(requests.filter(r => r.path.endsWith('/state')).length, before);
      await absent(page.getByRole('button', { name: 'Leave queue' }));
    } else { await nextPoll; assert.ok(requests.filter(r => r.path.endsWith('/state')).length > before); }
    if (state === 'called') await page.screenshot({ path: 'output/playwright/customer-called-desktop.png', fullPage: true });
  });
}
test('poll discovers terminal transition and unmount cancels future requests', async t => {
  let state = 'waiting';
  const { page, requests } = await scenario(t, { respond: path => path.endsWith('/state') ? { json: { state } } : undefined });
  await page.clock.install(); await page.goto(`${origin}/queue/${ticket.id}`); await visible(page.getByRole('heading', { name: states.waiting }));
  state = 'done'; await page.clock.runFor(5500); await visible(page.getByRole('heading', { name: states.done }));
  const count = requests.filter(r => r.path.endsWith('/state')).length;
  await page.clock.runFor(11000); assert.equal(requests.filter(r => r.path.endsWith('/state')).length, count);
  state = 'waiting'; await page.reload(); await visible(page.getByRole('heading', { name: states.waiting }));
  await page.getByRole('link', { name: 'View business details' }).click(); await visible(page.getByRole('heading', { name: 'Business information' }));
  const left = requests.filter(r => r.path.endsWith('/state')).length;
  await page.clock.runFor(11000); assert.equal(requests.filter(r => r.path.endsWith('/state')).length, left);
});
for (const status of [401, 403, 404]) test(`ticket handles ownership/missing error ${status} and does not poll`, async t => {
  const { page, requests } = await scenario(t, { respond: path => path.startsWith('/api/queues/') ? { status, json: { error: { code: status === 404 ? 'QUEUE_NOT_FOUND' : 'QUEUE_ACCESS_DENIED', message: 'raw' } } } : undefined });
  await page.clock.install(); await page.goto(`${origin}/queue/${ticket.id}`);
  await visible(page.getByText(status === 404 ? 'This queue ticket could not be found.' : /This ticket isn’t available in this session/));
  const count = requests.filter(r => r.path.startsWith('/api/queues/')).length;
  await page.clock.runFor(11000); assert.equal(requests.filter(r => r.path.startsWith('/api/queues/')).length, count);
});
test('cancellation requires confirmation, preserves ticket and handles persistence-pending without retry', async t => {
  let pending = true;
  const { page, requests } = await scenario(t, { user, respond: (path, request) => request.method() === 'PATCH' ? pending ? { status: 409, json: { error: { code: 'QUEUE_PERSISTENCE_PENDING', message: 'raw' } } } : { json: { message: 'queue state updated' } } : undefined });
  await page.clock.install(); await page.goto(`${origin}/queue/${ticket.id}`); await visible(page.getByRole('heading', { name: states.waiting }));
  await page.getByRole('button', { name: 'Leave queue' }).click(); await visible(page.getByRole('heading', { name: 'Leave this queue?' }));
  assert.equal(requests.filter(r => r.method === 'PATCH').length, 0);
  await page.getByRole('button', { name: 'Keep my place' }).click(); await page.getByRole('button', { name: 'Leave queue' }).click();
  await page.getByRole('button', { name: 'Confirm leave' }).click(); await visible(page.getByText(/Your ticket is still being saved/));
  await page.clock.runFor(11000); assert.equal(requests.filter(r => r.method === 'PATCH').length, 1);
  await visible(page.getByRole('heading', { name: states.waiting }));
  pending = false; await page.getByRole('button', { name: 'Confirm leave' }).click(); await visible(page.getByRole('heading', { name: states.cancelled }));
  const mutations = requests.filter(r => r.method === 'PATCH');
  assert.ok(mutations.every(r => r.path.endsWith('/state') && r.body === '{"state":"cancelled"}'));
  assert.equal(requests.filter(r => r.method === 'DELETE').length, 0);
  const count = requests.length; await page.clock.runFor(11000); assert.equal(requests.length, count);
});
test('My Queues lists account tickets, deduplicates business reads and navigates to ticket', async t => {
  const another = { ...ticket, id: '55555555-5555-4555-8555-555555555555', name: 'A025', state: 'processing' };
  const { page, requests } = await scenario(t, { user, queues: [ticket, another] });
  await page.goto(`${origin}/my-queues`); await visible(page.getByText('A025', { exact: true }));
  assert.equal(requests.filter(r => r.path === `/api/businesses/${business.id}`).length, 1);
  assert.equal(await page.getByRole('link', { name: 'My Queues', exact: true }).getAttribute('aria-current'), 'page');
  await page.screenshot({ path: 'output/playwright/customer-my-queues-desktop.png', fullPage: true });
  await page.locator('.ql-queue-card').first().click(); await visible(page.getByRole('heading', { name: states.waiting }));
  assert.equal(new URL(page.url()).pathname, `/queue/${ticket.id}`);
});
test('My Queues empty and error/retry states', async t => {
  let failed = true;
  const { page } = await scenario(t, { user, respond: path => path === '/api/queues/me' ? failed ? { status: 500, json: {} } : { json: [] } : undefined });
  await page.goto(`${origin}/my-queues`); await visible(page.getByText('We couldn’t load your queues.'));
  failed = false; await page.getByRole('button', { name: 'Try again' }).click(); await visible(page.getByRole('heading', { name: 'You don’t have any active queues.' }));
  await visible(page.getByRole('button', { name: 'Refresh queues' }));
});
test('guest My Queues offers sign-in without a list request or account navigation', async t => {
  const { page, requests } = await scenario(t);
  await page.goto(`${origin}/my-queues`); await visible(page.getByRole('heading', { name: 'Sign in to view your queues.' }));
  assert.equal(requests.filter(r => r.path === '/api/queues/me').length, 0);
  await absent(page.getByRole('link', { name: 'My Queues', exact: true }));
  await visible(page.getByRole('link', { name: 'Businesses', exact: true }));
});
test('customer pages stay within mobile viewport and use the existing design system', async t => {
  const { page } = await scenario(t);
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto(`${origin}/business/${business.id}/join`); await visible(page.getByRole('textbox', { name: 'Name', exact: true }));
  assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
  await page.screenshot({ path: 'output/playwright/customer-join-mobile.png', fullPage: true });
  await page.goto(`${origin}/queue/${ticket.id}`); await visible(page.getByRole('heading', { name: states.waiting }));
  assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
  await page.screenshot({ path: 'output/playwright/customer-ticket-mobile.png', fullPage: true });
});
test('join waits for the real session and business before exposing a form', async t => {
  const { page, requests } = await scenario(t, { respond: async path => {
    if (path === '/api/users/me') { await pause(450); return { json: user }; }
    if (path === `/api/businesses/${business.id}`) { await pause(200); return { json: business }; }
  } });
  await page.goto(`${origin}/business/${business.id}/join`);
  await visible(page.getByText('Loading business', { exact: true }));
  await absent(page.getByRole('textbox'));
  await visible(page.getByText('Joining with your account'));
  assert.equal(requests.filter(r => r.method === 'POST').length, 0);
});
test('My Queues exposes loading and business-name failures do not hide tickets', async t => {
  const { page } = await scenario(t, { user, respond: async path => {
    if (path === '/api/queues/me') { await pause(250); return { json: [ticket] }; }
    if (path.startsWith('/api/businesses/')) return { status: 500, json: {} };
  } });
  await page.goto(`${origin}/my-queues`); await visible(page.getByText('Loading your queues', { exact: true }));
  await visible(page.getByRole('heading', { name: 'Business details unavailable' }));
  await visible(page.getByText('A024', { exact: true }));
});
test('expired account list session is rechecked and allows signing in cleanly', async t => {
  let expired = false;
  const { page, requests } = await scenario(t, { respond: path => {
    if (path === '/api/users/me') return expired ? { status: 401, json: {} } : { json: user };
    if (path === '/api/queues/me') { expired = true; return { status: 401, json: {} }; }
  } });
  await page.goto(`${origin}/my-queues`); await visible(page.getByRole('heading', { name: 'Sign in to view your queues.' }));
  assert.ok(requests.some(r => r.path === '/api/queues/me'));
  await page.locator('main').getByRole('link', { name: 'Sign in', exact: true }).click();
  await visible(page.getByRole('textbox', { name: 'Username', exact: true }));
});
test('ticket reads pause in hidden tabs and resume on visibility', async t => {
  const { page, requests } = await scenario(t);
  await page.clock.install(); await page.goto(`${origin}/queue/${ticket.id}`);
  await visible(page.getByRole('heading', { name: states.waiting }));
  await page.evaluate(() => {
    window.testHidden = true;
    Object.defineProperty(document, 'hidden', { configurable: true, get: () => window.testHidden });
    document.dispatchEvent(new Event('visibilitychange'));
  });
  const before = requests.filter(r => r.path.endsWith('/state')).length;
  await page.clock.runFor(11000); assert.equal(requests.filter(r => r.path.endsWith('/state')).length, before);
  const resumed = page.waitForResponse(response => new URL(response.url()).pathname.endsWith('/state'));
  await page.evaluate(() => { window.testHidden = false; document.dispatchEvent(new Event('visibilitychange')); });
  await resumed; assert.ok(requests.filter(r => r.path.endsWith('/state')).length > before);
});
