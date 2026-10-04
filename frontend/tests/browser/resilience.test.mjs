import { before, after, test } from 'node:test';
import assert from 'node:assert/strict';
import { mkdir } from 'node:fs/promises';
import { createServer } from 'vite';
import { chromium } from 'playwright';

const business = { id: '11111111-1111-4111-8111-111111111111', name: 'Klinik Ayu', location: 'Jakarta', operational: true, openTime: '22:00', closeTime: '06:00', email: 'ayu@example.test', phoneNumber: '081234567890', role: 'owner' };
const user = { id: '22222222-2222-4222-8222-222222222222', username: 'Ayu', phonenumber: '081234567890' };
const counter = { id: '33333333-3333-4333-8333-333333333333', businessId: business.id, name: 'Meja Ayu', currentEmployeeId: user.id };
const queue = { id: '44444444-4444-4444-8444-444444444444', businessId: business.id, name: 'A001', state: 'waiting', priority: false };
let vite, browser, origin;
const visible = locator => locator.waitFor({ state: 'visible' });
const pause = ms => new Promise(resolve => setTimeout(resolve, ms));
before(async () => {
  vite = await createServer({ server: { host: '127.0.0.1', port: 0, open: false, ws: false } }); await vite.listen();
  origin = `http://127.0.0.1:${vite.httpServer.address().port}`;
  browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_CHANNEL ? { channel: process.env.PLAYWRIGHT_CHANNEL } : {}) });
  await mkdir('output/playwright', { recursive: true });
});
after(async () => { await browser?.close(); await vite?.close(); });
async function scenario(t, options = {}) {
  const context = await browser.newContext({ viewport: { width: 1280, height: 900 } });
  const page = await context.newPage(); page.setDefaultTimeout(6000);
  const errors = [], requests = [];
  page.on('pageerror', error => errors.push(error.message));
  page.on('console', message => {
    if (['error', 'warning'].includes(message.type()) && !(message.text().startsWith('Failed to load resource:') && message.location().url.startsWith(`${origin}/api/`))) errors.push(message.text());
  });
  t.after(async () => { try { assert.deepEqual(errors, [], 'runtime errors/warnings'); } finally { await context.close(); } });
  await context.route(`${origin}/api/**`, async route => {
    const request = route.request(), path = new URL(request.url()).pathname, method = request.method();
    requests.push({ path, method, body: request.postDataJSON() });
    if (await options.intercept?.(route, path, request)) return;
    let response = await options.respond?.(path, request);
    if (!response) {
      if (path === '/api/users/me') response = options.guest ? { status: 401, json: {} } : { json: user };
      else if (path === '/api/users/logout' || path === '/api/users/login') response = { json: { message: 'ok' } };
      else if (path === '/api/users/' && method === 'POST') response = { status: 201, json: user };
      else if (path === '/api/businesses/mine') response = { json: options.businesses ?? [business] };
      else if (path === '/api/businesses/' && method === 'GET') response = { json: { data: [business], nextCursor: null } };
      else if (path.endsWith('/search')) response = { json: [business] };
      else if (path.endsWith('/join')) response = { status: 201, json: queue };
      else if (path.endsWith('/counters')) response = { json: [counter] };
      else if (path.endsWith('/members')) response = { json: [{ userId: user.id, username: user.username, role: 'owner' }] };
      else if (path.endsWith('/queues') || path === '/api/queues/me') response = { json: [queue] };
      else if (path.startsWith('/api/subscriptions/')) response = { json: { subscription: { type: path.includes('/users/') ? 'user' : 'business', status: 'active', startDate: '2026-10-01T00:00:00Z' }, userPlan: { userPlanType: 'standard', slots: 0 }, businessPlan: { businessPlanType: 'free', capacity: 49 } } };
      else if (path.startsWith('/api/businesses/')) response = { json: business };
      else if (path.startsWith('/api/counters/')) response = { json: counter };
      else if (path.endsWith('/state') && method === 'GET') response = { json: { state: queue.state } };
      else if (path.startsWith('/api/queues/')) response = { json: queue };
      else response = { json: { message: 'ok' } };
    }
    try { await route.fulfill(response); } catch (error) { if (!/closed|cancel|abort|Invalid Interception/i.test(error.message)) throw error; }
  });
  return { page, context, requests };
}

test('malformed current-user object remains a recoverable session error', async t => {
  const { page } = await scenario(t, { respond: path => path === '/api/users/me' ? { json: { id: user.id } } : undefined });
  await page.goto(origin + '/profile'); await visible(page.getByRole('button', { name: /Retry session/ }));
  assert.equal(await page.getByLabel('Phone number', { exact: true }).count(), 0);
});
test('malformed plans remain a recoverable read error', async t => {
  const { page } = await scenario(t, { respond: path => path.startsWith('/api/subscriptions/') ? { json: { subscription: {} } } : undefined });
  await page.goto(origin + '/plans'); await visible(page.getByRole('button', { name: 'Try again' }).first());
});
test('malformed join success never claims a ticket was accepted', async t => {
  const { page, requests } = await scenario(t, { respond: path => path.endsWith('/join') ? { status: 201, json: {} } : undefined });
  await page.goto(origin + `/business/${business.id}/join`); await page.getByRole('button', { name: 'Join queue', exact: true }).click();
  await visible(page.getByText(/before joining again/)); assert.equal(await page.getByRole('button', { name: 'Join queue', exact: true }).isDisabled(), true);
  assert.equal(requests.filter(r => r.method === 'POST').length, 1);
});
test('duplicate discovery IDs within and across pages never produce duplicate cards', async t => {
  let reads = 0;
  const { page } = await scenario(t, { respond: path => path === '/api/businesses/' ? { json: { data: [business, business], nextCursor: ++reads === 1 ? { createdAt: '2026-10-01T00:00:00Z', id: business.id } : null } } : undefined });
  await page.goto(origin); await visible(page.getByRole('heading', { name: business.name }));
  assert.equal(await page.locator('.ql-business-card').count(), 1);
  await page.getByRole('button', { name: 'Load more' }).click(); await page.getByRole('button', { name: 'Load more' }).waitFor({ state: 'hidden' });
  assert.equal(await page.locator('.ql-business-card').count(), 1);
});

const routes = [
 ['login','/login',true], ['register','/register',true], ['home','/'],
 ['detail',`/business/${business.id}`], ['join',`/business/${business.id}/join`], ['ticket',`/queue/${queue.id}`],
 ['my-queues','/my-queues'], ['profile','/profile'], ['plans','/plans'], ['create','/business/create'], ['manage','/business/manage'],
 ['dashboard',`/business/${business.id}/dashboard`], ['counters',`/business/${business.id}/counters`],
 ['workspace',`/counter/${counter.id}`], ['settings',`/business/${business.id}/settings`], ['business-plans',`/business/${business.id}/plans`],
];
async function ready(page, guest=false) {
  await visible(page.getByRole(guest ? 'link' : 'button', { name: guest ? 'Create account' : 'Logout', exact: true }));
  await page.locator('.ql-skeleton').first().waitFor({ state:'hidden' });
  await visible(page.locator('main'));
}
for (const [name,path,guest] of routes) test(`fresh context, refresh, back and forward: ${name}`,async t=>{
 const {page}=await scenario(t,{guest}); await page.goto(origin+path); await ready(page,guest);
 const before=await page.locator('main').innerText(); assert.ok(before.trim().length>15); assert.doesNotMatch(before,/undefined|\[object Object\]|TypeError|stack trace/);
 await page.reload(); await ready(page,guest); assert.equal(new URL(page.url()).pathname,path);
 if(path==='/') await page.getByRole('link',{name:'My Queues',exact:true}).click();
 else await page.getByRole('link',{name:'QueueLite',exact:true}).click(); await ready(page,guest);
 await page.goBack(); await ready(page,guest); assert.equal(new URL(page.url()).pathname,path);
 await page.goForward(); await ready(page,guest); assert.equal(new URL(page.url()).pathname,path==='/'?'/my-queues':'/');
});
for (const path of ['/profile','/my-queues','/plans','/business/create','/business/manage',`/business/${business.id}/settings`,`/business/${business.id}/dashboard`,`/business/${business.id}/counters`,`/counter/${counter.id}`]) test(`fresh guest cannot request protected resources: ${path}`,async t=>{
 const {page,requests}=await scenario(t,{guest:true}); await page.goto(origin+path); await ready(page,true);
 await visible(page.getByRole('heading',{name:/Sign in to/})); assert.ok(requests.every(r=>r.path==='/api/users/me'));
});
test('invalid and nonexistent resource URLs fail without stale data',async t=>{
 const missing='99999999-9999-4999-8999-999999999999';
 const {page}=await scenario(t,{respond:path=>path.includes(missing)?{status:404,json:{}}:path.includes('not-a-uuid')?{status:400,json:{}}:undefined});
 for(const path of ['/unrecognized/path',...['not-a-uuid',missing].flatMap(id=>[`/business/${id}`,`/business/${id}/join`,`/queue/${id}`,`/counter/${id}`,`/business/${id}/settings`,`/business/${id}/dashboard`,`/business/${id}/counters`,`/business/${id}/plans`])]){
  await page.goto(origin+path); await ready(page); const content=await page.locator('main').innerText();
  assert.match(content,/not found|could not be found|not available|invalid|Permission required/i,`${path}: ${content}`);
  assert.equal(await page.getByRole('button',{name:/Join queue|Call next|Save business settings/}).count(),0);
 }
});

const reads = [
 ['home','/','/api/businesses/'],['detail',`/business/${business.id}`,`/api/businesses/${business.id}`],
 ['join',`/business/${business.id}/join`,`/api/businesses/${business.id}`], ['my-queues','/my-queues','/api/queues/me'],
 ['plans','/plans',`/api/subscriptions/users/${user.id}`],['ticket',`/queue/${queue.id}`,`/api/queues/${queue.id}`],
 ['dashboard',`/business/${business.id}/dashboard`,`/api/businesses/${business.id}/queues`],
 ['counters',`/business/${business.id}/counters`,`/api/businesses/${business.id}/members`],['workspace',`/counter/${counter.id}`,`/api/counters/${counter.id}`],
];
for(const [name,path,endpoint] of reads) for(const failure of [400,401,403,404,409,429,500,'offline','json','shape']) test(`${name} handles read ${failure} and explicit recovery`,async t=>{
 let fail=true;
 const {page,requests}=await scenario(t,{intercept:async(route,p)=>{if(p===endpoint && fail && failure==='offline'){await route.abort('failed');return true;}},respond:p=>p===endpoint && fail?(typeof failure==='number'?{status:failure,json:{error:{code:'QA_ERROR',message:'PRIVATE DATABASE STACK TRACE'}}}:failure==='json'?{body:'{broken json',contentType:'application/json'}:failure==='shape'?{json:{unexpected:true}}:undefined):undefined});
 await page.goto(origin+path); await ready(page); const content=await page.locator('main').innerText(); assert.doesNotMatch(content,/PRIVATE DATABASE|\[object Object\]|TypeError/);
 assert.ok(await page.getByRole('alert').count()>0 || /Sign in to|not found/i.test(content),`${name}: expected failure feedback`);
 const count=requests.filter(r=>r.path===endpoint).length; await pause(70); assert.equal(requests.filter(r=>r.path===endpoint).length,count);
 fail=false;
 const retry=page.getByRole('button',{name:/Try again|Refresh ticket|Refresh state/}).first();
 if(await retry.count()) { await retry.click(); await ready(page); assert.ok(requests.filter(r=>r.path===endpoint).length>count); }
 else { await page.reload(); await ready(page); }
});

for(const width of [375,390,768,1024,1280,1440,1920]) test(`all major pages fit ${width}px with accessible controls and screenshots`,async t=>{
 const account=await scenario(t),guest=await scenario(t,{guest:true});
 for(const [name,path,isGuest] of routes){
  const page=isGuest?guest.page:account.page; await page.setViewportSize({width,height:900}); await page.goto(origin+path); await ready(page,isGuest);
  const issues=await page.evaluate(()=>{
   const errors=[]; if(document.documentElement.scrollWidth>innerWidth+1) errors.push('document overflow');
   for(const element of document.querySelectorAll('main button,main input,main select,main textarea')){
    const rect=element.getBoundingClientRect(); if(rect.width && (rect.left< -1 || rect.right>innerWidth+1)) errors.push('clipped '+element.outerHTML.slice(0,130));
    if(element.matches('input,select,textarea') && !element.labels?.length && !element.getAttribute('aria-label') && !element.getAttribute('aria-labelledby')) errors.push('unlabelled '+element.tagName);
   } return errors;
  }); assert.deepEqual(issues,[],`${name} at ${width}px`);
  await page.screenshot({path:`output/playwright/resilience-${name}-${width}.png`,fullPage:true});
 }
 await account.page.goto(origin+`/business/${business.id}/join`); await ready(account.page); await account.page.getByRole('button',{name:'Join queue',exact:true}).click(); await visible(account.page.getByRole('heading',{name:'Your place is saved.'}));
 await account.page.screenshot({path:`output/playwright/resilience-join-success-${width}.png`,fullPage:true});
});

test('successful logout, history and direct protected reload clear account data',async t=>{
 let loggedIn=true;
 const {page,requests}=await scenario(t,{respond:path=>path==='/api/users/me'?{status:loggedIn?200:401,json:loggedIn?user:{}}:path==='/api/users/logout'?(loggedIn=false,{json:{message:'ok'}}):undefined});
 await page.goto(origin+'/profile'); await ready(page); await page.getByRole('button',{name:'Logout'}).click(); await ready(page,true);
 await page.goBack(); await visible(page.getByRole('heading',{name:'Sign in to continue.'}));
 const before=requests.length; await page.goto(origin+'/profile'); await ready(page,true); assert.ok(requests.slice(before).every(r=>r.path==='/api/users/me'));
 assert.deepEqual(await page.evaluate(()=>({local:{...localStorage},session:{...sessionStorage}})),{local:{},session:{}});
});
test('failed logout preserves authenticated state and shows retryable error',async t=>{
 const {page}=await scenario(t,{respond:path=>path==='/api/users/logout'?{status:500,json:{}}:undefined}); await page.goto(origin+'/profile'); await ready(page);
 await page.getByRole('button',{name:'Logout'}).click(); await visible(page.getByText('We could not sign you out. Please try again.')); assert.equal(await page.getByLabel('Username').inputValue(),user.username);
});
test('separate guest and account contexts never share account state',async t=>{
 const account=await scenario(t),guest=await scenario(t,{guest:true}); await account.page.goto(origin+'/profile');await ready(account.page);await guest.page.goto(origin+'/profile');await ready(guest.page,true);
 assert.equal(await guest.page.getByLabel('Username').count(),0);assert.ok(guest.requests.every(r=>r.path==='/api/users/me'));
});
test('protected-route sign-in returns to intended business destination',async t=>{
 let signedIn=false;
 const {page}=await scenario(t,{guest:true,respond:path=>path==='/api/users/me'?{status:signedIn?200:401,json:signedIn?user:{}}:path==='/api/users/login'?(signedIn=true,{json:{message:'ok'}}):undefined});
 const destination=`/business/${business.id}/counters`;await page.goto(origin+destination);await visible(page.getByRole('heading',{name:'Sign in to continue.'}));await page.locator('main').getByRole('link',{name:'Sign in',exact:true}).click();
 await page.getByLabel('Username').fill('Ayu');await page.getByLabel('Password',{exact:true}).fill('valid password');await page.getByRole('button',{name:'Sign in',exact:true}).click();await ready(page);
 assert.equal(new URL(page.url()).pathname,destination);await visible(page.getByRole('heading',{name:'Create counter',exact:true}));
});

const forms=[
 ['login','/login','/api/users/login','Sign in',true],['register','/register','/api/users/','Create account',true],
 ['guest-join',`/business/${business.id}/join`,`/api/queues/business/${business.id}/join`,'Join queue',true],
 ['profile','/profile',`/api/users/${user.id}`,'Save profile'],['password','/profile',`/api/users/${user.id}`,'Change password'],
 ['create','/business/create','/api/businesses/','Create business'],['settings',`/business/${business.id}/settings`,`/api/businesses/${business.id}`,'Save business settings'],
 ['counter-create',`/business/${business.id}/counters`,'/api/counters/','Create counter'],['counter-update',`/business/${business.id}/counters`,`/api/counters/${counter.id}`,'Save counter'],
];
async function fillForm(page,name){
 const fill=async(label,value)=>page.getByLabel(label,{exact:true}).fill(value);
 if(['login','register'].includes(name)) { await fill('Username',"  Ayu O’Connor-李  ");await fill('Password','valid password'); }
 if(name==='register'){await fill('Phone number','+62 812-3456-7890');await fill('Email (optional)','ayu@example.test');await fill('Confirm password','valid password');}
 if(name==='guest-join'){await fill('Name',"  Ayu O’Connor-李  ");await fill('Phone number','+62 812-3456-7890');}
 if(name==='profile'){await fill('Phone number','+62 812-3456-7890');await fill('Email (optional)','new@example.test');}
 if(name==='password'){await fill('New password','valid password');await fill('Confirm new password','valid password');}
 if(['create','settings'].includes(name))for(const [label,value] of Object.entries({'Business name':"  Klinik O’Connor-李  ",Location:'Jakarta','Open time':'22:00','Close time':'06:00','Business email':'ayu@example.test','Business phone number':'+62 812-3456-7890'}))await fill(label,value);
 if(name==='counter-create')await fill('Counter name',"  Meja O’Connor-李  ");
 if(name==='counter-update')await fill(`Name for ${counter.name}`,"  Meja O’Connor-李  ");
}
for(const [name,path,endpoint,button,guest] of forms)for(const failure of [400,401,403,404,409,500,'offline','json','shape'])test(`${name} preserves input and rejects mutation ${failure}`,async t=>{
 const {page,requests}=await scenario(t,{guest,intercept:async(route,p,request)=>{if(p===endpoint && request.method()!=='GET' && failure==='offline'){await route.abort('failed');return true;}},respond:(p,request)=>p===endpoint && request.method()!=='GET'?(typeof failure==='number'?{status:failure,json:{error:{code:'QA_ERROR',message:'PRIVATE DATABASE STACK TRACE'}}}:failure==='json'?{body:'not json',contentType:'application/json'}:failure==='shape'?{json:{unexpected:true}}:undefined):undefined});
 await page.goto(origin+path);await ready(page,guest);await fillForm(page,name);
 const form=page.getByRole('button',{name:button,exact:true}).locator('..').filter({has:page.locator('input')});
 const values=await page.locator('main input:not([readonly])').evaluateAll(inputs=>inputs.map(input=>input.value));
 await page.getByRole('button',{name:button,exact:true}).click();await visible(page.getByRole('alert').first()); await pause(60);
 assert.equal(requests.filter(r=>r.path===endpoint && r.method!=='GET').length,1);
 assert.doesNotMatch(await page.locator('main').innerText(),/PRIVATE DATABASE|Your place is saved|Profile saved|Password changed|settings saved/);
 assert.deepEqual(await page.locator('main input:not([readonly])').evaluateAll(inputs=>inputs.map(input=>input.value)),values,`${name}: input lost`);
});
for(const [name,path,endpoint,button,guest] of forms)test(`${name} validates blank inputs and locks repeated Enter submission`,async t=>{
 let release;const held=new Promise(resolve=>release=resolve);
 const {page,requests}=await scenario(t,{guest,respond:async(p,request)=>{if(p===endpoint && request.method()!=='GET'){await held;return {status:409,json:{}};}}});
 await page.goto(origin+path);await ready(page,guest);
 const target=page.getByRole('button',{name:button,exact:true});
 // Settings/profile/editor start filled; deliberately blank their required editable field.
 const blank=name==='profile'?'Phone number':name==='settings'?'Business name':name==='counter-update'?`Name for ${counter.name}`:undefined;
 if(blank)await page.getByLabel(blank,{exact:true}).fill('   ');
 await target.click();await visible(page.getByRole('alert').first());assert.equal(requests.filter(r=>r.method!=='GET').length,0);
 await fillForm(page,name);const enclosing=page.locator('form').nth(name==='password'||name==='counter-update'?1:0);
 await enclosing.evaluate(form=>{form.requestSubmit();form.requestSubmit();form.requestSubmit();});await pause(80);
 assert.equal(requests.filter(r=>r.path===endpoint && r.method!=='GET').length,1);assert.equal(await enclosing.locator('button[type=submit]').isDisabled(),true);release();await visible(page.getByRole('alert').first());
});
test('registration rejects malformed phone, whitespace password and excessive UTF-8 password bytes',async t=>{
 const {page,requests}=await scenario(t,{guest:true});await page.goto(origin+'/register');await ready(page,true);await fillForm(page,'register');
 for(const [label,value] of [['Phone number','abc'],['Password','   '],['Password','é'.repeat(37)]]){
  await fillForm(page,'register');await page.getByLabel(label,{exact:true}).fill(value);await page.getByRole('button',{name:'Create account',exact:true}).click();await visible(page.getByRole('alert').first());
  assert.equal(requests.filter(r=>r.method==='POST').length,0);
 }
});

for(const [name,path,endpoint,button,guest] of forms)test(`${name} handles very long input without overflow or losing rejected values`,async t=>{
 const {page,requests}=await scenario(t,{guest,respond:(p,request)=>p===endpoint && request.method()!=='GET'?{status:400,json:{error:{code:'INVALID_FORMAT',message:'invalid input'}}}:undefined});
 await page.setViewportSize({width:375,height:900});await page.goto(origin+path);await ready(page,guest);await fillForm(page,name);
 const label=['login','register'].includes(name)?'Username':name==='guest-join'?'Name':name==='profile'?'Phone number':name==='password'?'New password':['create','settings'].includes(name)?'Business name':name==='counter-create'?'Counter name':`Name for ${counter.name}`;
 const input=page.getByLabel(label,{exact:true});const value="Ayu O’Connor-李 ".repeat(700);await input.fill(value);
 const accepted=await input.inputValue(),maxLength=await input.getAttribute('maxlength');
 assert.equal(accepted,maxLength?value.slice(0,Number(maxLength)):value);
 if(['create','settings'].includes(name))await page.getByLabel('Description (optional)',{exact:true}).fill(value);
 await page.getByRole('button',{name:button,exact:true}).click();await visible(page.getByRole('alert').first());
 assert.equal(await input.inputValue(),accepted);assert.ok(requests.filter(r=>r.path===endpoint && r.method!=='GET').length<=1);
 assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1),true);
});
test('keyboard focus, error association and leave confirmation are usable',async t=>{
 const {page}=await scenario(t,{guest:true});await page.goto(origin+'/register');await ready(page,true);
 await page.getByRole('button',{name:'Create account',exact:true}).click();const username=page.getByLabel('Username');await visible(username);
 assert.equal(await username.getAttribute('aria-invalid'),'true');assert.ok(await username.getAttribute('aria-describedby'));assert.equal(await username.evaluate(input=>input===document.activeElement),true);
 await page.keyboard.press('Tab');assert.ok(await page.evaluate(()=>getComputedStyle(document.activeElement).outlineStyle!=='none'));
 await page.goto(origin+`/queue/${queue.id}`);await ready(page,true);await page.getByRole('button',{name:'Leave queue',exact:true}).focus();await page.keyboard.press('Enter');
 await visible(page.getByRole('alertdialog'));await page.keyboard.press('Escape');await page.getByRole('alertdialog').waitFor({state:'hidden'});
});
test('slow auth has understandable loading and times out with manual recovery',async t=>{
 const {page}=await scenario(t,{respond:async path=>{if(path==='/api/users/me'){await pause(21000);return {json:user};}}});
 await page.clock.install();await page.goto(origin+'/profile');await visible(page.getByText('Checking your account',{exact:true}));await page.clock.fastForward(20001);await visible(page.getByRole('button',{name:'Retry session check'}));
});
