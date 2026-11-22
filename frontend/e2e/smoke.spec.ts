import { test, expect, type Page } from '@playwright/test'
import { mockApi, login, FAKE_JWT } from './mockApi'

const SETTINGS = ['config', 'tasks', 'backup', 'logs', 'webhooks', 'apikeys', 'about']

/** Every static route, every view variant worth a look, and one route per dynamic entity. */
const ROUTES = [
  // static pages
  '/', '/?metric=views&compare=1', '/live', '/history', '/history?view=calendar', '/libraries', '/users',
  '/insights', '/discover', '/playback', '/nope',
  // users
  '/users/u2', '/users/u2?tab=history', '/users/u2?tab=taste&vs=u1', '/users/u1/recap/2026',
  // libraries and entities (fixture ids)
  '/libraries/lib-s', '/libraries/lib-s?tab=stats', '/libraries/lib-s?tab=activity', '/libraries/lib-s?layout=grid',
  '/movies/m1', '/movies/m1?tab=history', '/series/s1', '/series/s1?tab=history', '/series/s1/episodes/e5',
  '/artists/ar1', '/albums/al1', '/albums/al1?track=t3',
  // settings
  ...SETTINGS.map((s) => `/settings/${s}`),
]

/** Legacy URLs and their new homes. */
const REDIRECTS: [string, string][] = [
  ['/statistics', '/insights'], ['/activity', '/history'], ['/about', '/settings/about'], ['/settings', '/settings/config'],
  ['/libraries/lib-s/albums/al1', '/albums/al1'],
]

function watchErrors(page: Page) {
  const errors: string[] = []
  page.on('pageerror', (e) => errors.push(e.message))
  page.on('console', (m) => { if (m.type() === 'error') errors.push(m.text()) })
  return errors
}

async function expectNoOverflow(page: Page) {
  const overflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth)
  expect(overflow).toBeLessThanOrEqual(1)
}

for (const route of ROUTES) {
  test(`route ${route}: renders, no console error, no horizontal overflow`, async ({ page }) => {
    const unmocked = await mockApi(page)
    const errors = watchErrors(page)
    await login(page)
    await page.goto(route)
    await expect(page.locator('nav[aria-label]').first()).toBeAttached()
    if (route === '/nope') await expect(page.getByText(/page not found|page introuvable|no encontrada/i)).toBeVisible()
    else await expect(page.locator('main h1').first()).toBeAttached()
    if (route.startsWith('/settings/')) await expect(page.getByRole('tab', { selected: true })).toBeVisible()
    await page.waitForLoadState('networkidle')
    await expectNoOverflow(page)
    expect(unmocked).toEqual([])
    expect(errors).toEqual([])
  })
}

for (const [from, to] of REDIRECTS) {
  test(`redirect ${from} -> ${to}`, async ({ page }) => {
    await mockApi(page)
    await login(page)
    await page.goto(from)
    await expect(page).toHaveURL((u) => u.pathname === to)
  })
}

test.describe('login', () => {
  test('signs in and lands on the requested page', async ({ page }) => {
    await mockApi(page)
    const errors = watchErrors(page)
    await page.goto('/users?range=7d')
    await expect(page).toHaveURL((u) => u.pathname === '/login' && u.searchParams.get('next') === '/users?range=7d')
    const submit = page.getByRole('button', { name: /se connecter|sign in|iniciar/i })
    await expect(submit).toBeDisabled()
    await page.locator('#jf-user').fill('celine')
    await page.locator('#jf-pass').fill('secret')
    await submit.click()
    await expect(page).toHaveURL((u) => u.pathname === '/users' && u.searchParams.get('range') === '7d')
    expect(await page.evaluate(() => localStorage.getItem('jellystics-token'))).toBe(FAKE_JWT)
    await expectNoOverflow(page)
    expect(errors).toEqual([])
  })

  test('shows an error on bad credentials', async ({ page }) => {
    await mockApi(page)
    await page.route('**/auth/jellyfin-login', (r) => r.fulfill({ status: 401, contentType: 'application/json', body: '{}' }))
    await page.goto('/login')
    await page.locator('#jf-user').fill('celine')
    await page.locator('#jf-pass').fill('wrong')
    await page.getByRole('button', { name: /se connecter|sign in|iniciar/i }).click()
    await expect(page.getByRole('alert')).toBeVisible()
    await expect(page).toHaveURL((u) => u.pathname === '/login')
  })

  test('language switch updates the page and <html lang>', async ({ page }) => {
    await mockApi(page)
    await page.goto('/login')
    await expect(page.locator('#jf-user')).toBeVisible()
    await page.getByRole('combobox').click()
    await page.getByRole('option', { name: 'Español' }).click()
    await expect(page.locator('html')).toHaveAttribute('lang', 'es')
    await expect(page.getByRole('button', { name: /iniciar/i })).toBeVisible()
    await page.getByRole('combobox').click()
    await page.getByRole('option', { name: 'Français' }).click()
    await expect(page.locator('html')).toHaveAttribute('lang', 'fr')
  })
})
