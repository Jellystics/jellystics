import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { handleUnauthorized } from '@/lib/api'
import { getToken } from '@/lib/auth'
import { usePeriod } from '@/lib/useUrlState'
import { get, seg, V2 } from './client'
import { cleanParams, qk, type Filters, type HistoryFilters, type QueryParams } from './keys'
import type {
  AlbumDetail, ArtistDetail, CalendarDay, EpisodeDetail, HistoryPage, LibraryCard, LibraryDetail, MovieDetail,
  LibrariesDaily, Overview, OverviewPlays, SearchResult, SeriesDetail, TrackDetail, UserDetail, UserRow,
} from './types'

/** Request params = current URL period + page filters (normalised). */
export function usePeriodParams(filters: Filters | HistoryFilters = {}): QueryParams {
  const [period] = usePeriod()
  return cleanParams({ from: period.from, to: period.to, compare: period.compare, ...filters })
}

export function useOverview(filters?: Filters) {
  const p = usePeriodParams(filters)
  return useQuery({ queryKey: qk.overview(p), queryFn: get<Overview>(`${V2}/overview`, p), placeholderData: keepPreviousData })
}

/** Request params = current URL period without compare + page filters. */
function useRangeParams(filters: Filters = {}): QueryParams {
  const [period] = usePeriod()
  return cleanParams({ from: period.from, to: period.to, ...filters })
}

/** Plays of the period (one point each, capped server-side); compare is not used. */
export function useOverviewPlays(filters?: Filters) {
  const p = useRangeParams(filters)
  return useQuery({ queryKey: qk.overviewPlays(p), queryFn: get<OverviewPlays>(`${V2}/overview/plays`, p), placeholderData: keepPreviousData })
}

/** Views per (day or ISO week, library) of the period; compare is not used. */
export function useOverviewLibrariesDaily(filters?: Filters) {
  const p = useRangeParams(filters)
  return useQuery({ queryKey: qk.overviewLibrariesDaily(p), queryFn: get<LibrariesDaily>(`${V2}/overview/libraries-daily`, p), placeholderData: keepPreviousData })
}

export function useHistory(filters?: HistoryFilters) {
  const p = usePeriodParams(filters)
  return useQuery({ queryKey: qk.history(p), queryFn: get<HistoryPage>(`${V2}/history`, p), placeholderData: keepPreviousData })
}

/** Calendar of one month (YYYY-MM); the period is not used, only the filters. */
export function useHistoryCalendar(month: string | undefined, filters?: HistoryFilters) {
  const p = cleanParams({ month, ...filters })
  return useQuery({ queryKey: qk.historyCalendar(p), queryFn: get<CalendarDay[]>(`${V2}/history/calendar`, p), placeholderData: keepPreviousData })
}

export function useSearch(q: string) {
  const term = q.trim()
  return useQuery({
    queryKey: qk.search(term),
    queryFn: get<SearchResult>(`${V2}/search`, { q: term }),
    enabled: term.length > 0,
    placeholderData: keepPreviousData,
    staleTime: 60_000,
  })
}

export function useUsers(filters?: Filters) {
  const p = usePeriodParams(filters)
  return useQuery({ queryKey: qk.users(p), queryFn: get<UserRow[]>(`${V2}/users`, p), placeholderData: keepPreviousData })
}

function useEntity<T>(kind: 'user' | 'movie' | 'series' | 'episode' | 'artist' | 'album' | 'track', path: string, id: string | undefined, filters?: Filters) {
  const p = usePeriodParams(filters)
  return useQuery({
    queryKey: qk[kind](id ?? '', p),
    queryFn: get<T>(`${V2}/${path}/${seg(id ?? '')}`, p),
    enabled: !!id,
    placeholderData: keepPreviousData,
  })
}

export const useUser = (id: string | undefined, filters?: Filters) => useEntity<UserDetail>('user', 'users', id, filters)
export const useMovie = (id: string | undefined, filters?: Filters) => useEntity<MovieDetail>('movie', 'movies', id, filters)
export const useSeries = (id: string | undefined, filters?: Filters) => useEntity<SeriesDetail>('series', 'series', id, filters)
export const useEpisode = (id: string | undefined, filters?: Filters) => useEntity<EpisodeDetail>('episode', 'episodes', id, filters)
export const useArtist = (id: string | undefined, filters?: Filters) => useEntity<ArtistDetail>('artist', 'artists', id, filters)
export const useAlbum = (id: string | undefined, filters?: Filters) => useEntity<AlbumDetail>('album', 'albums', id, filters)
export const useTrack = (id: string | undefined, filters?: Filters) => useEntity<TrackDetail>('track', 'tracks', id, filters)

export function useLibraries(filters?: Filters) {
  const p = usePeriodParams(filters)
  return useQuery({ queryKey: qk.libraries(p), queryFn: get<LibraryCard[]>(`${V2}/libraries`, p), placeholderData: keepPreviousData })
}

export function useLibrary(id: string | undefined, filters?: Filters) {
  const p = usePeriodParams(filters)
  return useQuery({
    queryKey: qk.library(id ?? '', p),
    queryFn: get<LibraryDetail>(`${V2}/libraries/${seg(id ?? '')}`, p),
    enabled: !!id,
    placeholderData: keepPreviousData,
  })
}

/** Downloads /history.csv for params (authenticated fetch → file save). */
export async function downloadHistoryCsv(params: QueryParams): Promise<void> {
  const qs = new URLSearchParams(Object.entries(params).map(([k, v]) => [k, String(v)]))
  const token = getToken()
  const res = await fetch(`${V2}/history.csv?${qs}`, { headers: token ? { Authorization: `Bearer ${token}` } : {} })
  if (res.status === 401) handleUnauthorized()
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
  const blob = await res.blob()
  const name = /filename="([^"]+)"/.exec(res.headers.get('Content-Disposition') ?? '')?.[1] ?? 'history.csv'
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url; a.download = name
  document.body.appendChild(a)
  a.click()
  a.remove()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}
