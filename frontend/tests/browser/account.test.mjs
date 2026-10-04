import { after, before, test } from 'node:test';
import assert from 'node:assert/strict';
import { mkdir } from 'node:fs/promises';
import { createServer } from 'vite';
import { chromium } from 'playwright';

const user = { id: '22222222-2222-4222-8222-222222222222', username: 'Ayu', phonenumber: '081234567890', email: 'ayu@example.test' };
const business = { id: '11111111-1111-4111-8111-111111111111', name: 'Northside Barbers', location: 'Pluit, Jakarta', description: 'Local appointments', operational: true, openTime: '09:00', closeTime: '21:00', email: 'hello@example.test', phoneNumber: '6281234567890', role: 'owner' };
const second = { ...business, id: '33333333-3333-4333-8333-333333333333', name: 'GreenCare', role: 'admin' };
const subscription = { id: '44444444-4444-4444-8444-444444444444', startDate: '2026-09-01T00:00:00Z', endDate: '2026-10-01T00:00:00Z', status: 'active' };
const userInfo = { subscription: { ...subscription, type: 'user', userId: user.id }, userPlan: { id: 'plan', userPlanType: 'premium', name: 'Premium', description: 'Your customer plan', price: 499, slots: 7, lastSlotsResetAt: subscription.startDate } };
const businessInfo = id => ({ subscription: { ...subscription, type: 'business', businessId: id }, businessPlan: { id: 'plan', businessPlanType: id === second.id ? 'plus' : 'free', description: 'Your business plan', price: 299, capacity: id === second.id ? 321 : 43, analysis: false, insight: false, prioritySupport: true, lastCapacityResetAt: subscription.startDate } });
const pause = ms => new Promise(resolve => setTimeout(resolve, ms));
const visible = locator => locator.waitFor({ state: 'visible' });
let vite, browser, origin;
before(async () => {
  vite = await createServer({ server: { host: '127.0.0.1', port: 0, open: false, ws: false } });
  await vite.listen(); origin = `http://127.0.0.1:${vite.httpServer.address().port}`;
  browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_CHANNEL ? { channel: process.env.PLAYWRIGHT_CHANNEL } : {}) });
  await mkdir('output/playwright', { recursive: true });
});
after(async () => { await browser?.close(); await vite?.close(); });
async function scenario(t, options = {}) {
  const page = await browser.newPage({ viewport: { width: 1280, height: 900 } }); page.setDefaultTimeout(8000);
  let currentUser = { ...user }; const memberships = structuredClone(options.businesses ?? [business, second]);
  const requests = [], errors = [];
  page.on('pageerror', error => errors.push(error.message));
  page.on('console', message => { if (message.type() === 'error' && !(message.text().startsWith('Failed to load resource:') && message.location().url.startsWith(`${origin}/api/`))) errors.push(message.text()); });
  t.after(async () => { try { assert.deepEqual(errors, [], 'application errors'); } finally { await page.close(); } });
  await page.route(`${origin}/api/**`, async route => {
    const request = route.request(), path = new URL(request.url()).pathname, method = request.method();
    requests.push({ path, method, body: request.postDataJSON(), headers: request.headers() });
    let response = await options.respond?.(path, request);
    if (response?.abort) return route.abort('failed');
    if (!response) {
      if (path === '/api/users/me') response = options.guest ? { status: 401, json: {} } : { json: currentUser };
      else if (path === '/api/businesses/mine') response = { json: memberships };
      else if (path === `/api/users/${user.id}` && method === 'PUT') { currentUser = { ...currentUser, ...request.postDataJSON() }; response = { json: { message: 'user updated' } }; }
      else if (path === '/api/businesses/' && method === 'POST') { const created = { ...business, ...request.postDataJSON(), id: '55555555-5555-4555-8555-555555555555' }; memberships.push(created); response = { status: 201, json: created }; }
      else if (path.startsWith('/api/businesses/') && method === 'PUT') { Object.assign(memberships.find(item => path.endsWith(item.id)), request.postDataJSON()); response = { json: { message: 'business updated' } }; }
      else if (path.startsWith('/api/businesses/')) response = { json: memberships.find(item => path.endsWith(item.id)) ?? business };
      else if (path === `/api/subscriptions/users/${user.id}`) response = { json: userInfo };
      else if (path.startsWith('/api/subscriptions/businesses/')) response = { json: businessInfo(path.split('/').at(-1)) };
      else throw new Error(`Unexpected fixture request ${method} ${path}`);
    }
    try { await route.fulfill(response); } catch (error) { if (!/closed|cancel|abort|Invalid Interception/i.test(error.message)) throw error; }
  });
  return { page, requests, memberships };
}
async function fillBusiness(page) {
  for (const [label, value] of Object.entries({ 'Business name': 'New clinic', Location: 'Jakarta', 'Open time': '22:00', 'Close time': '06:00', 'Business email': 'clinic@example.test', 'Business phone number': '081234567890' })) await page.getByLabel(label, { exact: true }).fill(value);
  await page.getByLabel('Description (optional)').fill('Evening appointments');
}

test('profile loads current user, saves phone/email, clears email, and changes password separately', async t => {
  const { page, requests } = await scenario(t); await page.goto(`${origin}/profile`);
  await visible(page.getByLabel('Username')); assert.equal(await page.getByLabel('Username').inputValue(), user.username); assert.equal(await page.getByLabel('Username').getAttribute('readonly'), '');
  await visible(page.getByText('Priority slots remaining: 7'));
  await page.getByLabel('Phone number', { exact: true }).fill('628111222333'); await page.getByLabel('Email (optional)').fill('new@example.test');
  await page.getByRole('button', { name: 'Save profile' }).click(); await visible(page.getByText('Profile saved.', { exact: true }));
  assert.deepEqual(requests.find(r => r.method === 'PUT').body, { phonenumber: '628111222333', email: 'new@example.test' });
  await page.getByLabel('Email (optional)').fill(''); await page.getByRole('button', { name: 'Save profile' }).click(); await visible(page.getByText('Profile saved.', { exact: true }));
  assert.equal(requests.filter(r => r.method === 'PUT').at(-1).body.email, '');
  await page.getByLabel('New password', { exact: true }).fill('new password'); await page.getByLabel('Confirm new password', { exact: true }).fill('new password');
  await page.getByRole('button', { name: 'Change password', exact: true }).click(); await visible(page.getByText('Password changed.', { exact: true }));
  assert.deepEqual(requests.filter(r => r.method === 'PUT').at(-1).body, { password: 'new password' });
  assert.equal(await page.getByLabel('New password', { exact: true }).inputValue(), '');
  assert.equal(await page.getByRole('link', { name: 'Plans & Subscription →' }).getAttribute('href'), '/plans');
  assert.equal(await page.getByRole('link', { name: 'My service counters →' }).getAttribute('href'), '/counters');
  assert.equal(await page.locator('main').getByRole('link', { name: 'Create a business →' }).getAttribute('href'), '/business/create');
  await page.screenshot({ path: 'output/playwright/account-profile-desktop.png', fullPage: true });
});
test('profile validation blocks requests and API errors are safe', async t => {
  const { page, requests } = await scenario(t, { respond: (path, request) => request.method() === 'PUT' ? { status: 500, json: { error: { code: 'UPDATE_ERROR', message: 'private database details' } } } : undefined });
  await page.goto(`${origin}/profile`); await visible(page.getByLabel('Phone number', { exact: true }));
  await page.getByLabel('Phone number', { exact: true }).fill('abc'); await page.getByLabel('Email (optional)').fill('bad'); await page.getByRole('button', { name: 'Save profile' }).click();
  await visible(page.getByText('Enter a phone number with 8–15 digits.')); assert.equal(requests.filter(r => r.method === 'PUT').length, 0);
  await page.getByLabel('Phone number', { exact: true }).fill(user.phonenumber); await page.getByLabel('Email (optional)').fill(user.email); await page.getByRole('button', { name: 'Save profile' }).click();
  await visible(page.getByText('We couldn’t complete this request. Please try again.')); assert.equal(await page.getByText('private database details').count(), 0);
});
for (const path of ['/profile', '/plans', '/business/create', `/business/${business.id}/settings`, '/business/manage']) test(`guest has clean sign-in path on ${path}`, async t => {
  const { page, requests } = await scenario(t, { guest: true }); await page.goto(origin + path); await visible(page.getByRole('heading', { name: 'Sign in to continue.' }));
  assert.equal(await page.getByRole('textbox').count(), 0);
  assert.ok((await page.locator('main').getByRole('link', { name: 'Sign in', exact: true }).getAttribute('href')).includes('returnTo='));
  assert.ok(requests.length > 0 && requests.every(r => r.path === '/api/users/me'));
});
test('new account can discover business creation from the header and profile', async t => {
  const { page } = await scenario(t, { businesses: [] }); await page.goto(`${origin}/profile`); await visible(page.getByText('You don’t own or manage a business yet.'));
  assert.equal(await page.getByRole('link', { name: 'Manage Business', exact: true }).count(), 0);
  const headerCreate = page.locator('header').getByRole('link', { name: 'Create Business', exact: true });
  assert.equal(await headerCreate.getAttribute('href'), '/business/create');
  await visible(page.locator('main').getByRole('link', { name: 'Create a business →', exact: true }));
  await headerCreate.click(); await visible(page.getByRole('button', { name: 'Create business', exact: true }));
});
test('create validates, submits only backend fields once, and navigates to settings', async t => {
  let release; const held = new Promise(resolve => { release = resolve; });
  const { page, requests } = await scenario(t, { businesses: [], respond: async (path, request) => { if (path === '/api/businesses/' && request.method() === 'POST') await held; } });
  await page.goto(`${origin}/business/create`); await visible(page.getByRole('button', { name: 'Create business', exact: true }));
  await page.getByRole('button', { name: 'Create business', exact: true }).click(); await visible(page.getByText('Enter a business name.')); assert.equal(requests.filter(r => r.method === 'POST').length, 0);
  await fillBusiness(page); await page.getByRole('button', { name: 'Create business', exact: true }).click();
  assert.equal(await page.getByRole('button', { name: 'Create business', exact: true }).isDisabled(), true);
  await page.locator('form').evaluate(form => { form.requestSubmit(); form.requestSubmit(); }); await pause(100); assert.equal(requests.filter(r => r.method === 'POST').length, 1); release();
  await visible(page.getByRole('heading', { name: 'Business settings', exact: true })); await visible(page.getByLabel('Operational', { exact: true }));
  assert.match(new URL(page.url()).pathname, /55555555-5555-4555-8555-555555555555\/settings$/);
  assert.deepEqual(requests.find(r => r.method === 'POST').body, { name: 'New clinic', location: 'Jakarta', description: 'Evening appointments', openTime: '22:00', closeTime: '06:00', email: 'clinic@example.test', phoneNumber: '081234567890' });
});
test('create error remains actionable; ambiguous setup blocks duplicate creation', async t => {
  const { page, requests } = await scenario(t, { respond: (path, request) => request.method() === 'POST' ? { status: 500, json: {} } : undefined });
  await page.goto(`${origin}/business/create`); await fillBusiness(page); await page.getByRole('button', { name: 'Create business', exact: true }).click();
  await visible(page.getByText(/We couldn’t confirm whether setup completed/)); await visible(page.getByRole('link', { name: 'Check your businesses' }));
  assert.equal(await page.getByRole('button', { name: 'Create business', exact: true }).isDisabled(), true); assert.equal(requests.filter(r => r.method === 'POST').length, 1);
});
test('settings load, save all editable fields and false operational state', async t => {
  const { page, requests } = await scenario(t); await page.goto(`${origin}/business/${business.id}/settings`); await visible(page.getByLabel('Business name', { exact: true }));
  assert.equal(await page.getByLabel('Business name', { exact: true }).inputValue(), business.name); await page.getByLabel('Business name', { exact: true }).fill('Updated business'); await page.getByRole('checkbox', { name: 'Operational', exact: true }).click();
  await page.getByRole('button', { name: 'Save business settings' }).click(); await visible(page.getByText('Business settings saved.'));
  assert.deepEqual(requests.find(r => r.method === 'PUT').body, { name: 'Updated business', location: business.location, description: business.description, openTime: business.openTime, closeTime: business.closeTime, email: business.email, phoneNumber: business.phoneNumber, operational: false });
  await page.screenshot({ path: 'output/playwright/account-settings-desktop.png', fullPage: true });
  await page.getByRole('link', { name: 'Manage Business', exact: true }).click();
  await visible(page.getByRole('heading', { name: 'Updated business', exact: true }));
});
for (const role of ['unrelated', 'counter']) test(`settings reject ${role} account before offering edits`, async t => {
  const { page, requests } = await scenario(t, { businesses: role === 'counter' ? [{ ...business, role }] : [] }); await page.goto(`${origin}/business/${business.id}/settings`);
  await visible(page.getByRole('heading', { name: 'Permission required' })); assert.equal(await page.getByRole('button', { name: 'Save business settings' }).count(), 0);
  assert.equal(requests.some(r => r.path === `/api/businesses/${business.id}` || r.method === 'PUT'), false);
});
test('server rejects revoked business authorization with a proper permission message', async t => {
  const { page } = await scenario(t, { respond: (path, request) => request.method() === 'PUT' ? { status: 403, json: { error: { code: 'BUSINESS_ACCESS_DENIED', message: 'private' } } } : undefined });
  await page.goto(`${origin}/business/${business.id}/settings`); await visible(page.getByRole('button', { name: 'Save business settings' })); await page.getByRole('button', { name: 'Save business settings' }).click();
  await visible(page.getByText('You do not have permission to manage this business or account.'));
});
test('plans show fetched subscriptions and user/business comparison tables without purchase actions', async t => {
  const { page, requests } = await scenario(t); await page.goto(`${origin}/plans`); await visible(page.getByText('Priority slots remaining: 7')); await visible(page.getByText('Queue capacity remaining: 43'));
  await visible(page.getByRole('heading', { name: 'Customer plans', exact: true })); await visible(page.getByRole('heading', { name: 'Business plans', exact: true }));
  await visible(page.getByRole('columnheader', { name: 'Premium', exact: true })); await visible(page.getByRole('columnheader', { name: 'Max', exact: true }));
  await visible(page.getByRole('rowheader', { name: 'Extra Priority Pass', exact: true })); await visible(page.getByRole('rowheader', { name: 'AI Weekly Insights', exact: true }));
  await page.getByLabel('Select business').click(); await page.getByRole('option', { name: `${second.name} (admin)` }).click(); await visible(page.getByText('Queue capacity remaining: 321')); await visible(page.getByRole('heading', { name: 'Plus', exact: true }));
  assert.equal(await page.getByRole('button', { name: 'Upgrade — coming later' }).isDisabled(), true); assert.equal(await page.getByRole('button', { name: 'Change plan — coming later' }).isDisabled(), true);
  assert.ok(requests.every(r => r.method === 'GET')); const text = await page.locator('main').innerText(); assert.doesNotMatch(text, /499|299|\$|checkout/i);
  await page.screenshot({ path: 'output/playwright/account-plans-desktop.png', fullPage: true });
  await page.getByRole('link', { name: 'Manage Business', exact: true }).click(); await visible(page.getByRole('heading', { name: second.name }));
  assert.equal(await page.getByRole('link', { name: 'Business settings →' }).getAttribute('href'), `/business/${second.id}/settings`);
});
test('subscription errors can be retried and counter-only staff cannot see business plans', async t => {
  let fail = true;
  const { page, requests } = await scenario(t, { businesses: [{ ...business, role: 'counter' }], respond: path => path === `/api/subscriptions/users/${user.id}` && fail ? { status: 500, json: {} } : undefined });
  await page.goto(`${origin}/plans`); await visible(page.getByText('We couldn’t complete this request. Please try again.')); fail = false;
  await page.getByRole('button', { name: 'Try again' }).click(); await visible(page.getByText('Priority slots remaining: 7'));
  assert.equal(await page.getByRole('heading', { name: 'Business plans', exact: true }).count(), 0);
  assert.equal(await page.getByRole('button', { name: 'Change plan — coming later' }).count(), 0);
  assert.equal(requests.some(r => r.path.startsWith('/api/subscriptions/businesses/')), false);
});
test('account without a business sees user plans but no business plan list', async t => {
  const { page, requests } = await scenario(t, { businesses: [] }); await page.goto(`${origin}/plans`);
  await visible(page.getByRole('heading', { name: 'Customer plans', exact: true }));
  assert.equal(await page.getByRole('heading', { name: 'Business plans', exact: true }).count(), 0);
  assert.equal(await page.getByLabel('Select business').count(), 0);
  assert.equal(requests.some(r => r.path.startsWith('/api/subscriptions/businesses/')), false);
});
test('mobile forms and account navigation fit without horizontal overflow', async t => {
  const { page } = await scenario(t); await page.setViewportSize({ width: 390, height: 844 });
  for (const path of ['/profile', '/plans', '/business/create', `/business/${business.id}/settings`]) {
    await page.goto(origin + path); await visible(page.getByRole('heading', { level: 1 })); await visible(page.locator('main .ql-card').first());
    await pause(100); assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth), true, path);
  }
  await visible(page.getByLabel('Operational', { exact: true })); await page.screenshot({ path: 'output/playwright/account-settings-mobile.png', fullPage: true });
});
