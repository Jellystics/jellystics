// Realistic /api/v2 payloads for the phase 3 page tests and the e2e mock API.
// Keep this file free of runtime imports (types only) so Playwright can import it.
import type { CalendarDay, HistoryPage, LibrariesDaily, Overview, OverviewPlays, PlayPoint, UserDetail, UserRow } from '@/api'
import type { JellyfinSession } from '@/api/live'
import type { ConcurrencyDay, UserRetention, UserShare } from '@/api/usersExtra'

const DAY = 86_400_000
const iso = (d: Date) => d.toISOString().slice(0, 10)
const today = () => iso(new Date())
const daysBack = (n: number) => Array.from({ length: n }, (_, i) => iso(new Date(Date.now() - (n - 1 - i) * DAY)))

export function overviewFixture(): Overview {
  const dates = daysBack(30)
  const daily = dates.map((date, i) => {
    const s = 3600 + (i % 7) * 900, v = 10 + (i % 5)
    return { date, watchSeconds: s, views: v, clicks: v + 4, byType: { movie: { watchSeconds: s / 4, views: 2, clicks: 3 }, series: { watchSeconds: s / 2, views: v - 4, clicks: v - 2 }, music: { watchSeconds: s / 4, views: 2, clicks: 3 } } }
  })
  return {
    bucket: 'day' as const,
    totals: { watchSeconds: 4_492_800, views: 3412, clicks: 4507, activeUsers: 6, completion: 0.71 },
    previous: { watchSeconds: 4_000_000, views: 3100, clicks: 4100, activeUsers: 6, completion: 0.74 },
    daily,
    previousDaily: daily.map((d) => ({ ...d, watchSeconds: d.watchSeconds * 0.9, views: d.views - 1, clicks: d.clicks - 1 })),
    topItems: [
      { id: 's1', kind: 'series', name: 'Station Nord', views: 412, watchSeconds: 900_000, artId: 's1' },
      { id: 'm1', kind: 'movie', name: 'Les Rivages', views: 288, watchSeconds: 500_000, artId: 'm1' },
      { id: 'al1', kind: 'album', name: 'Halo', views: 241, watchSeconds: 50_000, artId: 'al1' },
    ],
    topUsers: [
      { id: 'u1', name: 'Céline', watchSeconds: 1_123_200, views: 980 },
      { id: 'u2', name: 'Léa', watchSeconds: 684_000, views: 601 },
    ],
    byMethod: { direct: 68, stream: 14, transcode: 18 },
    weekHour: Array.from({ length: 7 }, (_, d) => Array.from({ length: 24 }, (_, h) => (h > 18 ? d + h - 18 : 0))),
    concurrencyPeak: dates.map((date, i) => ({ date, peak: 2 + (i % 4) })),
    onThisDay: [{ itemId: 's2', kind: 'series', name: 'Les Héritiers', users: ['Hugo', 'Léa'], date: `${Number(today().slice(0, 4)) - 1}${today().slice(4)}` }],
  }
}

export function historyFixture(): HistoryPage {
  return {
    page: 1, pageSize: 25, total: 2,
    rows: [
      {
        id: 'h1', startedAt: `${today()}T21:14:00+01:00`, endedAt: `${today()}T22:00:00+01:00`, user: { id: 'u2', name: 'Léa' },
        item: { id: 'e1', kind: 'episode', name: 'Le phare', parentId: 's1', parentName: 'Station Nord', season: 2, episode: 4 },
        mediaType: 'Episode', watchSeconds: 2760, completion: 0.98, client: 'Android TV', device: 'Shield', method: 'DirectPlay', isView: true, pausedSeconds: 0,
      },
      {
        id: 'h2', startedAt: `${today()}T20:01:00+01:00`, endedAt: `${today()}T20:01:20+01:00`, user: { id: 'u1', name: 'Céline' },
        item: { id: 'm1', kind: 'movie', name: 'Les Rivages' },
        mediaType: 'Movie', watchSeconds: 20, completion: 0.01, client: 'Web', device: 'Chrome', method: 'Transcode', isView: false, pausedSeconds: 0,
      },
    ],
  }
}

export function calendarFixture(month = today().slice(0, 7)): CalendarDay[] {
  const [y, m] = month.split('-').map(Number)
  const n = new Date(y, m, 0).getDate()
  return Array.from({ length: n }, (_, i) => {
    const date = `${month}-${String(i + 1).padStart(2, '0')}`
    return i === 4
      ? { date, views: 5, watchSeconds: 9000, items: [{ id: 's1', name: 'Station Nord', kind: 'series', user: 'Léa' }, { id: 'm1', name: 'Les Rivages', kind: 'movie', user: 'Marc' }, { id: 'al1', name: 'Halo', kind: 'album', user: 'Hugo' }, { id: 'm2', name: 'Orbital', kind: 'movie', user: 'Marc' }] }
      : { date, views: 0, watchSeconds: 0, items: [] }
  })
}

export function usersFixture(): UserRow[] {
  return [
    { id: 'u1', name: 'Céline', watchSeconds: 1_123_200, views: 980, byType: { movie: 300_000, series: 700_000, music: 123_200 }, lastActivity: new Date(Date.now() - 180_000).toISOString(), trend: Array.from({ length: 14 }, (_, i) => 1000 + i * 100) },
    { id: 'u2', name: 'Léa', watchSeconds: 684_000, views: 601, byType: { movie: 100_000, series: 484_000, music: 100_000 }, lastActivity: new Date(Date.now() - 3_600_000).toISOString(), trend: Array.from({ length: 14 }, (_, i) => (i % 3) * 500) },
    { id: 'u3', name: 'Inès', watchSeconds: 0, views: 0, byType: { movie: 0, series: 0, music: 0 }, lastActivity: null, trend: Array(14).fill(0) },
  ]
}

export function userFixture(id = 'u2', name = 'Léa'): UserDetail {
  const cal = daysBack(365).map((date, i) => ({ date, watchSeconds: i % 3 === 0 ? 0 : 1800 + (i % 5) * 600 }))
  return {
    id, name, memberSince: '2023-03-04T20:00:00+01:00', favoriteDevice: 'Android TV',
    totals: { watchSeconds: 684_000, views: 601, clicks: 700, activeUsers: 1, completion: 0.82 },
    previous: { watchSeconds: 657_000, views: 580, clicks: 650, activeUsers: 1, completion: 0.8 },
    calendar: cal,
    inProgress: [{ seriesId: 's1', name: 'Station Nord', nextSeason: 2, nextEpisode: 6, nextId: 'e6', lastPlayed: new Date(Date.now() - 180_000).toISOString(), progress: 0.62 }],
    hours: Array.from({ length: 24 }, (_, h) => (h === 22 ? 30 : h > 18 ? 10 : 1)),
    profile: { regularity: 0.87, discovery: 0.34, concentration: 0.61, completion: 0.82 },
    genres: { movie: [{ name: 'Drame', watchSeconds: 36_000 }], series: [{ name: 'Science-fiction', watchSeconds: 120_000 }, { name: 'Policier', watchSeconds: 60_000 }], music: [] },
    devices: [
      { device: 'Shield', client: 'Android TV', version: '0.18.3', views: 412 },
      { device: 'Fire TV Stick', client: 'Android TV', version: '0.15.8', views: 22 },
      { device: 'Chrome', client: 'Web', version: '10.11.0', views: 51 },
    ],
    streak: { current: 9, best: 23 },
  }
}

export function concurrencyFixture(): ConcurrencyDay {
  const d = today()
  const slots = Array(96).fill(0)
  for (let i = 80; i < 88; i++) slots[i] = i < 84 ? 1 : 2
  return {
    date: d, peak: 2, peakAt: `${d}T21:00:00+01:00`, slotMinutes: 15, slots,
    sessions: [
      { user: { id: 'u2', name: 'Léa' }, itemId: 'e1', name: 'Le phare', parentName: 'Station Nord', kind: 'series', start: `${d}T20:00:00+01:00`, end: `${d}T20:50:00+01:00` },
      { user: { id: 'u3', name: 'Marc' }, itemId: 'm1', name: 'Les Rivages', kind: 'movie', start: `${d}T21:00:00+01:00`, end: `${d}T22:00:00+01:00` },
    ],
  }
}

export function retentionFixture(): UserRetention {
  return { offsets: 5, rows: [{ week: '2026-10-26', cohort: 6, retained: [6, 5, 5, 4, null] }, { week: '2026-11-02', cohort: 5, retained: [5, 5, 4, null, null] }] }
}

export function shareFixture(): UserShare {
  return {
    bucket: 'week', periods: ['2026-10-26', '2026-11-02', '2026-11-09', '2026-11-16'],
    users: [
      { id: 'u1', name: 'Céline', watchSeconds: 1_123_200, share: 0.62, values: [0.5, 0.6, 0.7, 0.62] },
      { id: 'u2', name: 'Léa', watchSeconds: 684_000, share: 0.38, values: [0.5, 0.4, 0.3, 0.38] },
    ],
  }
}

export function sessionsFixture(): JellyfinSession[] {
  return [
    {
      Id: 'sess1', UserId: 'u2', UserName: 'Léa', Client: 'Android TV', DeviceName: 'Shield',
      NowPlayingItem: { Id: 'e1', Name: 'Le phare', Type: 'Episode', SeriesName: 'Station Nord', SeriesId: 's1', RunTimeTicks: 30_000_000_000, MediaSources: [{ Bitrate: 12_000_000 }] },
      PlayState: { IsPaused: false, PositionTicks: 12_000_000_000, PlayMethod: 'DirectPlay' },
    },
    {
      Id: 'sess2', UserId: 'u3', UserName: 'Marc', Client: 'Web', DeviceName: 'Chrome',
      NowPlayingItem: { Id: 'm1', Name: 'Les Rivages', Type: 'Movie', RunTimeTicks: 60_000_000_000 },
      PlayState: { IsPaused: true, PositionTicks: 6_000_000_000, PlayMethod: 'Transcode' },
      TranscodingInfo: { Bitrate: 21_000_000, HardwareAccelerationType: 'vaapi' },
    },
    { Id: 'idle', UserId: 'u1', UserName: 'Céline', Client: 'Web', DeviceName: 'Firefox', NowPlayingItem: null },
  ]
}

/** /api/v2/overview/plays: deterministic plays over the last 30 days, newest first. */
export function playsFixture(capped = false): OverviewPlays {
  const items: { type: PlayPoint['type']; item: PlayPoint['item']; minutes: number }[] = [
    { type: 'series', item: { id: 'e1', kind: 'episode', name: 'Le phare', parentId: 's1', parentName: 'Station Nord' }, minutes: 46 },
    { type: 'series', item: { id: 'e5', kind: 'episode', name: 'La relève', parentId: 's2', parentName: 'Les Héritiers' }, minutes: 52 },
    { type: 'movie', item: { id: 'm1', kind: 'movie', name: 'Les Rivages' }, minutes: 118 },
    { type: 'movie', item: { id: 'm2', kind: 'movie', name: 'Orbital' }, minutes: 132 },
    { type: 'music', item: { id: 't3', kind: 'track', name: 'Lueur', parentId: 'al1', parentName: 'Halo' }, minutes: 4 },
  ]
  const users = [{ id: 'u1', name: 'Céline' }, { id: 'u2', name: 'Léa' }, { id: 'u3', name: 'Hugo' }]
  const now = Date.now()
  const points = Array.from({ length: 240 }, (_, i): PlayPoint => {
    const it = items[(i * 7) % items.length]
    const frac = ((i * 37) % 100) / 100
    return {
      id: `p${i}`, startedAt: new Date(now - (i * 3 + (i % 5)) * 3_600_000).toISOString(), type: it.type, item: it.item, user: users[i % users.length],
      watchSeconds: Math.round(it.minutes * 60 * (i % 4 === 0 ? frac : 0.85 + frac * 0.15)),
    }
  })
  return { total: capped ? 5200 : points.length, limit: 5000, capped, points }
}

/** /api/v2/overview/libraries-daily: views per day for four libraries. */
export function librariesDailyFixture(): LibrariesDaily {
  const libs: { id: string; name: string; type: LibrariesDaily['libraries'][number]['type'] }[] = [
    { id: 'lib-s', name: 'Séries', type: 'series' }, { id: 'lib-m', name: 'Films', type: 'movie' },
    { id: 'lib-mu', name: 'Musique', type: 'music' }, { id: 'lib-k', name: 'Films enfants', type: 'movie' },
  ]
  const base = [9, 4, 6, 2]
  const points = daysBack(30).flatMap((date, d) => libs.map((l, i) => ({ date, libraryId: l.id, views: Math.max(0, base[i] + ((d * (i + 3)) % 5) - 2) }))).filter((p) => p.views > 0)
  const libraries = libs
    .map((l) => ({ ...l, views: points.filter((p) => p.libraryId === l.id).reduce((a, p) => a + p.views, 0) }))
    .sort((a, b) => b.views - a.views)
  return { bucket: 'day', libraries, points }
}
