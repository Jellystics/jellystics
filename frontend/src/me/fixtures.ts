// /api/me/* responses shaped exactly like the backend contract (backend/internal/handler/me.go),
// no path aliases (the e2e imports this file),
// shared by the vitest suite and the Playwright e2e.
import type { MeDevices, MeHabits, MeHistory, MeHistoryRow, MeOverview, MeProfile, MeScatter, MeTop, MeTypes } from './types'

const zero = { watchSeconds: 0, views: 0, clicks: 0 }
const day = (i: number) => `2026-11-${String(i + 1).padStart(2, '0')}`

export const ME_USER_ID = 'a1b2c3d4e5f60718293a4b5c6d7e8f90'

export const meProfile: MeProfile = {
  id: ME_USER_ID, name: 'Alice', primaryImageTag: 'tag1', timezone: 'Europe/Paris', memberSince: '2025-11-24T12:00:00+01:00',
  totals: { watchSeconds: 34_200, views: 18, clicks: 21, completion: 0.8625 },
  previous: { watchSeconds: 0, views: 0, clicks: 0, completion: null },
  favoriteDevice: 'Chrome',
  calendar: Array.from({ length: 30 }, (_, i) => ({ date: day(i), watchSeconds: i % 3 ? 0 : 1800 })),
  inProgress: [],
  hours: Array.from({ length: 24 }, (_, h) => (h >= 20 && h <= 22 ? 4 : 0)),
  profile: { regularity: 0.1, discovery: 0.75, concentration: 1, completion: 0.75 },
  playedToEnd: { plays: 18, finished: 12, measured: 0, measuredFinished: 0, ratio: 0.6667 },
  pluginSince: null,
  genres: { movie: [{ name: 'Sci-Fi', watchSeconds: 100 }], series: [{ name: 'Drama', watchSeconds: 240 }], music: [] },
  streak: { current: 3, best: 7 },
}

export const meOverview: MeOverview = {
  bucket: 'day',
  totals: { watchSeconds: 34_200, views: 18, clicks: 21, completion: 0.8625 },
  previous: { watchSeconds: 28_800, views: 12, clicks: 15, completion: 0.8 },
  daily: Array.from({ length: 24 }, (_, i) => {
    const s = (i % 4) * 1200, v = i % 4
    return { date: day(i), watchSeconds: s, views: v, clicks: v + (i % 2), byType: { movie: { watchSeconds: s / 2, views: 0, clicks: 0 }, series: { watchSeconds: s / 2, views: v, clicks: v }, music: zero } }
  }),
  previousDaily: [],
  byMethod: { direct: 10, stream: 6, transcode: 2 },
  playedToEnd: { plays: 18, finished: 12, measured: 4, measuredFinished: 3, ratio: 0.6667 },
  pluginSince: null,
}

export const meTop: MeTop = {
  items: [
    { id: 's1', kind: 'series', name: 'Les Rivages', views: 9, watchSeconds: 24_000, artId: 's1' },
    { id: 'm1', kind: 'movie', name: 'Inception', views: 2, watchSeconds: 9_000, artId: 'm1' },
    { id: 'al1', kind: 'album', name: 'Blue Train', views: 1, watchSeconds: 1_200, artId: 'al1' },
  ],
}

export const meTypes: MeTypes = {
  byType: { movie: { watchSeconds: 9_000, views: 2, clicks: 2 }, series: { watchSeconds: 24_000, views: 15, clicks: 18 }, music: { watchSeconds: 1_200, views: 1, clicks: 1 } },
  playedToEnd: {
    movie: { plays: 2, finished: 2, measured: 0, measuredFinished: 0, ratio: 1 },
    series: { plays: 15, finished: 10, measured: 4, measuredFinished: 3, ratio: 0.6666666666666666 },
    music: { plays: 1, finished: 0, measured: 0, measuredFinished: 0, ratio: null },
  },
}

export const meHabits: MeHabits = {
  weekHour: Array.from({ length: 7 }, (_, d) => Array.from({ length: 24 }, (_, h) => (h >= 20 && h <= 22 ? d + 1 : 0))),
  hours: Array.from({ length: 24 }, (_, h) => (h >= 20 && h <= 22 ? 6 : 0)),
  weekdays: [1, 0, 0, 0, 0, 1, 2],
}

export const meDevices: MeDevices = {
  devices: [
    { device: 'Chrome', client: 'Jellyfin Web', version: '10.11.0', views: 14, clicks: 16, watchSeconds: 27_000, lastUsed: '2026-11-23T21:00:00+01:00' },
    { device: 'Living room TV', client: 'Jellyfin Android TV', version: '0.18.0', views: 4, clicks: 5, watchSeconds: 7_200, lastUsed: null },
  ],
  clients: [
    { client: 'Jellyfin Web', clicks: 16, views: 14, transcodes: 1, transcodeRatio: 0.0714, falseStarts: 2, falseStartRatio: 0.125 },
    { client: 'Jellyfin Android TV', clicks: 5, views: 4, transcodes: 1, transcodeRatio: 0.25, falseStarts: 1, falseStartRatio: null },
  ],
}

export const meScatter: MeScatter = {
  total: 3, limit: 5000, capped: false,
  points: [
    { id: 'r6', startedAt: '2026-11-23T21:00:00+01:00', watchSeconds: 2_700, type: 'series', item: { id: 'e3', kind: 'episode', name: 'Return', parentId: 's1', parentName: 'Les Rivages' } },
    { id: 'r5', startedAt: '2026-11-20T20:30:00+01:00', watchSeconds: 8_400, type: 'movie', item: { id: 'm1', kind: 'movie', name: 'Inception' } },
    { id: 'r4', startedAt: '2026-11-18T09:00:00+01:00', watchSeconds: 300, type: 'music', item: { id: 't1', kind: 'track', name: 'Moment’s Notice', parentId: 'al1', parentName: 'Blue Train' } },
  ],
}

/** 30 history rows, newest first. */
export const meHistoryRows: MeHistoryRow[] = Array.from({ length: 30 }, (_, i) => ({
  id: `r${i + 1}`,
  startedAt: `2026-11-${String(23 - (i % 20)).padStart(2, '0')}T21:${String(i % 60).padStart(2, '0')}:00+01:00`,
  endedAt: `2026-11-${String(23 - (i % 20)).padStart(2, '0')}T21:59:00+01:00`,
  item: i % 3 === 1
    ? { id: `m${i}`, kind: 'movie', name: `Film ${i + 1}` }
    : { id: `e${i}`, kind: 'episode', name: `Episode ${i + 1}`, parentId: 's1', parentName: 'Les Rivages', season: 1, episode: i + 1 },
  mediaType: i % 3 === 1 ? 'Movie' : 'Episode',
  watchSeconds: 600 + i * 60,
  completion: i % 5 === 0 ? null : 0.5 + (i % 5) / 10,
  client: 'Jellyfin Web', device: 'Chrome',
  method: ['DirectPlay', 'DirectStream', 'Transcode', ''][i % 4],
  isView: i % 6 !== 5,
  pausedSeconds: 0,
}))

export function meHistoryPage(page = 1, pageSize = 25): MeHistory {
  return { rows: meHistoryRows.slice((page - 1) * pageSize, page * pageSize), total: meHistoryRows.length, page, pageSize }
}

/** Body of GET ./api/<path>?<query> (undefined for an unknown path). */
export function meResponse(path: string, query: URLSearchParams): unknown {
  switch (path) {
    case 'profile': return meProfile
    case 'overview': return meOverview
    case 'top': return meTop
    case 'types': return meTypes
    case 'habits': return meHabits
    case 'devices': return meDevices
    case 'scatter': return meScatter
    case 'history': return meHistoryPage(Number(query.get('page') || 1), Number(query.get('pageSize') || 25))
    default: return undefined
  }
}
