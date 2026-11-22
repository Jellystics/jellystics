import { test, expect } from '@playwright/test'
import { mockApi, login } from './mockApi'

const ROUTES = ['/', '/live', '/history', '/history?view=calendar', '/libraries', '/users', '/insights', '/discover', '/playback', '/settings/config', '/settings/about', '/nope']

test.beforeEach(async ({ page }) => { await mockApi(page) })

test('login page renders without errors', async ({ page }) => {
  const errors: string[] = []; page.on('pageerror', (e) => errors.push(e.message))
  await page.goto('/login')
  await expect(page.getByRole('button', { name: /se connecter|sign in|iniciar/i })).toBeVisible()
  expect(errors).toEqual([])
})

for (const route of ROUTES) {
  test(`route ${route}: shell, no console error, no horizontal overflow`, async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', (e) => errors.push(e.message))
    page.on('console', (m) => { if (m.type() === 'error') errors.push(m.text()) })
    await login(page)
    await page.goto(route)
    await expect(page.locator('nav[aria-label]').first()).toBeAttached()
    await page.waitForLoadState('networkidle')
    const overflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth)
    expect(overflow).toBeLessThanOrEqual(1)
    expect(errors).toEqual([])
  })
}

// --- phase 7 settings ---
for (const section of ['tasks', 'backup', 'logs', 'webhooks', 'apikeys']) {
  test(`settings section ${section}: no console error, no horizontal overflow`, async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', (e) => errors.push(e.message))
    page.on('console', (m) => { if (m.type() === 'error') errors.push(m.text()) })
    await login(page)
    await page.goto(`/settings/${section}`)
    await expect(page.getByRole('tab', { selected: true })).toBeVisible()
    await page.waitForLoadState('networkidle')
    const overflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth)
    expect(overflow).toBeLessThanOrEqual(1)
    expect(errors).toEqual([])
  })
}
// --- end phase 7 settings ---
