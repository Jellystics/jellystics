// Shapes of the /api/me/* responses (relayed by the Jellyfin plugin). See backend/internal/handler/me.go.
import type { PlayedToEnd } from '../api/types' // relative: the e2e imports the fixtures

export type { PlayedToEnd }
export type Family = 'movie' | 'series' | 'music'
export interface Totals { watchSeconds: number; views: number; clicks: number; completion: number | null }
export interface Counts { watchSeconds: number; views: number; clicks: number }

export interface MeProfile {
  id: string
  name: string
  primaryImageTag: string | null
  memberSince: string | null
  totals: Totals
  previous?: Totals
  favoriteDevice: string | null
  calendar: { date: string; watchSeconds: number }[]
  inProgress: { seriesId: string; name: string; nextSeason: number | null; nextEpisode: number | null; nextId: string; lastPlayed: string; progress: number }[]
  hours: number[]
  profile: { regularity: number | null; discovery: number | null; concentration: number | null; completion: number | null }
  playedToEnd: PlayedToEnd
  pluginSince: string | null
  genres: Record<Family, { name: string; watchSeconds: number }[]>
  streak: { current: number; best: number }
}

export interface DailyPoint extends Counts { date: string; byType: Record<Family, Counts> }

export interface MeOverview {
  bucket: 'day' | 'week'
  totals: Totals
  previous?: Totals
  daily: DailyPoint[]
  previousDaily?: DailyPoint[]
  byMethod: { direct: number; stream: number; transcode: number }
  playedToEnd: PlayedToEnd
  pluginSince: string | null
}

export interface MeTopItem { id: string; kind: 'movie' | 'series' | 'album'; name: string; views: number; watchSeconds: number; artId: string }
export interface MeTop { items: MeTopItem[] }

export interface MeHistoryRow {
  id: string
  startedAt: string
  endedAt: string
  item: { id: string; kind: 'movie' | 'episode' | 'track'; name: string; parentId?: string; parentName?: string; season?: number; episode?: number }
  mediaType: 'Movie' | 'Episode' | 'Audio'
  watchSeconds: number
  completion: number | null
  client: string
  device: string
  method: string
  isView: boolean
  pausedSeconds: number
}
export interface MeHistory { rows: MeHistoryRow[]; total: number; page: number; pageSize: number }

export interface MeTypes { byType: Record<Family, Counts>; playedToEnd: Record<Family, PlayedToEnd> }
export interface MeHabits { weekHour: number[][]; hours: number[]; weekdays: number[] }

export interface MeDevices {
  devices: { device: string; client: string; version: string; views: number; clicks: number; watchSeconds: number; lastUsed: string | null }[]
  clients: { client: string; clicks: number; views: number; transcodes: number; transcodeRatio: number | null; falseStarts: number; falseStartRatio: number | null }[]
}

export interface MeScatter {
  total: number
  limit: number
  capped: boolean
  points: { id: string; startedAt: string; watchSeconds: number; type: Family; item: { id: string; kind: string; name: string; parentId?: string; parentName?: string } }[]
}
