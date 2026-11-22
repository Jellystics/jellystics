// Entity-page widgets added in phase 4 (backend/internal/handler/v2_media_extra.go).
// Kept in their own file so the phase 2 client stays untouched.
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { apiFetch } from '@/lib/api'
import { usePeriodParams } from './hooks'
import type { Filters, QueryParams } from './keys'
import type { HistoryPage, ISODate } from './types'

/** Completions of a movie by number of sittings it took (4 = four or more). */
export interface SittingsBucket { sittings: 1 | 2 | 3 | 4; count: number }
/** Views in the range vs catalog titles, per production decade (1990 = the 1990s). */
export interface DecadeRow { decade: number; views: number; catalog: number }
export type GrowthBucket = 'day' | 'week' | 'month' | 'year'
/** Movies/episodes/tracks added vs viewed for the first time, per bucket start date. */
export interface LibraryGrowth { bucket: GrowthBucket; points: { date: ISODate; added: number; firstViews: number }[] }
/** Albums of the same artist with their full listens ((user, day) with every track viewed). */
export interface FullListenAlbum { id: string; name: string; year: number | null; tracks: number; listens: number; fullListens: number }

const V2 = '/api/v2'
const seg = encodeURIComponent

export const qkx = {
  movieSittings: (id: string, p: QueryParams) => ['v2', 'movie', id, 'sittings', p] as const,
  libraryDecades: (id: string, p: QueryParams) => ['v2', 'library', id, 'decades', p] as const,
  libraryGrowth: (id: string, p: QueryParams) => ['v2', 'library', id, 'growth', p] as const,
  albumFullListens: (id: string, p: QueryParams) => ['v2', 'album', id, 'full-listens', p] as const,
  seriesHistory: (id: string, p: QueryParams) => ['v2', 'series', id, 'history', p] as const,
}

function useSub<T>(key: (id: string, p: QueryParams) => readonly unknown[], path: string, sub: string, id: string | undefined, filters?: Filters) {
  const p = usePeriodParams(filters)
  return useQuery({
    queryKey: key(id ?? '', p),
    queryFn: ({ signal }) => apiFetch<T>(`${V2}/${path}/${seg(id ?? '')}/${sub}`, { params: p, signal }),
    enabled: !!id,
    placeholderData: keepPreviousData,
  })
}

export const useMovieSittings = (id: string | undefined, filters?: Filters) => useSub<SittingsBucket[]>(qkx.movieSittings, 'movies', 'sittings', id, filters)
export const useLibraryDecades = (id: string | undefined, filters?: Filters) => useSub<DecadeRow[]>(qkx.libraryDecades, 'libraries', 'decades', id, filters)
export const useLibraryGrowth = (id: string | undefined, filters?: Filters) => useSub<LibraryGrowth>(qkx.libraryGrowth, 'libraries', 'growth', id, filters)
export const useAlbumFullListens = (id: string | undefined, filters?: Filters) => useSub<FullListenAlbum[]>(qkx.albumFullListens, 'albums', 'full-listens', id, filters)
export const useSeriesHistory = (id: string | undefined, filters?: Filters) => useSub<HistoryPage>(qkx.seriesHistory, 'series', 'history', id, filters)
