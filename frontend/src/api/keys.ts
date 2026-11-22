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

/**
 * Normalises request params: drops empty values and false booleans, and sends
 * booleans as 1, so equal requests always share one cache key and one URL.
 */
export function cleanParams(p: Record<string, string | number | boolean | undefined | null>): QueryParams {
  const out: QueryParams = {}
  Object.keys(p).sort().forEach((k) => {
    const v = p[k]
    if (v === undefined || v === null || v === '' || v === false) return
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
}
