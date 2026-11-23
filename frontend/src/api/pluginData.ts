// Precise data recorded by the Jellystics Jellyfin plugin
// (backend/internal/handler/v2_plugin.go): logins, manual vs automatic played
// marks, delay between a real addition and the first view, most skipped
// series. `collectedSince` is the first plugin row of the source (null: none
// yet) — widgets hide instead of showing zeros.
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { get, seg, V2 } from './client'
import { usePeriodParams } from './hooks'
import { cleanParams, qk, type Filters, type QueryParams } from './keys'
import type { ISODate, ISOTime, SkipStats } from './types'

export interface LoginDevice { device: string; client: string; version: string; count: number; lastAt: ISOTime }
export interface UserLogins {
  collectedSince: ISODate | null
  /** Successful logins in the range. */
  count: number
  /** Last successful login ever. */
  lastLogin: ISOTime | null
  devices: LoginDevice[]
  /** Failed attempts in the range: admins only (null otherwise). */
  failed: number | null
}

/** Users per current played state of an item (all time). */
export interface ItemMarks {
  collectedSince: ISODate | null
  /** Marked played by hand (TogglePlayed). */
  manual: number
  /** Marked played by the end of a playback. */
  automatic: number
  /** Marked played otherwise (import, third-party API…). */
  other: number
  unplayed: number
  /** Users with a recorded view (Jellystics history). */
  watched: number
  /** Manual marks of users without any recorded view. */
  manualWithoutView: number
}

export type FirstViewBucket = '<1h' | '1-24h' | '1-7d' | '1-4w' | '>4w'
export interface FirstViewDelay {
  collectedSince: ISODate | null
  added: number
  viewed: number
  notViewed: number
  medianSeconds: number | null
  buckets: { bucket: FirstViewBucket; count: number }[]
}

export interface SkippedSeries { id: string; name: string; plays: number; introRate: number | null; skipRate: number | null; avgSkippedSeconds: number | null }
export interface SkipsInsight { collectedSince: ISODate | null; totals: SkipStats; series: SkippedSeries[] }

function useSub<T>(key: (id: string, p: QueryParams) => readonly unknown[], path: string, id: string | undefined, p: QueryParams) {
  return useQuery({ queryKey: key(id ?? '', p), queryFn: get<T>(path, p), enabled: !!id, placeholderData: keepPreviousData })
}

export function useUserLogins(userId: string | undefined, filters?: Filters) {
  return useSub<UserLogins>(qk.userLogins, `${V2}/users/${seg(userId ?? '')}/logins`, userId, usePeriodParams(filters))
}

/** Played marks are a current state: the period is not sent. */
export function useItemMarks(id: string | undefined, filters?: Filters) {
  return useSub<ItemMarks>(qk.itemMarks, `${V2}/items/${seg(id ?? '')}/marks`, id, cleanParams({ ...filters }))
}

export function useLibraryFirstViewDelay(id: string | undefined, filters?: Filters) {
  return useSub<FirstViewDelay>(qk.libraryFirstViewDelay, `${V2}/libraries/${seg(id ?? '')}/first-view-delay`, id, usePeriodParams(filters))
}

export function useInsightsSkips(filters?: Filters) {
  const p = usePeriodParams(filters)
  return useQuery({ queryKey: qk.insights('skips', p), queryFn: get<SkipsInsight>(`${V2}/insights/skips`, p), placeholderData: keepPreviousData })
}
