import type { MediaFamily } from './types'

/** Common /api/v2 query contract (dates are local, `to` exclusive). */
export interface PeriodParams { from: string; to: string; compare?: boolean }
export interface Filters {
  userId?: string
  libraryId?: string
  type?: MediaFamily
  page?: number
  pageSize?: number
  sort?: string
  desc?: boolean
  q?: string
}
export interface HistoryFilters extends Filters { client?: string; device?: string; method?: string; views?: boolean }
export type QueryParams = Record<string, string | number | boolean>

/** Booleans sent as 0/1 whenever defined (the backend distinguishes absent from false). */
const EXPLICIT_BOOLEANS = new Set(['desc'])

/**
 * Normalises request params: drops empty values and false booleans (except
 * `desc`, sent as 0), sends true as 1, so equal requests always share one
 * cache key and one URL.
 */
export function cleanParams(p: Record<string, string | number | boolean | undefined | null>): QueryParams {
  const out: QueryParams = {}
  Object.keys(p).sort().forEach((k) => {
    const v = p[k]
    if (v === undefined || v === null || v === '') return
    if (v === false) { if (EXPLICIT_BOOLEANS.has(k)) out[k] = 0; return }
    out[k] = v === true ? 1 : v
  })
  return out
}

/** Query-key factory: every key starts with 'v2' and carries the full request params. */
export const qk = {
  all: ['v2'] as const,
  overview: (p: QueryParams) => ['v2', 'overview', p] as const,
  history: (p: QueryParams) => ['v2', 'history', p] as const,
  historyCalendar: (p: QueryParams) => ['v2', 'history', 'calendar', p] as const,
  search: (q: string) => ['v2', 'search', q] as const,
  users: (p: QueryParams) => ['v2', 'users', p] as const,
  user: (id: string, p: QueryParams) => ['v2', 'user', id, p] as const,
  movie: (id: string, p: QueryParams) => ['v2', 'movie', id, p] as const,
  series: (id: string, p: QueryParams) => ['v2', 'series', id, p] as const,
  episode: (id: string, p: QueryParams) => ['v2', 'episode', id, p] as const,
  artist: (id: string, p: QueryParams) => ['v2', 'artist', id, p] as const,
  album: (id: string, p: QueryParams) => ['v2', 'album', id, p] as const,
  track: (id: string, p: QueryParams) => ['v2', 'track', id, p] as const,
  libraries: (p: QueryParams) => ['v2', 'libraries', p] as const,
  library: (id: string, p: QueryParams) => ['v2', 'library', id, p] as const,
  // Entity sub-resources (entityExtras.ts)
  movieSittings: (id: string, p: QueryParams) => ['v2', 'movie', id, 'sittings', p] as const,
  libraryDecades: (id: string, p: QueryParams) => ['v2', 'library', id, 'decades', p] as const,
  libraryGrowth: (id: string, p: QueryParams) => ['v2', 'library', id, 'growth', p] as const,
  albumFullListens: (id: string, p: QueryParams) => ['v2', 'album', id, 'full-listens', p] as const,
  seriesHistory: (id: string, p: QueryParams) => ['v2', 'series', id, 'history', p] as const,
  // Users extras (usersExtra.ts)
  retention: (p: QueryParams) => ['v2', 'users-extra', 'retention', p] as const,
  share: (p: QueryParams) => ['v2', 'users-extra', 'share', p] as const,
  concurrency: (p: QueryParams) => ['v2', 'concurrency', p] as const,
  // Analyses (analytics.ts)
  insights: (name: string, p: QueryParams) => ['v2', 'insights', name, p] as const,
  discover: (name: string, p: QueryParams) => ['v2', 'discover', name, p] as const,
  playback: (name: string, p: QueryParams) => ['v2', 'playback', name, p] as const,
  recap: (userId: string, period: string) => ['v2', 'recap', userId, period] as const,
  languages: (userId: string, p: QueryParams) => ['v2', 'user', userId, 'languages', p] as const,
}
