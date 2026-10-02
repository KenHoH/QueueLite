import { after, test } from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';
const vite = await createServer({ configFile: false, optimizeDeps: { noDiscovery: true, include: [] }, server: { middlewareMode: true, ws: false }, appType: 'custom' });
after(() => vite.close());
const { createAuthStore, RegistrationSignInError } = await vite.ssrLoadModule('/src/state/authStore.ts');
const { APIError } = await vite.ssrLoadModule('/src/api/errors.ts');
const { validateAuth, authErrorMessage } = await vite.ssrLoadModule('/src/pages/authForm.ts');
const user = { id: 'user-id', username: 'Matthew', phonenumber: '08123456789' };
const credentials = { username: 'Matthew', password: 'test-password' };
const empty = { username: '', password: '', confirmPassword: '', phonenumber: '', email: '' };
const unauthorized = () => new APIError(401, 'CLIENT_HTTP_401', 'unauthorized');
const service = (overrides = {}) => ({ getCurrentUser: async () => user, login: async () => ({}), registerUser: async () => user, logout: async () => ({}), ...overrides });
test('login required validation and safe invalid-credential messages', () => {
  assert.deepEqual(validateAuth(empty, false), { username: 'Enter your username.', password: 'Enter your password.' });
  assert.deepEqual(validateAuth({ ...empty, ...credentials }, false), {});
  assert.equal(authErrorMessage(new APIError(401, 'INVALID_CREDENTIALS', 'internal detail')), 'Incorrect username or password.');
  assert.doesNotMatch(authErrorMessage(new Error('secret')), /secret/);
});
test('registration checks matching passwords, phone and optional email without a minimum length', () => {
  const fields = { ...empty, ...credentials, phonenumber: '081234567890', confirmPassword: 'different' };
  assert.equal(validateAuth(fields, true).confirmPassword, 'Passwords must match.');
  assert.deepEqual(validateAuth({ ...fields, password: 'a', confirmPassword: 'a' }, true), {});
  assert.ok(validateAuth({ ...fields, email: 'invalid', phonenumber: '' }, true).email);
  assert.ok(validateAuth({ ...fields, phonenumber: '' }, true).phonenumber);
});
test('bootstrap restores current user; unauthorized becomes guest; outage remains recoverable', async () => {
  const store = createAuthStore(service());
  await store.refreshUser();
  assert.deepEqual(store.getSnapshot(), { status: 'authenticated', user });
  const guest = createAuthStore(service({ getCurrentUser: async () => { throw unauthorized(); } }));
  await assert.rejects(guest.refreshUser());
  assert.equal(guest.getSnapshot().status, 'guest');
  let down = true;
  const recovery = createAuthStore(service({ getCurrentUser: async () => { if (down) throw new Error('offline'); return user; } }));
  await assert.rejects(recovery.refreshUser());
  assert.equal(recovery.getSnapshot().status, 'error');
  down = false; await recovery.refreshUser();
  assert.equal(recovery.getSnapshot().status, 'authenticated');
});
test('login loads identity after credentials and rejects invalid credentials', async () => {
  const calls = [];
  const store = createAuthStore(service({ login: async body => { calls.push(body); }, getCurrentUser: async () => { calls.push('me'); return user; } }));
  await store.login(credentials);
  assert.deepEqual(calls, [credentials, 'me']);
  assert.equal(store.getSnapshot().user, user);
  const invalid = createAuthStore(service({ login: async () => { throw new APIError(401, 'INVALID_CREDENTIALS', 'invalid'); } }));
  await assert.rejects(invalid.login(credentials));
  assert.notEqual(invalid.getSnapshot().status, 'authenticated');
});
test('register then login then current user; partial success is explicit', async () => {
  const calls = [];
  const registration = { ...credentials, phonenumber: '08' };
  const store = createAuthStore(service({ registerUser: async body => { calls.push(['register', body]); return user; }, login: async body => { calls.push(['login', body]); }, getCurrentUser: async () => { calls.push(['me']); return user; } }));
  await store.register(registration);
  assert.deepEqual(calls, [['register', registration], ['login', credentials], ['me']]);
  assert.equal(store.getSnapshot().status, 'authenticated');
  const partial = createAuthStore(service({ login: async () => { throw new Error('offline'); } }));
  await assert.rejects(partial.register(registration), error => error instanceof RegistrationSignInError);
});
test('logout clears identity only after server success', async () => {
  let fail = true;
  const store = createAuthStore(service({ logout: async () => { if (fail) throw new Error('offline'); } }));
  await store.refreshUser();
  await assert.rejects(store.logout());
  assert.equal(store.getSnapshot().status, 'authenticated');
  fail = false; await store.logout();
  assert.deepEqual(store.getSnapshot(), { status: 'guest', user: null });
});
test('stale bootstrap cannot overwrite login and duplicate mutations are blocked', async () => {
  let rejectBootstrap;
  let calls = 0;
  const store = createAuthStore(service({ getCurrentUser: () => ++calls === 1 ? new Promise((_, reject) => { rejectBootstrap = reject; }) : Promise.resolve(user) }));
  const bootstrap = store.refreshUser();
  const login = store.login(credentials);
  await assert.rejects(store.login(credentials), /in progress/);
  await login;
  rejectBootstrap(unauthorized()); await assert.rejects(bootstrap);
  assert.equal(store.getSnapshot().status, 'authenticated');
});
test('real API services use cookie client, exact paths, and exact request bodies', async () => {
  const originalFetch = globalThis.fetch;
  const calls = [];
  globalThis.fetch = async (url, init) => { calls.push({ url, ...init }); return Response.json(url.endsWith('/me') || url.endsWith('/users/') ? user : { message: 'ok' }); };
  try {
    const api = await vite.ssrLoadModule('/src/api/auth.ts');
    await api.registerUser({ ...credentials, phonenumber: '08' });
    await api.login(credentials); await api.getCurrentUser(); await api.logout();
    assert.deepEqual(calls.map(call => [call.url, call.method]), [['/api/users/', 'POST'], ['/api/users/login', 'POST'], ['/api/users/me', 'GET'], ['/api/users/logout', 'POST']]);
    assert.ok(calls.every(call => call.credentials === 'include'));
    assert.deepEqual(JSON.parse(calls[0].body), { ...credentials, phonenumber: '08' });
    assert.equal(calls[3].body, undefined);
  } finally { globalThis.fetch = originalFetch; }
});

test('failed login superseding bootstrap leaves a recoverable session state', async () => {
  let finishBootstrap;
  const store = createAuthStore(service({ getCurrentUser: () => new Promise(resolve => { finishBootstrap = resolve; }), login: async () => { throw unauthorized(); } }));
  const bootstrap = store.refreshUser();
  await assert.rejects(store.login(credentials));
  assert.equal(store.getSnapshot().status, 'error');
  finishBootstrap(user); await bootstrap;
  assert.equal(store.getSnapshot().status, 'error');
});


