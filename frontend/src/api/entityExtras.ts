// Entity-page widgets added in phase 4 (backend/internal/handler/v2_media_extra.go)
// and a series' history (/api/v2/history?seriesId=).
// Kept in their own file so the phase 2 client stays untouched.
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { get, seg, V2 } from './client'
import { usePeriodParams } from './hooks'
// `qkx` aliases the shared key factory (keys.ts) so the hook lines below stay short.
import { qk as qkx, type Filters, type QueryParams } from './keys'
import type { HistoryPage, ISODate } from './types'

/** Completions of a movie by number of sittings it took (4 = four or more). */
export interface SittingsBucket { sittings: 1 | 2 | 3 | 4; count: number }
/** Views in the range vs catalog titles, per production decade (1990 = the 1990s). */
export interface DecadeRow { decade: number; views: number; catalog: number }
export type GrowthBucket = 'day' | 'week' | 'month' | 'year'
/**
 * Movies/episodes/tracks added vs viewed for the first time, per bucket start date.
 * source "events": real additions / removals recorded by the Jellystics plugin
 * from `preciseFrom` on (catalog creation dates of the items still present
 * before it); `precise` marks the buckets fed by plugin events only.
 */
export interface LibraryGrowth {
  bucket: GrowthBucket
  source: 'events' | 'catalog'
  preciseFrom: ISODate | null
  points: { date: ISODate; added: number; removed: number; firstViews: number; precise: boolean }[]
}
/** Albums of the same artist with their full listens ((user, day) with every track viewed). */
export interface FullListenAlbum { id: string; name: string; year: number | null; tracks: number; listens: number; fullListens: number }

function useSub<T>(key: (id: string, p: QueryParams) => readonly unknown[], path: string, sub: string, id: string | undefined, filters?: Filters) {
  const p = usePeriodParams(filters)
  return useQuery({
    queryKey: key(id ?? '', p),
    queryFn: get<T>(`${V2}/${path}/${seg(id ?? '')}/${sub}`, p),
    enabled: !!id,
    placeholderData: keepPreviousData,
  })
}

export const useMovieSittings = (id: string | undefined, filters?: Filters) => useSub<SittingsBucket[]>(qkx.movieSittings, 'movies', 'sittings', id, filters)
export const useLibraryDecades = (id: string | undefined, filters?: Filters) => useSub<DecadeRow[]>(qkx.libraryDecades, 'libraries', 'decades', id, filters)
export const useLibraryGrowth = (id: string | undefined, filters?: Filters) => useSub<LibraryGrowth>(qkx.libraryGrowth, 'libraries', 'growth', id, filters)
export const useAlbumFullListens = (id: string | undefined, filters?: Filters) => useSub<FullListenAlbum[]>(qkx.albumFullListens, 'albums', 'full-listens', id, filters)
/** Paged history of a series' episodes: /history filtered by seriesId. */
export function useSeriesHistory(id: string | undefined, filters?: Filters) {
  const p: QueryParams = { ...usePeriodParams(filters), seriesId: id ?? '' }
  return useQuery({
    queryKey: qkx.seriesHistory(id ?? '', p),
    queryFn: get<HistoryPage>(`${V2}/history`, p),
    enabled: !!id,
    placeholderData: keepPreviousData,
  })
}
