import type { Page } from '@playwright/test'
import { entityApi, libraryHistoryFixture } from '../src/app/pages/library/fixtures' // phase 4

const b64 = (o: object) => Buffer.from(JSON.stringify(o)).toString('base64url')
export const FAKE_JWT = `${b64({ alg: 'HS256', typ: 'JWT' })}.${b64({ userId: 'u1', userName: 'Céline', isAdmin: true, sid: 's' })}.sig`

const API_PREFIX = /^\/(api|auth|stats|sessions|sync|backup|logs|utils|webhooks|proxy|socket\.io)(\/|$)/

// --- phase 5 ---
const hours24 = (peak: number, n: number) => Array.from({ length: 24 }, (_, h) => Math.max(0, n - Math.abs(h - peak) * 2))
const PHASE5_METHODS_DAILY = Array.from({ length: 30 }, (_, i) => ({ date: `2026-10-${String(i + 1).padStart(2, '0')}`, direct: 6 + (i % 4), stream: 2, transcode: 1 + (i % 3) }))
const PHASE5: Record<string, unknown> = {
  '/api/v2/insights/completion': { distribution: [21, 9, 6, 4, 4, 4, 5, 7, 13, 124], byType: { movie: 0.64, series: 0.81, music: 0.73 }, units: 197 },
  '/api/v2/insights/sessions': { buckets: [41, 26, 19, 34, 61, 52, 30, 21, 9], sessions: 293, medianSeconds: 2460 },
  '/api/v2/insights/resumes': { fromStart: 71, resumed: 29, byType: { movie: 0.41, series: 0.22, music: 0.03 } },
  '/api/v2/insights/rewatches': [{ id: 'e1', kind: 'episode', name: 'Pilote', parentId: 's1', parentName: 'Station Nord', views: 3, user: { id: 'u1', name: 'Céline' } }],
  '/api/v2/insights/genres': { bucket: 'month', months: ['2026-09', '2026-10', '2026-11'], series: [{ name: 'Drame', values: [216000, 198000, 230400] }, { name: 'Science-fiction', values: [162000, 180000, 144000] }], other: [72000, 64800, 79200] },
  '/api/v2/insights/moments': { movie: hours24(21, 20), series: hours24(21, 30), music: hours24(9, 22) },
  '/api/v2/insights/pauses': [{ id: 'm1', kind: 'movie', name: 'Les Rivages', ratio: 0.18, pausedSeconds: 1300, pauseCount: 9, watchSeconds: 7200 }],
  '/api/v2/insights/binges': [{ id: 's1', name: 'Station Nord', binges: 14, users: 3 }],
  '/api/v2/insights/rhythm': { weekHour: Array.from({ length: 7 }, (_, d) => hours24(21, 6 + d)) },
  '/api/v2/discover/unwatched': { count: 86, catalog: 352, sizeBytes: 1_200_000_000_000, items: [{ id: 'm9', kind: 'movie', name: 'Atlas', dateAdded: '2025-10-05', sizeBytes: 48_000_000_000 }] },
  '/api/v2/discover/abandoned': { count: 1, items: [{ id: 'm3', kind: 'movie', name: 'Orbital', user: { id: 'u1', name: 'Céline' }, progress: 0.42, lastPlayed: '2026-07-01T20:00:00+02:00', positionSeconds: 3240 }] },
  '/api/v2/discover/added': { count: 23, seen: 9, items: [] },
  '/api/v2/discover/storage': [{ id: 'm9', kind: 'movie', name: 'Atlas', sizeBytes: 48_000_000_000, views: 0 }, { id: 'm1', kind: 'movie', name: 'Les Rivages', sizeBytes: 18_000_000_000, views: 48 }],
  '/api/v2/playback/methods': { bucket: 'day', totals: { direct: 240, stream: 60, transcode: 60 }, daily: PHASE5_METHODS_DAILY },
  '/api/v2/playback/clients': [{ client: 'Jellyfin Web', clicks: 120, views: 100, transcodes: 62, transcodeRatio: 0.62, falseStarts: 20, falseStartRatio: 0.17 }],
  '/api/v2/playback/quality': {
    collectedSince: null, captured: 0, views: 360, transcodes: 60,
    resolution: { served: { '4K': 18, '1080p': 61, '720p': 15, SD: 6 }, sourceUhd: 25, downscaledUhd: 9 },
    hdr: { native: 11, toneMapped: 9, sdr: 80 }, reasons: [], hwAccel: [],
    conversions: [{ videoFrom: 'HEVC', videoTo: 'H264', audioFrom: 'TRUEHD', audioTo: 'AAC', views: 41 }],
    codecPairs: [{ videoFrom: 'HEVC', videoTo: 'H264', views: 41 }],
    streams: { audioOnly: 46, videoOnly: 21, both: 33 }, bitrate: { samples: 0, avg: null, p95: null, buckets: [0, 0, 0, 0, 0, 0] },
  },
  '/api/v2/playback/network': { collectedSince: null, captured: 0, local: 0, remote: 0, unknown: 360, places: [] },
  '/api/v2/recap/u1/2026': {
    user: { id: 'u1', name: 'Céline' }, period: '2026', granularity: 'month', from: '2026-01-01', to: '2027-01-01',
    totals: { watchSeconds: 6_523_200, views: 5210, activeDays: 312 }, previous: { watchSeconds: 0, views: 0, activeDays: 0 },
    topSeries: [{ id: 's1', name: 'Station Nord', watchSeconds: 1_036_800, views: 300 }], topMovies: [], topArtists: [],
    buckets: Array.from({ length: 12 }, (_, i) => ({ key: `2026-${String(i + 1).padStart(2, '0')}`, watchSeconds: 360_000 + i * 36_000, views: 400, newArtists: i % 3 })),
    records: { bestBucket: null, binge: { seriesId: 's1', seriesName: 'Station Nord', episodes: 9, date: '2026-08-14' }, longestStreak: 23 },
    ranking: [{ id: 's1', kind: 'series', name: 'Station Nord', parentName: null, views: 300, rank: 1, previousRank: null }],
  },
}
// --- end phase 5 ---

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
  // --- phase 5 ---
  ...PHASE5,
  // --- end phase 5 ---
}

// --- phase 3 ---
// Realistic payloads for the core pages (/, /live, /history, /users, /users/:id).
import { calendarFixture, concurrencyFixture, historyFixture, overviewFixture, retentionFixture, sessionsFixture, shareFixture, userFixture, usersFixture } from '../src/app/pages/overview/fixtures'
const PHASE3: Record<string, () => unknown> = {
  '/api/v2/overview': overviewFixture,
  '/api/v2/history': historyFixture,
  '/api/v2/history/calendar': () => calendarFixture(),
  '/api/v2/users': usersFixture,
  '/api/v2/users/retention': retentionFixture,
  '/api/v2/users/share': shareFixture,
  '/api/v2/users/u1': () => userFixture('u1', 'Céline'),
  '/api/v2/users/u2': () => userFixture(),
  '/api/v2/concurrency': concurrencyFixture,
  '/sessions/current': sessionsFixture,
}
// --- end phase 3 ---
// --- phase 4 ---
// Entity pages: realistic payloads for the fixture ids (lib-s, m1, s1, e5, ar1, al1, t3).
Object.assign(OBJECTS, entityApi, { '/api/v2/history': libraryHistoryFixture })
// --- end phase 4 ---

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
    if (pathname in PHASE3) return route.fulfill(json(PHASE3[pathname]())) // --- phase 3 ---
    if (pathname in OBJECTS) return route.fulfill(json(OBJECTS[pathname]))
    return route.fulfill(json([]))
  })
}

/** Puts a fake session token in localStorage before any page script runs. */
export async function login(page: Page) {
  await page.addInitScript((token) => { localStorage.setItem('jellystics-token', token) }, FAKE_JWT)
}
