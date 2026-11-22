import type { Page } from '@playwright/test'

const b64 = (o: object) => Buffer.from(JSON.stringify(o)).toString('base64url')
export const FAKE_JWT = `${b64({ alg: 'HS256', typ: 'JWT' })}.${b64({ userId: 'u1', userName: 'Céline', isAdmin: true, sid: 's' })}.sig`

const API_PREFIX = /^\/(api|auth|stats|sessions|sync|backup|logs|utils|webhooks|proxy|socket\.io)(\/|$)/

// Realistic minimal shapes for endpoints whose consumers expect an object rather than a list.
const OBJECTS: Record<string, unknown> = {
  '/stats/getViewsByLibraryType': {},
  '/stats/getCompletionRate': { overall: { avgCompletionRate: 0 }, byType: [] },
  '/stats/getWatchHeatmap': { cells: [] },
  '/stats/getGlobalStats': { TotalPlays: 0, TotalWatchTime: 0, ActiveUsers: 0, TotalUsers: 0, TotalLibraries: 0, TotalItems: 0 },
  '/stats/getBingeStats': { totalBingeSessions: 0, topBingedSeries: [] },
  '/stats/getViewingDiversity': { uniqueGenres: 0, uniqueLibraries: 0, uniqueItems: 0, diversityScore: 0, genreBreakdown: [] },
  '/sessions/current': [],
  // --- phase 7 settings ---
  '/api/getconfig': { JF_HOST: 'http://jellyfin.test', app_url: '', settings: { KeepLogsForDays: 30 } },
  '/api/getTaskSettings': {},
  '/api/CheckForUpdates': { current_version: '1.0.0', latest_version: 'v1.0.0', update_available: false, message: 'Up to date' },
  // --- end phase 7 settings ---
}

const json = (body: unknown, status = 200) => ({ status, contentType: 'application/json', body: JSON.stringify(body) })

/** Answers every backend call with an empty-but-valid payload. */
export async function mockApi(page: Page) {
  await page.route((url) => API_PREFIX.test(url.pathname), async (route) => {
    const req = route.request()
    const { pathname } = new URL(req.url())
    if (req.resourceType() === 'document') return route.continue()
    if (pathname === '/auth/isConfigured') return route.fulfill(json({ state: 1, timezone: 'Europe/Paris', jellyfinUrl: 'http://jellyfin.test' }))
    if (pathname === '/auth/check-server') return route.fulfill(json({ serverName: 'Jellyfin', url: 'http://jellyfin.test' }))
    if (pathname === '/auth/jellyfin-login') return route.fulfill(json({ token: FAKE_JWT }))
    if (pathname.startsWith('/socket.io')) return route.fulfill({ status: 200, contentType: 'text/plain', body: '' })
    if (pathname in OBJECTS) return route.fulfill(json(OBJECTS[pathname]))
    return route.fulfill(json([]))
  })
}

/** Puts a fake session token in localStorage before any page script runs. */
export async function login(page: Page) {
  await page.addInitScript((token) => { localStorage.setItem('jellystics-token', token) }, FAKE_JWT)
}
