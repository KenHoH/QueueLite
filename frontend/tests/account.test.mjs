import { after, test } from 'node:test';
import assert from 'node:assert/strict';
import { readdirSync, readFileSync } from 'node:fs';
import { createServer } from 'vite';
const vite = await createServer({ configFile: false, optimizeDeps: { noDiscovery: true, include: [] }, server: { middlewareMode: true, ws: false }, appType: 'custom' });
after(() => vite.close());
const forms = await vite.ssrLoadModule('/src/pages/accountForms.ts');
const plans = await vite.ssrLoadModule('/src/pages/subscriptionPlans.ts');
test('malformed membership responses become recoverable API errors', async () => {
  const original = globalThis.fetch;
  globalThis.fetch = async () => Response.json({ id: 'not-a-membership-list' });
  try {
    const { getMyBusinesses } = await vite.ssrLoadModule('/src/api/businesses.ts');
    await assert.rejects(getMyBusinesses(), error => error.code === 'CLIENT_INVALID_RESPONSE' && error.kind === 'response');
  } finally { globalThis.fetch = original; }
});
test('quiet profile refresh updates identity without unmounting the authenticated page', async () => {
  const { createAuthStore } = await vite.ssrLoadModule('/src/state/authStore.ts');
  let user = { id: 'user', username: 'Ayu', phonenumber: '081234567890' };
  const store = createAuthStore({ getCurrentUser: async () => user });
  await store.refreshUser();
  const statuses = []; store.subscribe(() => statuses.push(store.getSnapshot().status));
  user = { ...user, email: 'new@example.test' }; await store.refreshUser(true);
  assert.deepEqual(statuses, ['authenticated']); assert.equal(store.getSnapshot().user.email, 'new@example.test');
});
test('profile validates phone, optional email, and separate password confirmation against bcrypt byte limit', () => {
  assert.deepEqual(forms.validateProfile({ phonenumber: '+62 812-3456-7890', email: '' }), {});
  assert.deepEqual(Object.keys(forms.validateProfile({ phonenumber: 'abc', email: 'invalid' })), ['phonenumber', 'email']);
  assert.deepEqual(forms.validatePassword('new password', 'new password'), {});
  assert.ok(forms.validatePassword('new', 'other').confirmation);
  assert.ok(forms.validatePassword('é'.repeat(37), 'é'.repeat(37)).password);
});
test('business validation accepts overnight hours and requires actual contact fields', () => {
  const business = { ...forms.emptyBusiness, name: 'Night clinic', location: 'Jakarta', openTime: '22:00', closeTime: '06:00', email: 'hello@example.test', phoneNumber: '081234567890' };
  assert.deepEqual(forms.validateBusiness(business), {});
  const errors = forms.validateBusiness({ ...forms.emptyBusiness, openTime: '24:00', closeTime: '06:60' });
  for (const field of ['name', 'location', 'openTime', 'closeTime', 'email', 'phoneNumber']) assert.ok(errors[field]);
});
test('date-like form controls stay behind the shadcn input layer', () => {
  const sourceRoot = new URL('../src/', import.meta.url);
  const sources = readdirSync(sourceRoot, { recursive: true })
    .filter(path => typeof path === 'string' && path.endsWith('.tsx'))
    .map(path => readFileSync(new URL(path, sourceRoot), 'utf8'));
  assert.doesNotMatch(sources.join('\n'), /<input\b[^>]*type=["'](?:date|datetime-local|month|week|time)["']/);
  const businessSetup = readFileSync(new URL('../src/pages/BusinessSetup.tsx', import.meta.url), 'utf8');
  assert.match(businessSetup, /<Input label="Open time" type="time"/);
  assert.match(businessSetup, /<Input label="Close time" type="time"/);
  const foundation = readFileSync(new URL('../src/components/foundation.tsx', import.meta.url), 'utf8');
  assert.match(foundation, /<ShadcnInput/);
});
test('only owner/admin can manage; errors retain safe account copy', async () => {
  const { APIError } = await vite.ssrLoadModule('/src/api/errors.ts');
  assert.equal(forms.canManage({ role: 'owner' }), true); assert.equal(forms.canManage({ role: 'admin' }), true);
  assert.equal(forms.canManage({ role: 'counter' }), false); assert.equal(forms.canManage({ role: 'other' }), false);
  assert.match(forms.accountError(new APIError(403, 'BUSINESS_ACCESS_DENIED', 'private details')), /permission/);
  assert.doesNotMatch(forms.accountError(new APIError(500, 'UPDATE_ERROR', 'private details')), /private/);
});
test('plan catalogs match product copy and business plans require management membership', () => {
  assert.deepEqual(plans.userPlanComparison.plans, ['Standard', 'Premium']);
  assert.deepEqual(plans.businessPlanComparison.plans, ['Free', 'Plus', 'Pro', 'Max']);
  assert.equal(plans.userPlanComparison.features.find(item => item.name === 'Priority Passes').values[1], '4/month');
  assert.equal(plans.businessPlanComparison.features.find(item => item.name === 'Queue Capacity').values[3], 'Up to 10,000/day');
  assert.equal(plans.canShowBusinessPlans([]), false);
  assert.equal(plans.canShowBusinessPlans([{ role: 'counter' }]), false);
  assert.equal(plans.canShowBusinessPlans([{ role: 'owner' }]), true);
  assert.equal(plans.canShowBusinessPlans([{ role: 'admin' }]), true);
});
test('account APIs use context-only memberships, exact editable fields and read-only plan endpoints', async () => {
  const original = globalThis.fetch, calls = [];
  globalThis.fetch = async (url, init) => { calls.push({ url, init }); return Response.json(url.endsWith('/mine') ? [] : url.startsWith('/api/subscriptions/') ? { subscription: { type: url.includes('/users/') ? 'user' : 'business', status: 'active', startDate: '2026-10-01T00:00:00Z' }, userPlan: { userPlanType: 'standard', slots: 0 }, businessPlan: { businessPlanType: 'free', capacity: 50 } } : { message: 'ok' }); };
  try {
    const businesses = await vite.ssrLoadModule('/src/api/businesses.ts');
    const users = await vite.ssrLoadModule('/src/api/users.ts');
    const subscriptions = await vite.ssrLoadModule('/src/api/subscriptions.ts');
    await businesses.getMyBusinesses();
    await users.updateUser('user', { phonenumber: '081234567890', email: '' });
    await businesses.updateBusiness('business', { operational: false });
    await subscriptions.getUserSubscription('user'); await subscriptions.getBusinessSubscription('business');
    assert.equal(calls[0].url, '/api/businesses/mine'); assert.equal(calls[0].init.body, undefined);
    assert.deepEqual(JSON.parse(calls[1].init.body), { phonenumber: '081234567890', email: '' });
    assert.deepEqual(JSON.parse(calls[2].init.body), { operational: false });
    assert.ok(calls.slice(3).every(call => call.init.method === 'GET' && call.init.body === undefined));
    assert.ok(calls.every(call => call.init.credentials === 'include'));
  } finally { globalThis.fetch = original; }
});
