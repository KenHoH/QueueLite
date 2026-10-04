import { after, before, test } from 'node:test';
import assert from 'node:assert/strict';
import { mkdir } from 'node:fs/promises';
import { createServer } from 'vite';
import { chromium } from 'playwright';

// Test-only HTTP interception exercises the real pages/client in a real browser.
// No fixture data is included in the production application.
const first = { id: '11111111-1111-4111-8111-111111111111', name: 'Northside Barbers', location: 'Pluit, Jakarta', description: 'A calm place for an appointment.', operational: true, openTime: '09:00', closeTime: '21:00', email: 'hello@example.test', phoneNumber: '628123456789' };
const second = { ...first, id: '22222222-2222-4222-8222-222222222222', name: 'GreenCare', operational: false };
const cursor = { createdAt: '2026-09-30T09:00:00Z', id: first.id };
const user = { id: '33333333-3333-4333-8333-333333333333', username: 'Ayu', phonenumber: '08123456789' };
const pause = ms => new Promise(resolve => setTimeout(resolve, ms));
let vite, browser, origin;
before(async () => {
  vite = await createServer({ server: { host: '127.0.0.1', port: 0, open: false, ws: false } });
  await vite.listen();
  origin = `http://127.0.0.1:${vite.httpServer.address().port}`;
  browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_CHANNEL ? { channel: process.env.PLAYWRIGHT_CHANNEL } : {}) });
  await mkdir('output/playwright', { recursive: true });
});
after(async () => { await browser?.close(); await vite?.close(); });

async function scenario(t, options = {}) {
  const page = await browser.newPage({ viewport: { width: 1280, height: 900 } });
  page.setDefaultTimeout(8000);
  const requests = [], errors = [];
  t.after(async () => { try { assert.deepEqual(errors, [], 'application console errors/exceptions'); } finally { await page.close(); } });
  page.on('pageerror', error => errors.push(error.message));
  page.on('console', message => {
    // Expected HTTP errors belong to the API-state scenarios (including guest
    // session 401). Asset failures and application errors must still fail tests.
    if (message.type() === 'error' && !(message.text().startsWith('Failed to load resource:') && message.location().url.startsWith(`${origin}/api/`))) errors.push(message.text());
  });
  await page.route(`${origin}/api/**`, async route => {
    const url = new URL(route.request().url());
    requests.push(url);
    if (url.pathname === '/api/users/me') {
      return route.fulfill({ status: options.user ? 200 : 401, json: options.user ?? { error: { code: 'UNAUTHORIZED', message: 'guest' } } });
    }
    if (url.pathname === '/api/businesses/mine') return route.fulfill({ json: [] });
    const response = await options.respond?.(url, route.request());
    try { await route.fulfill(response ?? { status: 200, json: url.pathname === '/api/businesses/search' ? [second] : url.pathname === '/api/businesses/' ? { data: [first], nextCursor: cursor } : first }); }
    catch (error) { if (!/closed|cancel|abort|Invalid Interception/i.test(error.message)) throw error; }
  });
  return { page, requests };
}
const visible = async locator => locator.waitFor({ state: 'visible' });
const absent = async locator => locator.waitFor({ state: 'hidden' });

test('Home initial loading, real fields, guest header, card keyboard navigation and desktop composition', async t => {
  const { page } = await scenario(t, { respond: async () => { await pause(250); } });
  await page.goto(origin);
  await visible(page.getByText('Loading businesses', { exact: true }).first());
  await visible(page.getByRole('heading', { name: first.name }));
  await visible(page.getByRole('link', { name: 'Sign in', exact: true }));
  await visible(page.getByRole('link', { name: 'Create account' }));
  assert.equal(await page.getByRole('link', { name: 'Home', exact: true }).getAttribute('aria-current'), 'page');
  await visible(page.getByText('09:00 – 21:00'));
  await absent(page.getByText('Your active queues'));
  await page.screenshot({ path: 'output/playwright/discovery-home-desktop.png', fullPage: true });
  const card = page.locator('.ql-business-card');
  await card.focus(); await page.keyboard.press('Enter');
  await visible(page.getByRole('heading', { name: 'Business information' }));
  assert.equal(new URL(page.url()).pathname, `/business/${first.id}`);
});
test('guest Create account button renders through Radix Slot and navigates without a page error', async t => {
  const { page } = await scenario(t);
  await page.goto(origin);
  const createAccount = page.getByRole('link', { name: 'Create account', exact: true });
  await visible(createAccount);
  const colors = await createAccount.evaluate(element => {
    const style = getComputedStyle(element);
    return { foreground: style.color, background: style.backgroundColor };
  });
  assert.notEqual(colors.foreground, colors.background, 'button-link label must contrast with its background');
  await createAccount.click();
  await visible(page.getByRole('heading', { name: 'Create your QueueLite account.' }));
  assert.equal(new URL(page.url()).pathname, '/register');
});
test('logged-in Home greets the real username and preserves account controls', async t => {
  const { page } = await scenario(t, { user });
  await page.goto(origin);
  await visible(page.getByRole('heading', { name: /Good (morning|afternoon|evening), Ayu\./ }));
  await visible(page.getByRole('button', { name: 'Logout' }));
  await absent(page.getByRole('link', { name: 'Sign in', exact: true }));
});
test('search debounces, calls backend, shows results and clearing restores listing', async t => {
  const { page, requests } = await scenario(t);
  await page.goto(origin); await visible(page.getByRole('heading', { name: first.name }));
  const search = page.getByRole('searchbox', { name: 'Search businesses' });
  await search.fill('G'); await search.fill('Green');
  assert.equal(requests.filter(url => url.pathname.endsWith('/search')).length, 0);
  await visible(page.getByText('Searching businesses', { exact: true }));
  await visible(page.getByRole('heading', { name: second.name }));
  assert.deepEqual(requests.filter(url => url.pathname.endsWith('/search')).map(url => url.searchParams.get('name')), ['Green']);
  await absent(page.getByRole('button', { name: 'Load more' }));
  await page.getByRole('button', { name: 'Clear search' }).click();
  await visible(page.getByRole('heading', { name: first.name }));
  assert.ok(requests.filter(url => url.pathname === '/api/businesses/').length >= 2);
});
test('empty listing and no search results have distinct states', async t => {
  const { page } = await scenario(t, { respond: url => ({ json: url.pathname.endsWith('/search') ? [] : { data: [], nextCursor: null } }) });
  await page.goto(origin); await visible(page.getByRole('heading', { name: 'No businesses are available yet.' }));
  await page.getByRole('searchbox').fill('Unknown');
  await visible(page.getByRole('heading', { name: 'No businesses found for ‘Unknown’.' }));
  await visible(page.getByText('Try another search.'));
});
test('Home errors recover through retry', async t => {
  let fail = true;
  const { page } = await scenario(t, { respond: () => fail ? { status: 500, json: {} } : undefined });
  await page.goto(origin); await visible(page.getByText('We couldn’t load businesses right now.'));
  fail = false; await page.getByRole('button', { name: 'Try again', exact: true }).click();
  await visible(page.getByRole('heading', { name: first.name }));
});
test('Load more forwards opaque cursor, appends, disables during loading and stops at null', async t => {
  const { page, requests } = await scenario(t, { respond: async url => {
    if (url.searchParams.has('cursorID')) { await pause(200); return { json: { data: [second], nextCursor: null } }; }
  } });
  await page.goto(origin); await visible(page.getByRole('heading', { name: first.name }));
  await page.getByRole('button', { name: 'Load more' }).click();
  assert.equal(await page.getByRole('button', { name: 'Load more' }).isDisabled(), true);
  await visible(page.getByRole('heading', { name: second.name }));
  await visible(page.getByRole('heading', { name: first.name }));
  const next = requests.find(url => url.searchParams.has('cursorID'));
  assert.equal(next.searchParams.get('cursorID'), cursor.id);
  assert.equal(next.searchParams.get('cursorCreatedAt'), cursor.createdAt);
  assert.equal(next.searchParams.get('limit'), '12');
  await absent(page.getByRole('button', { name: 'Load more' }));
});
test('pagination failure preserves cards and retry reuses the cursor; empty final page ends loading', async t => {
  let fail = true;
  const { page } = await scenario(t, { respond: url => url.searchParams.has('cursorID') ? fail ? { status: 500, json: {} } : { json: { data: [], nextCursor: null } } : undefined });
  await page.goto(origin); await visible(page.getByRole('heading', { name: first.name }));
  await page.getByRole('button', { name: 'Load more' }).click();
  await visible(page.getByText('We couldn’t load more businesses. Your results are still here.'));
  await visible(page.getByRole('heading', { name: first.name }));
  fail = false; await page.getByRole('button', { name: 'Try loading more again' }).click();
  await absent(page.getByRole('button', { name: 'Load more' }));
  await visible(page.getByRole('heading', { name: first.name }));
});
test('late search and pagination responses cannot replace a newer query or cleared list', async t => {
  const { page } = await scenario(t, { respond: async url => {
    if (url.searchParams.get('name') === 'Slow' || url.searchParams.has('cursorID')) { await pause(700); return { json: url.pathname.endsWith('/search') ? [second] : { data: [second], nextCursor: null } }; }
  } });
  await page.goto(origin); await visible(page.getByRole('heading', { name: first.name }));
  await page.getByRole('button', { name: 'Load more' }).click();
  await page.getByRole('searchbox').fill('Slow');
  await page.waitForRequest(request => new URL(request.url()).searchParams.get('name') === 'Slow');
  await page.getByRole('button', { name: 'Clear search' }).click();
  await visible(page.getByRole('heading', { name: first.name }));
  await pause(850);
  await absent(page.getByRole('heading', { name: second.name }));
});
test('new search wins over an older in-flight search', async t => {
  const { page } = await scenario(t, { respond: async url => {
    if (url.searchParams.get('name') === 'Old') { await pause(700); return { json: [first] }; }
  } });
  await page.goto(origin); await visible(page.getByRole('heading', { name: first.name }));
  await page.getByRole('searchbox').fill('Old');
  await page.waitForRequest(request => new URL(request.url()).searchParams.get('name') === 'Old');
  await page.getByRole('searchbox').fill('New');
  await visible(page.getByRole('heading', { name: second.name }));
  await pause(800); await absent(page.getByRole('heading', { name: first.name }));
});
test('detail loads all business fields and CTA reaches the real join form', async t => {
  const { page } = await scenario(t, { respond: async () => { await pause(200); } });
  await page.goto(`${origin}/business/${first.id}`);
  await visible(page.getByText('Loading business details', { exact: true }));
  await visible(page.getByRole('heading', { name: first.name }));
  for (const value of [first.location, first.description, first.email, first.phoneNumber, 'Operational', '09:00 – 21:00']) await visible(page.getByText(value, { exact: true }));
  assert.equal(await page.getByRole('link', { name: 'Join queue' }).getAttribute('href'), `/business/${first.id}/join`);
  await page.screenshot({ path: 'output/playwright/discovery-detail-desktop.png', fullPage: true });
  await page.getByRole('link', { name: 'Join queue' }).click();
  await visible(page.getByRole('textbox', { name: 'Name', exact: true }));
});
test('unavailable business disables joining', async t => {
  const { page } = await scenario(t, { respond: () => ({ json: second }) });
  await page.goto(`${origin}/business/${second.id}`);
  await visible(page.getByRole('heading', { name: second.name }));
  assert.equal(await page.getByRole('button', { name: 'Join queue' }).isDisabled(), true);
  await visible(page.getByText('This business is currently unavailable.'));
});
test('404 and invalid business IDs render friendly not-found with navigation', async t => {
  const { page, requests } = await scenario(t, { respond: () => ({ status: 404, json: { error: { code: 'BUSINESS_NOT_FOUND', message: 'missing' } } }) });
  await page.goto(`${origin}/business/${first.id}`); await visible(page.getByRole('heading', { name: 'Business not found.' }));
  await visible(page.getByRole('link', { name: 'Back to businesses' }).first());
  const count = requests.filter(url => url.pathname.startsWith('/api/businesses/')).length;
  await page.goto(`${origin}/business/not-a-uuid`); await visible(page.getByRole('heading', { name: 'Business not found.' }));
  assert.equal(requests.filter(url => url.pathname.startsWith('/api/businesses/')).length, count);
});
test('detail generic error retries successfully', async t => {
  let fail = true;
  const { page } = await scenario(t, { respond: () => fail ? { status: 500, json: {} } : undefined });
  await page.goto(`${origin}/business/${first.id}`); await visible(page.getByText('We couldn’t load this business.'));
  fail = false; await page.getByRole('button', { name: 'Try again', exact: true }).click();
  await visible(page.getByRole('heading', { name: first.name }));
});
test('mobile discovery and detail stack without horizontal overflow', async t => {
  const { page } = await scenario(t, { user: { ...user, username: 'A very long customer username that should wrap' } });
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto(origin); await visible(page.getByRole('heading', { name: first.name }));
  assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth), true);
  await page.screenshot({ path: 'output/playwright/discovery-home-mobile.png', fullPage: true });
  await page.goto(`${origin}/business/${first.id}`); await visible(page.getByRole('link', { name: 'Join queue' }));
  assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth), true);
  await page.screenshot({ path: 'output/playwright/discovery-detail-mobile.png', fullPage: true });
});
