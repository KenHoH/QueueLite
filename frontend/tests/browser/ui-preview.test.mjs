import { test } from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';
import { chromium } from 'playwright';

test('standalone visual preview renders every page on desktop and phone without API interception', { timeout: 180000 }, async () => {
  const vite = await createServer({ mode: 'ui-preview', server: { host: '127.0.0.1', port: 0, ws: false } });
  let browser;
  try {
    await vite.listen();
    const origin = `http://127.0.0.1:${vite.httpServer.address().port}`;
    browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_EXECUTABLE_PATH ? { executablePath: process.env.PLAYWRIGHT_EXECUTABLE_PATH } : process.env.PLAYWRIGHT_CHANNEL ? { channel: process.env.PLAYWRIGHT_CHANNEL } : {}) });
    const context = await browser.newContext();
    const page = await context.newPage();
    const errors = [];
    page.on('pageerror', e => errors.push(e.message));
    await page.goto(origin);
    await page.locator('#ql-ui-preview').waitFor();
    const { pages } = await context.request.get(origin + '/__ui-preview/pages').then(r => r.json());
    for (const width of [1440, 390]) {
      await page.setViewportSize({ width, height: 900 });
      for (let index = 0; index < pages.length; index++) {
        const [label] = pages[index];
        const { path } = await context.request.post(origin + '/__ui-preview/select', { data: { page: index } }).then(r => r.json());
        await page.goto(origin + path);
        await page.locator('main h1').waitFor();
        await page.locator('[role="status"][aria-label]').waitFor({ state: 'hidden' });
        await page.waitForFunction(() => !document.querySelector('.ql-skeleton'));
        assert.equal(await page.locator('main [role="alert"]').count(), 0, `${label} at ${width}px has an application error`);
        assert.ok(!/Permission required|Sign in to continue|unexpected response|couldn.t load/i.test(await page.locator('main').innerText()), `${label} at ${width}px did not load`);
        assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1), `${label} overflows at ${width}px`);
      }
    }
    assert.deepEqual(errors, []);
    await context.close();
  } finally { await browser?.close(); await vite.close(); }
});
