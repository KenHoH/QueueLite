import { after, test } from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';
import { readFile } from 'node:fs/promises';
const vite = await createServer({ configFile: false, optimizeDeps: { noDiscovery: true, include: [] }, server: { middlewareMode: true, ws: false }, appType: 'custom' });
after(() => vite.close());
const { validateAPIResponse } = await vite.ssrLoadModule('/src/api/responseSchema.ts');
const { createAPIClient } = await vite.ssrLoadModule('/src/api/client.ts');

for (const [path, method] of [
 ['/users/me','GET'],['/businesses/','GET'],['/businesses/search','GET'],['/businesses/mine','GET'],['/businesses/id','GET'],
 ['/businesses/id/counters','GET'],['/businesses/id/members','GET'],['/businesses/id/queues','GET'],
 ['/queues/id','GET'],['/queues/id/state','GET'],['/queues/me','GET'],['/queues/business/id/join','POST'],
 ['/counters/id','GET'],['/counters/id/business/id/call-next','POST'],['/counters/id/queues/id/process','POST'],
 ['/subscriptions/users/id','GET'],['/subscriptions/businesses/id','GET'],['/users/id','PUT'],
]) test(`wire contract rejects malformed data for ${method} ${path}`, () => {
  for (const body of [null, {}, [null], 5, false, { id: 'id', name: {} }]) assert.throws(() => validateAPIResponse(path, method, body), e => e.code === 'CLIENT_INVALID_RESPONSE' && e.kind === 'response');
});

test('deadline aborts hanging reads and mutations once without retry', async () => {
 for (const method of ['GET','POST']) {
  let calls=0;
  const request=createAPIClient('/api',async (_,init)=> { calls++; return new Promise((_,reject)=>init.signal.addEventListener('abort',()=>reject(new DOMException('aborted','AbortError')))); },{timeoutMs:10});
  await assert.rejects(request('/queues/id',{method}),e=>e.code==='CLIENT_TIMEOUT' && e.kind==='network'); assert.equal(calls,1);
 }
});
test('caller cancellation remains distinguishable from deadline', async () => {
 const controller=new AbortController();
 const request=createAPIClient('/api',async (_,init)=>new Promise((_,reject)=>init.signal.addEventListener('abort',()=>reject(new DOMException('aborted','AbortError')))),{timeoutMs:1000});
 const pending=request('/queues/id',{signal:controller.signal}); controller.abort(); await assert.rejects(pending,e=>e.name==='AbortError');
});
test('malformed mutation JSON is ambiguous and preserves actual response status', async()=>{
 const request=createAPIClient('/api',async()=>Response.json({}, {status:201}),{validate:validateAPIResponse});
 await assert.rejects(request('/queues/business/id/join',{method:'POST'}),e=>e.status===201 && e.kind==='response');
});

test('small text colors retain sufficient contrast on both app surfaces',async()=>{
 const css=await readFile('src/design/tokens.css','utf8');
 const color=name=>css.match(new RegExp(`--ql-${name}: #([a-fA-F0-9]{6})`))[1];
 const luminance=hex=>{const rgb=hex.match(/../g).map(v=>parseInt(v,16)/255).map(v=>v<=.04045?v/12.92:((v+.055)/1.055)**2.4);return rgb[0]*.2126+rgb[1]*.7152+rgb[2]*.0722;};
 for(const foreground of ['text','text-secondary','action','error-text','priority-text'])for(const background of ['surface','background','soft-mint']){
  const f=luminance(color(foreground)),b=luminance(color(background));assert.ok((Math.max(f,b)+.05)/(Math.min(f,b)+.05)>=4.5,`${foreground} on ${background}`);
 }
});
