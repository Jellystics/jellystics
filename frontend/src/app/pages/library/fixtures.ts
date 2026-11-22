// Realistic /api/v2 payloads for the phase 4 entity pages. Used by the Vitest
// page tests and the Playwright mocks (no path aliases: e2e imports it too).

const totals = (watchSeconds: number, views: number, clicks: number, activeUsers: number, completion: number | null) => ({ watchSeconds, views, clicks, activeUsers, completion })
const trend = (seed: number) => Array.from({ length: 14 }, (_, i) => ((i * 7 + seed * 3) % 9) + (i > 9 ? 2 : 0))
const u = (id: string, name: string) => ({ id, name })
const LIB_S = { id: 'lib-s', name: 'Séries' }
const LIB_MU = { id: 'lib-mu', name: 'Musique' }
const USERS = [u('u1', 'Céline'), u('u2', 'Hugo'), u('u3', 'Léa'), u('u4', 'Malik'), u('u5', 'Inès')]

const row = (id: string, user: { id: string; name: string }, item: Record<string, unknown>, mediaType: string, startedAt: string, watchSeconds: number, completion: number | null, method = 'DirectPlay', isView = true) => ({
  id, startedAt, endedAt: startedAt, user, item, mediaType, watchSeconds, completion,
  client: 'Jellyfin Web', device: 'Chrome', method, isView, pausedSeconds: 0,
})

export const librariesFixture = [
  { id: 'lib-m', name: 'Films', type: 'movies', items: 248, sizeBytes: 3_120_000_000_000, totals: totals(1_324_800, 520, 610, 6, 0.64) },
  { id: 'lib-s', name: 'Séries', type: 'tvshows', items: 4, sizeBytes: 412_000_000_000, totals: totals(2_203_200, 1804, 2100, 6, 0.78) },
  { id: 'lib-mu', name: 'Musique', type: 'music', items: 31, sizeBytes: 48_000_000_000, totals: totals(133_200, 612, 840, 3, 0.73) },
]

export const libraryFixture = {
  library: librariesFixture[1],
  items: {
    rows: [
      { id: 's1', kind: 'series', name: 'Station Nord', views: 412, viewers: 6, watchSeconds: 1_036_800, retention: 0.33, trend: trend(1) },
      { id: 's2', kind: 'series', name: 'Brigade 9', views: 198, viewers: 4, watchSeconds: 507_600, retention: 0.5, trend: trend(2) },
      { id: 's3', kind: 'series', name: 'Les Héritiers', views: 140, viewers: 3, watchSeconds: 352_800, retention: 0.66, trend: trend(3) },
    ],
    total: 3, page: 1, pageSize: 25,
  },
  genres: [{ name: 'Drame', watchSeconds: 1_200_000, views: 900 }, { name: 'Science-fiction', watchSeconds: 800_000, views: 610 }, { name: 'Policier', watchSeconds: 300_000, views: 250 }],
  methods: { direct: 680, stream: 140, transcode: 180 },
  unwatched: { count: 1, list: [{ id: 's4', kind: 'series', name: 'Le Phare', dateAdded: '2026-10-02' }] },
  timeToWatch: { count: 41, avgDays: 12.4, medianDays: 3 },
}

export const decadesFixture = [
  { decade: 1990, views: 90, catalog: 1 },
  { decade: 2010, views: 280, catalog: 1 },
  { decade: 2020, views: 430, catalog: 2 },
]

export const growthFixture = {
  bucket: 'week',
  points: ['2026-10-26', '2026-11-02', '2026-11-09', '2026-11-16'].map((date, i) => ({ date, added: [12, 30, 18, 22][i], firstViews: [10, 14, 16, 15][i] })),
}

const stops = (n: number, end: number) => Array.from({ length: n }, (_, i) => (i < 2 ? 20 - i * 4 : i === n - 1 ? end : i === n - 2 ? Math.round(end / 2) : (i * 5) % 9))

export const movieFixture = {
  item: { id: 'm1', name: 'Les Rivages', year: 2024, runtimeSeconds: 7680, genres: ['Drame'], sizeBytes: 18_200_000_000, resolution: '4K', library: { id: 'lib-m', name: 'Films' } },
  totals: { views: 48, clicks: 61, viewers: 5, completedRatio: 0.6, transcodeRatio: 0.21, watchSeconds: 302_400 },
  funnel: { clicks: 61, views: 48, half: 37, completed: 29 },
  stopPositions: stops(24, 92),
  viewers: USERS.slice(0, 3).map((user, i) => ({ user, views: [14, 9, 6][i], lastWatched: `2026-11-${20 - i}T21:00:00+01:00`, bestCompletion: [0.99, 0.95, 0.52][i] })),
  history: [
    row('h1', USERS[0], { id: 'm1', kind: 'movie', name: 'Les Rivages' }, 'Movie', '2026-11-20T21:00:00+01:00', 7600, 0.99),
    row('h2', USERS[1], { id: 'm1', kind: 'movie', name: 'Les Rivages' }, 'Movie', '2026-11-19T20:30:00+01:00', 1200, 0.16, 'Transcode'),
  ],
}

export const sittingsFixture = [{ sittings: 1, count: 18 }, { sittings: 2, count: 7 }, { sittings: 3, count: 3 }, { sittings: 4, count: 1 }]

const ep = (id: string, number: number, name: string, views: number, viewers: number, avgCompletion: number, abandons: number) => ({ id, number, name, views, viewers, avgCompletion, abandons })
export const seriesFixture = {
  item: { id: 's1', name: 'Station Nord', years: { from: 2022, to: 2025 }, seasons: 2, episodes: 5, genres: ['Drame', 'Science-fiction'], library: LIB_S },
  totals: { watchSeconds: 1_036_800, views: 412, clicks: 497, viewers: 5, retention: 0.33, binges: 14 },
  retention: [[6, 6, 5], [4, 3]],
  seasons: [
    { number: 1, episodes: [ep('e1', 1, 'Le Signal', 6, 6, 0.94, 0), ep('e2', 2, 'Dérive', 6, 5, 0.88, 1), ep('e3', 3, 'Les Glaces', 5, 5, 0.91, 0)] },
    { number: 2, episodes: [ep('e4', 1, 'Hivernage', 5, 4, 0.62, 2), ep('e5', 2, 'La Fracture', 5, 3, 0.86, 0)] },
  ],
  viewers: USERS.map((user, i) => ({ user, episodesWatched: 5 - i, lastWatched: `2026-11-${20 - i}T21:00:00+01:00` })),
  cadence: [{ bucket: '<1min', count: 420 }, { bucket: '1-15min', count: 130 }, { bucket: 'nextDay', count: 190 }, { bucket: '>7d', count: 70 }],
  releaseLag: [{ bucket: '<24h', count: 18 }, { bucket: '1-7d', count: 34 }, { bucket: '>1y', count: 65 }],
}

export const seriesHistoryFixture = {
  rows: [
    row('sh1', USERS[0], { id: 'e5', kind: 'episode', name: 'La Fracture', parentId: 's1', parentName: 'Station Nord', season: 2, episode: 2 }, 'Episode', '2026-11-20T21:10:00+01:00', 2810, 0.98),
    row('sh2', USERS[1], { id: 'e4', kind: 'episode', name: 'Hivernage', parentId: 's1', parentName: 'Station Nord', season: 2, episode: 1 }, 'Episode', '2026-11-19T21:10:00+01:00', 1500, 0.52, 'DirectStream'),
  ],
  total: 2, page: 1, pageSize: 25,
}

export const episodeFixture = {
  item: { id: 'e5', name: 'La Fracture', seriesId: 's1', seriesName: 'Station Nord', season: 2, episode: 2, runtimeSeconds: 2880, premiereDate: '2024-03-03', library: LIB_S, prev: { id: 'e4', season: 2, episode: 1 }, next: null },
  totals: { views: 5, viewers: 5, avgCompletion: 0.86, pausedSecondsAvg: 240 },
  funnel: { clicks: 7, views: 5, half: 5, completed: 4 },
  stopPositions: stops(20, 6),
  viewers: USERS.map((user, i) => ({ user, startedAt: `2026-11-${20 - i}T21:${10 + i * 7}:00+01:00`, watchSeconds: [2810, 2790, 2400, 1500, 2850][i], completion: [0.98, 0.97, 0.83, 0.52, 0.99][i], device: 'Chrome' })),
}

export const artistFixture = {
  item: { id: 'ar1', name: 'Lune Pâle', genres: ['Indie', 'Dream pop'], albums: 2, tracks: 23, library: LIB_MU },
  totals: { listens: 612, watchSeconds: 133_200, listeners: 3, skipRate: 0.19 },
  topTracks: [
    { id: 't1', name: 'Rivière noire', albumId: 'al2', albumName: 'Rivière noire', views: 182, clicks: 214, skipRate: 0.15, listeners: 3 },
    { id: 't2', name: 'Halo', albumId: 'al1', albumName: 'Halo', views: 141, clicks: 160, skipRate: 0.12, listeners: 3 },
    { id: 't3', name: 'Basse lumière', albumId: 'al1', albumName: 'Halo', views: 96, clicks: 151, skipRate: 0.36, listeners: 2 },
  ],
  albums: [{ id: 'al1', name: 'Halo', year: 2023, tracks: 11, listens: 341 }, { id: 'al2', name: 'Rivière noire', year: 2025, tracks: 12, listens: 271 }],
  skipByHour: Array.from({ length: 24 }, (_, h) => Math.round((0.12 + Math.exp(-((h - 8) ** 2) / 3) * 0.18) * 100) / 100),
}

export const albumFixture = {
  item: { id: 'al1', name: 'Halo', artistId: 'ar1', artistName: 'Lune Pâle', year: 2023, tracks: 3, runtimeSeconds: 2520, library: LIB_MU },
  totals: { listens: 341, fullListenRatio: 0.62, fullAlbumListens: 9, listeners: 3, skipRate: 0.16 },
  tracks: [
    { id: 't4', number: 1, name: 'Aube', runtimeSeconds: 192, views: 44, clicks: 51, skipRate: 0.14, listeners: 3 },
    { id: 't2', number: 2, name: 'Halo', runtimeSeconds: 241, views: 141, clicks: 160, skipRate: 0.12, listeners: 3 },
    { id: 't3', number: 3, name: 'Basse lumière', runtimeSeconds: 222, views: 96, clicks: 151, skipRate: 0.36, listeners: 2 },
  ],
}

export const fullListensFixture = [
  { id: 'al1', name: 'Halo', year: 2023, tracks: 11, listens: 341, fullListens: 9 },
  { id: 'al2', name: 'Rivière noire', year: 2025, tracks: 12, listens: 271, fullListens: 6 },
]

export const trackFixture = {
  item: { id: 't3', name: 'Basse lumière', albumId: 'al1', albumName: 'Halo', artistId: 'ar1', artistName: 'Lune Pâle' },
  totals: { views: 96, clicks: 151, skipRate: 0.36, listeners: 2 },
  recent: [{ user: USERS[1], startedAt: '2026-11-22T10:00:00+01:00', watchSeconds: 222 }, { user: USERS[0], startedAt: '2026-11-21T18:00:00+01:00', watchSeconds: 60 }],
}

/** Path → payload map of every phase 4 endpoint (ids of the fixtures above). */
export const entityApi: Record<string, unknown> = {
  '/api/v2/libraries': librariesFixture,
  '/api/v2/libraries/lib-s': libraryFixture,
  '/api/v2/libraries/lib-s/decades': decadesFixture,
  '/api/v2/libraries/lib-s/growth': growthFixture,
  '/api/v2/movies/m1': movieFixture,
  '/api/v2/movies/m1/sittings': sittingsFixture,
  '/api/v2/series/s1': seriesFixture,
  '/api/v2/series/s1/history': seriesHistoryFixture,
  '/api/v2/episodes/e5': episodeFixture,
  '/api/v2/artists/ar1': artistFixture,
  '/api/v2/albums/al1': albumFixture,
  '/api/v2/albums/al1/full-listens': fullListensFixture,
  '/api/v2/tracks/t3': trackFixture,
}
