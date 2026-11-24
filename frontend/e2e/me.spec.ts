import { existsSync, readFileSync } from 'node:fs'
import { extname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { test, expect, type Page, type Route } from '@playwright/test'
import { meResponse, ME_USER_ID } from '../src/me/fixtures'

// The « Mes statistiques » page as the Jellystics plugin serves it: dist/me.html and its relative assets under
// <base>/Jellystics/Me/, the API relayed under <base>/Jellystics/Me/api/. Here base = /jellyfin.
const BASE = '/jellyfin'
const ME = `${BASE}/Jellystics/Me/`
const SHOTS = process.env.ME_SHOTS_DIR
// Files come straight from the build (the webServer builds it), like the plugin proxies them from Jellystics /me/.
const DIST = fileURLToPath(new URL('../../dist', import.meta.url))
const TYPES: Record<string, string> = { '.html': 'text/html', '.js': 'text/javascript', '.css': 'text/css', '.json': 'application/json', '.png': 'image/png', '.svg': 'image/svg+xml' }
const file = (rel: string) => {
  const path = resolve(DIST, rel)
  if (!path.startsWith(DIST) || !existsSync(path)) return { status: 404, body: '' }
  return { status: 200, contentType: TYPES[extname(path)] ?? 'application/octet-stream', body: readFileSync(path) }
}

const CSP = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'"

type Api = (path: string, query: URLSearchParams) => { status: number; body?: unknown }

async function serveMe(page: Page, api: Api = (p, q) => ({ status: 200, body: meResponse(p, q) })) {
  const seen: string[] = []
  await page.route(`**${BASE}/**`, async (route: Route) => {
    const url = new URL(route.request().url())
    const p = url.pathname
    if (p.startsWith(`${ME}api/`)) {
      const path = p.slice(`${ME}api/`.length)
      seen.push(`${path}${url.search}|${route.request().headers().authorization ?? ''}`)
      const r = api(path, url.searchParams)
      return route.fulfill({ status: r.status, contentType: 'application/json', body: r.body === undefined ? '' : JSON.stringify(r.body) })
    }
    // /Jellystics/Me → /Jellystics/Me/ is a server-side redirect of the plugin (me.html has no inline script).
    if (p.startsWith(ME)) {
      const rest = p.slice(ME.length) || 'me.html'
      const r = file(rest)
      // Strict CSP: the page must work without inline scripts.
      return route.fulfill(r.contentType === 'text/html' ? { ...r, headers: { 'content-security-policy': CSP } } : r)
    }
    // Jellyfin images: answer with the Jellystics logo (any PNG will do).
    if (/\/Images\/Primary$/.test(p)) return route.fulfill(file('logo-64.png'))
    return route.fulfill({ status: 404, body: '' })
  })
  return seen
}

async function signIn(page: Page) {
  await page.addInitScript((userId) => {
    localStorage.setItem('jellyfin_credentials', JSON.stringify({
      Servers: [
        { Id: 'other', AccessToken: 'wrong', UserId: 'x', ManualAddress: 'http://elsewhere:8096', DateLastAccessed: 9e12 },
        { Id: 'srv', AccessToken: 'tok-123', UserId: userId, ManualAddress: `${location.origin}/jellyfin`, DateLastAccessed: 1 },
      ],
    }))
  }, ME_USER_ID)
}

function watchErrors(page: Page) {
  const errors: string[] = []
  page.on('pageerror', (e) => errors.push(e.message))
  page.on('console', (m) => { if (m.type() === 'error') errors.push(m.text()) })
  return errors
}

async function expectNoOverflow(page: Page) {
  expect(await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth)).toBeLessThanOrEqual(1)
}

test('me page: renders every section under the plugin path, no console error, no horizontal overflow', async ({ page }, info) => {
  const seen = await serveMe(page)
  await signIn(page)
  const errors = watchErrors(page)
  await page.goto(ME)
  await expect(page.getByRole('heading', { level: 2, name: 'Alice' })).toBeVisible()
  await expect(page.getByRole('link', { name: /Jellyfin/ }).first()).toHaveAttribute('href', `${BASE}/web/`)
  await expect(page.getByText('Les Rivages').first()).toBeVisible()
  await expect(page.getByText('Episode 1', { exact: true })).toBeVisible()
  await expect(page.getByText('Living room TV')).toBeVisible()
  await page.waitForLoadState('networkidle')
  await expectNoOverflow(page)
  expect(await page.evaluate(() => document.documentElement.dataset.theme)).toBe(info.project.use.colorScheme === 'light' ? 'light' : 'dark')
  for (const s of seen) {
    expect(s).toMatch(/\|MediaBrowser Token="tok-123"$/)
    expect(s.toLowerCase()).not.toContain('user')
  }
  expect(errors).toEqual([])
  if (SHOTS) await page.screenshot({ path: `${SHOTS}/me-${info.project.name}.png`, fullPage: true })

  await page.getByRole('button', { name: /next page|page suivante|siguiente/i }).click()
  await expect(page.getByText('Episode 27', { exact: true })).toBeVisible()
  await expectNoOverflow(page)
})

test('me page: unavailable state when Jellystics is unreachable', async ({ page }, info) => {
  await serveMe(page, () => ({ status: 502, body: { error: 'unreachable' } }))
  await signIn(page)
  await page.goto(ME)
  await expect(page.getByRole('alert')).toBeVisible()
  await expectNoOverflow(page)
  if (SHOTS) await page.screenshot({ path: `${SHOTS}/me-unavailable-${info.project.name}.png` })
})

test('me page: asks to sign in to Jellyfin without credentials', async ({ page }) => {
  const seen = await serveMe(page)
  await page.goto(ME)
  await expect(page.getByRole('link', { name: /Jellyfin/ }).last()).toHaveAttribute('href', `${BASE}/web/`)
  await expect(page.getByRole('alert')).toBeVisible()
  expect(seen).toEqual([])
})

test('me page: works under a strict CSP (no inline script)', async ({ page }) => {
  const errors = watchErrors(page)
  const csp: string[] = []
  page.on('console', (m) => { if (/Content Security Policy/i.test(m.text())) csp.push(m.text()) })
  await serveMe(page)
  await signIn(page)
  const res = page.waitForResponse((r) => r.url().endsWith(ME))
  await page.goto(ME)
  expect((await res).headers()['content-security-policy']).toContain("script-src 'self'")
  await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
  await expect(page.getByRole('heading', { level: 2, name: 'Alice' })).toBeVisible()
  expect(csp).toEqual([])
  expect(errors).toEqual([])
})
