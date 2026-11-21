/** Local visual review only. Installed by Vite for `serve --mode ui-preview`.
 * No code from this module is imported by the browser application or production build.
 */
import { randomUUID } from 'node:crypto';
import type { IncomingMessage, ServerResponse } from 'node:http';
import type { Plugin } from 'vite';
import type { Business, Counter, Queue, QueueState, User } from '../src/api/types';

type Persona = 'owner' | 'customer' | 'staff' | 'guest';
const businessID = '11111111-1111-4111-8111-111111111111';
const quickFixID = '22222222-2222-4222-8222-222222222222';
const userID = '33333333-3333-4333-8333-333333333333';
const ticketID = '44444444-4444-4444-8444-444444444444';
const counterID = '55555555-5555-4555-8555-555555555555';
const guestOwnerID = '66666666-6666-4666-8666-666666666666';
const now = () => new Date().toISOString();
const active = (q: Queue) => ['waiting', 'called', 'processing'].includes(q.state);

function initialState() {
  const user: User = { id: userID, username: 'Matthew', phonenumber: '081234567890', email: 'matthew@preview.example' };
  const names = ['Northside Barbers', 'QuickFix Service Center', 'GreenCare Clinic', 'Studio Nine Salon'];
  const businesses: Business[] = names.map((name, i) => ({ id: i === 0 ? businessID : i === 1 ? quickFixID : `${i + 6}1111111-1111-4111-8111-111111111111`, name, location: ['Pluit, Jakarta', 'Pantai Indah Kapuk, Jakarta', 'Muara Karang, Jakarta', 'Pluit, Jakarta'][i], description: ['A fresh cut and a little time for yourself. Visit our neighborhood team for thoughtful, unhurried service.', 'Dependable help for the devices you use every day. Bring your device in for an assessment with our technicians.', 'Thoughtful care, closer to home. Meet our neighborhood healthcare team.', 'A good visit starts with a little time for yourself.'][i], operational: i !== 3, openTime: '09:00', closeTime: '21:00', email: 'hello@preview.example', phoneNumber: '0215887100' }));
  const queues: Queue[] = Array.from({ length: 8 }, (_, i) => ({ id: i === 7 ? ticketID : randomUUID(), businessId: businessID, userId: i === 7 ? userID : randomUUID(), name: `A${String(i + 13).padStart(3, '0')}`, priority: false, state: i < 3 ? 'processing' : 'waiting', createdAt: now(), ...(i < 3 ? { processingAt: new Date(Date.now() - 5 * 60000).toISOString() } : {}) }));
  const counters: Counter[] = [0, 1, 2].map(i => ({ id: i === 1 ? counterID : randomUUID(), businessId: businessID, name: `Counter ${i + 1}`, currentEmployeeId: userID, currentQueueId: queues[i].id }));
  counters.forEach((c, i) => { queues[i].calledByCounterId = c.id; });
  queues.push(...Array.from({ length: 33 }, (_, i): Queue => ({ id: randomUUID(), businessId: businessID, userId: randomUUID(), name: `A${String(i + 1).padStart(3, '0')}`, priority: false, state: i < 30 ? 'done' : i < 32 ? 'skipped' : 'cancelled', createdAt: new Date(Date.now() - 86400000).toISOString(), processingAt: new Date(Date.now() - 86400000 + 60000).toISOString(), doneAt: new Date(Date.now() - 86400000 + 11 * 60000).toISOString() })));
  counters.push(...[1, 2].map(i => ({ id: randomUUID(), businessId: quickFixID, name: `Counter ${i}`, currentEmployeeId: userID })));
  return { persona: 'owner' as Persona, user, businesses, queues, counters, premium: false, members: [{ userId: userID, username: 'Matthew', role: 'owner' }] };
}
type Session = ReturnType<typeof initialState>;
const pages = [
  ['Home', '/'], ['Business details', `/business/${quickFixID}`], ['Join form (guest)', `/business/${quickFixID}/join`, 'join'],
  ['Join confirmation', `/business/${quickFixID}/join`, 'confirmation'], ['My queues', '/my-queues'],
  ...(['waiting', 'called', 'processing', 'done', 'skipped', 'cancelled'] as QueueState[]).map(state => [`Ticket: ${state}`, `/queue/${ticketID}`, state]),
  ['Business overview', `/business/${businessID}/dashboard`], ['Counter workspace', `/counter/${counterID}`],
  ['Counters & members', `/business/${businessID}/counters`], ['Members', `/business/${businessID}/members`],
  ['Business settings', `/business/${businessID}/settings`], ['Share queue', `/business/${businessID}/share`], ['Business plans', `/business/${businessID}/plans`],
  ['Manage businesses', '/business/manage'], ['Create business', '/business/create'], ['Profile', '/profile'], ['Customer plans', '/plans'],
  ['Sign in', '/login', 'guest'], ['Register', '/register', 'guest'], ['404 page', '/preview-page-not-found'],
];

async function body(req: IncomingMessage): Promise<Record<string, unknown>> {
  let raw = '';
  for await (const chunk of req) { raw += chunk; if (raw.length > 32768) throw new Error('Request too large'); }
  return raw ? JSON.parse(raw) : {};
}
function json(res: ServerResponse, value: unknown, status = 200) {
  res.statusCode = status;
  res.setHeader('Content-Type', 'application/json');
  res.setHeader('Cache-Control', 'no-store');
  res.end(JSON.stringify(value));
}
function owned(s: Session) { return s.queues.filter(q => q.userId === (s.persona === 'guest' ? guestOwnerID : userID)); }
function snapshot(s: Session, queue: Queue) {
  const business = s.businesses.find(b => b.id === queue.businessId)!;
  const waiting = s.queues.filter(q => q.businessId === business.id && q.state === 'waiting');
  const counters = s.counters.filter(c => c.businessId === business.id).map(c => {
    const q = s.queues.find(q => q.id === c.currentQueueId && ['called', 'processing'].includes(q.state));
    return { id: c.id, name: c.name, state: q?.state ?? 'idle', ...(q ? { currentQueueId: q.id, currentQueueName: q.name } : {}) };
  });
  const serving = counters.filter(c => c.currentQueueId);
  const index = waiting.findIndex(q => q.id === queue.id);
  return { queue, business: { id: business.id, name: business.name, operational: business.operational }, counters, totalCounters: counters.length, activeCounters: serving.length, currentlyServing: serving.map(c => ({ counterId: c.id, counterName: c.name, queueId: c.currentQueueId, queueName: c.currentQueueName, state: c.state })), totalWaiting: waiting.length, totalActiveQueues: waiting.length + serving.length, ...(waiting[0] ? { nextQueue: { queueId: waiting[0].id, queueName: waiting[0].name } } : {}), ...(index >= 0 ? { customerPosition: { position: index + 1, ahead: index, queueName: queue.name }, estimatedWaitMinutes: Math.ceil(index / Math.max(serving.length, 1)) * 5 } : {}), updatedAt: now() };
}
function addTicket(s: Session, businessId: string, userId = s.persona === 'guest' ? guestOwnerID : userID) {
  const queue: Queue = { id: randomUUID(), businessId, userId, name: `A${String(1 + s.queues.filter(q => q.businessId === businessId).length).padStart(3, '0')}`, state: 'waiting', priority: false, createdAt: now() };
  s.queues.push(queue); return queue;
}
function callNext(s: Session, counter: Counter) {
  const next = s.queues.find(q => q.businessId === counter.businessId && q.state === 'waiting');
  counter.currentQueueId = next?.id;
  if (!next) return { queue: null };
  next.state = 'called'; next.calledByCounterId = counter.id; next.calledAt = now();
  return { queueId: next.id, queueName: next.name };
}

export function uiPreviewPlugin(): Plugin {
  const sessions = new Map<string, Session>();
  return {
    name: 'queuelite-local-ui-preview',
    apply: 'serve',
    transformIndexHtml() {
      return [{ tag: 'script', attrs: { type: 'module', src: '/__ui-preview/toolbar.js' }, injectTo: 'body' }];
    },
    configureServer(server) {
      server.middlewares.use(async (req, res, next) => {
        const url = new URL(req.url ?? '/', 'http://127.0.0.1');
        const path = url.pathname;
        if (!path.startsWith('/api/') && !path.startsWith('/__ui-preview/')) return next();
        const method = req.method ?? 'GET';
        const cookieID = req.headers.cookie?.match(/(?:^|;\s*)ql_ui_preview=([a-f0-9-]+)/)?.[1];
        const id = cookieID && sessions.has(cookieID) ? cookieID : randomUUID();
        if (!sessions.has(id)) { if (sessions.size > 100) sessions.delete(sessions.keys().next().value!); sessions.set(id, initialState()); }
        if (id !== cookieID) res.setHeader('Set-Cookie', `ql_ui_preview=${id}; Path=/; HttpOnly; SameSite=Lax`);
        const s = sessions.get(id)!;
        const ok = () => json(res, { message: 'Preview updated' });
        const fail = (code: string, message: string, status = 404) => json(res, { error: { code, message } }, status);
        try {
          if (path === '/__ui-preview/toolbar.js') { res.setHeader('Content-Type', 'text/javascript'); return res.end(toolbar); }
          if (path === '/__ui-preview/pages') return json(res, { pages, persona: s.persona });
          const data = method === 'GET' ? {} : await body(req);
          if (path === '/__ui-preview/reset' && method === 'POST') { sessions.set(id, initialState()); return ok(); }
          if (path === '/__ui-preview/select' && method === 'POST') {
            const preset = pages[Number(data.page)];
            if (preset) {
              const state = preset[2];
              s.persona = ['guest', 'join', 'confirmation'].includes(state ?? '') ? 'guest' : 'owner';
              if (state === 'join' || state === 'confirmation') s.queues = s.queues.filter(q => !(q.businessId === quickFixID && q.userId === guestOwnerID));
              if (state === 'confirmation') addTicket(s, quickFixID, guestOwnerID);
              if (['waiting', 'called', 'processing', 'done', 'skipped', 'cancelled'].includes(state ?? '')) {
                const ticket = s.queues.find(q => q.id === ticketID)!;
                ticket.state = state as QueueState;
                s.counters.forEach(c => { if (c.currentQueueId === ticketID) delete c.currentQueueId; });
                if (state === 'called' || state === 'processing') {
                  const c = s.counters.find(c => c.id === counterID)!;
                  const previous = s.queues.find(q => q.id === c.currentQueueId);
                  if (previous) previous.state = 'done';
                  c.currentQueueId = ticketID; ticket.calledByCounterId = c.id; ticket.processingAt = now();
                }
              }
              return json(res, { path: preset[1] });
            }
            if (['owner', 'customer', 'staff', 'guest'].includes(String(data.persona))) s.persona = data.persona as Persona;
            return json(res, { path: '/' });
          }
          const api = path.slice(4).replace(/\/$/, '') || '/';
          if (api === '/users/me') return s.persona === 'guest' ? fail('UNAUTHORIZED', 'Guest preview', 401) : json(res, s.user);
          if (api === '/users/logout') { s.persona = 'guest'; return ok(); }
          if (api === '/users/login') { s.persona = 'owner'; return ok(); }
          if (api === '/users' && method === 'POST') { s.user.username = String(data.username || 'Matthew'); return json(res, s.user, 201); }
          if (api.startsWith('/users/') && method === 'PUT') { if (typeof data.phonenumber === 'string') s.user.phonenumber = data.phonenumber; if (typeof data.email === 'string') s.user.email = data.email; return ok(); }
          if (api === '/businesses/mine') return json(res, s.persona === 'guest' || s.persona === 'customer' ? [] : s.businesses.slice(0, 2).map(b => ({ ...b, role: s.persona === 'staff' ? 'counter' : 'owner' })));
          if (api === '/businesses/search') return json(res, s.businesses.filter(b => b.name.toLowerCase().includes((url.searchParams.get('name') ?? '').toLowerCase())));
          if (api === '/businesses' && method === 'GET') return json(res, { data: s.businesses, nextCursor: null });
          if (api === '/businesses' && method === 'POST') { const b = { ...s.businesses[0], ...data, id: randomUUID(), operational: true } as Business; s.businesses.unshift(b); s.counters.push({ id: randomUUID(), businessId: b.id, name: 'Default Counter', currentEmployeeId: userID }); return json(res, b, 201); }
          const businessMatch = api.match(/^\/businesses\/([^/]+)(?:\/(counters|queues|members))?$/);
          if (businessMatch) {
            const b = s.businesses.find(b => b.id === businessMatch[1]); if (!b) return fail('BUSINESS_NOT_FOUND', 'Business not found');
            if (businessMatch[2] === 'counters') return json(res, s.counters.filter(c => c.businessId === b.id && (s.persona !== 'staff' || c.id === counterID)));
            if (businessMatch[2] === 'queues') return json(res, s.queues.filter(q => q.businessId === b.id));
            if (businessMatch[2] === 'members') { if (method === 'PUT') { const member = { userId: randomUUID(), username: String(data.identifier || 'Preview member'), role: String(data.role || 'counter') }; s.members.push(member); return json(res, member); } return json(res, s.members); }
            if (method === 'PUT') { Object.assign(b, data); return ok(); }
            if (method === 'DELETE') { s.businesses = s.businesses.filter(item => item.id !== b.id); return ok(); }
            return json(res, b);
          }
          if (api === '/queues/me' || api === '/queues/guest/me') return json(res, owned(s).filter(active));
          const resolve = api.match(/^\/queues\/qr\/([^/]+)\/resolve$/);
          if (resolve) { const q = owned(s).find(q => q.businessId === resolve[1] && active(q)); return json(res, { businessId: resolve[1], authenticated: s.persona !== 'guest', requiresGuestForm: s.persona === 'guest', ...(q ? { activeQueueId: q.id, activeQueue: q } : {}) }); }
          const join = api.match(/^\/queues\/business\/([^/]+)\/join$/);
          if (join) { const b = s.businesses.find(b => b.id === join[1]); if (!b) return fail('BUSINESS_NOT_FOUND', 'Business not found'); if (!b.operational) return fail('BUSINESS_UNAVAILABLE', 'Business unavailable', 409); const existing = owned(s).find(q => q.businessId === b.id && active(q)); if (existing) return fail('ACTIVE_QUEUE_EXISTS', 'Active preview ticket exists', 409); return json(res, addTicket(s, b.id), 201); }
          const queueMatch = api.match(/^\/queues\/([^/]+)(?:\/(customer-status|state))?$/);
          if (queueMatch) { const q = s.queues.find(q => q.id === queueMatch[1]); if (!q) return fail('QUEUE_NOT_FOUND', 'Ticket not found'); if (method === 'PATCH' || method === 'PUT') { q.state = 'cancelled'; return ok(); } if (queueMatch[2] === 'customer-status') return json(res, snapshot(s, q)); if (queueMatch[2] === 'state') return json(res, { state: q.state }); return json(res, q); }
          if (api === '/counters' && method === 'POST') { const c = { id: randomUUID(), businessId: String(data.businessId), name: String(data.name), currentEmployeeId: userID }; s.counters.push(c); return json(res, c, 201); }
          const counterMatch = api.match(/^\/counters\/([^/]+)(?:\/(.*))?$/);
          if (counterMatch) {
            const c = s.counters.find(c => c.id === counterMatch[1]); if (!c) return fail('COUNTER_NOT_FOUND', 'Counter not found');
            const action = counterMatch[2];
            if (action?.endsWith('/call-next')) return json(res, callNext(s, c));
            if (action?.startsWith('queues/')) { const q = s.queues.find(q => q.id === action.split('/')[1]); if (!q) return fail('QUEUE_NOT_FOUND', 'Ticket not found'); q.state = action.endsWith('/process') ? 'processing' : action.endsWith('/skip') ? 'skipped' : 'done'; q.processingAt = now(); q.doneAt = now(); if (q.state === 'processing') return json(res, { queueId: q.id, state: q.state }); delete c.currentQueueId; return action.endsWith('/skip') ? json(res, callNext(s, c)) : ok(); }
            if (method === 'PUT') { Object.assign(c, data); if (!data.currentEmployeeId) delete c.currentEmployeeId; return ok(); }
            if (method === 'DELETE') { s.counters = s.counters.filter(item => item.id !== c.id); return ok(); }
            return json(res, c);
          }
          if (api.startsWith('/subscriptions/updateSubscription/')) { s.premium = true; return ok(); }
          if (api.startsWith('/subscriptions/users/')) return json(res, { subscription: { id: 'preview-user-subscription', type: 'user', status: 'active', startDate: now() }, userPlan: { id: 'preview-user-plan', userPlanType: s.premium ? 'premium' : 'standard', name: s.premium ? 'Premium' : 'Standard', description: 'Sample customer subscription', price: 0, slots: s.premium ? 4 : 0, lastSlotsResetAt: now() } });
          if (api.startsWith('/subscriptions/businesses/')) return json(res, { subscription: { id: 'preview-business-subscription', type: 'business', status: 'active', startDate: now() }, businessPlan: { id: 'preview-business-plan', businessPlanType: 'free', capacity: 43, price: 0, description: 'Sample business subscription', analysis: false, insight: false, prioritySupport: false, lastCapacityResetAt: now() } });
          return fail('UI_PREVIEW_ONLY', 'This action is not part of the visual preview.', 501);
        } catch { return fail('PREVIEW_INVALID_REQUEST', 'Preview request could not be read.', 400); }
      });
    },
  };
}

const toolbar = `
const host=document.createElement('aside');host.id='ql-ui-preview';document.body.append(host);const root=host.attachShadow({mode:'open'});
root.innerHTML=\`<style>:host{position:fixed;bottom:12px;right:12px;z-index:1000;font:13px/1.5 system-ui;color:#202825;max-width:calc(100vw - 24px)}details{background:white;border:1px solid #c9d8cd;border-radius:10px;box-shadow:0 4px 20px #0001}summary{padding:10px 14px;cursor:pointer;color:#29473e;min-height:24px}.panel{display:grid;gap:10px;padding:4px 14px 14px;width:min(290px,calc(100vw - 56px))}label{display:grid;gap:4px}select,button{font:inherit;padding:8px;border:1px solid #d3dbd5;border-radius:6px;background:white;color:#202825;min-height:40px}button{cursor:pointer}small{color:#65706a}</style><details><summary>UI preview · sample data</summary><div class="panel"><small>Preview visual lokal. Pilih halaman atau peran. Tidak terhubung ke backend.</small><label>Halaman<select id="page"><option value="">Pilih halaman…</option></select></label><label>Peran<select id="persona"><option value="owner">Business owner</option><option value="customer">Customer</option><option value="staff">Counter staff</option><option value="guest">Guest</option></select></label><button id="reset">Reset data contoh</button></div></details>\`;
const config=await fetch('/__ui-preview/pages').then(r=>r.json());const select=root.getElementById('page');config.pages.forEach((p,i)=>{const o=document.createElement('option');o.value=String(i);o.textContent=p[0];select.append(o)});root.getElementById('persona').value=config.persona;
const post=(path,value)=>fetch('/__ui-preview/'+path,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(value)}).then(r=>r.json());
select.onchange=async()=>{if(select.value==='')return;const result=await post('select',{page:Number(select.value)});location.assign(result.path)};
root.getElementById('persona').onchange=async e=>{const result=await post('select',{persona:e.target.value});location.assign(result.path)};
root.getElementById('reset').onclick=async()=>{await post('reset',{});location.assign('/')};
`;
