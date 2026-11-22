// Phase 3 endpoints (backend/internal/handler/v2_users_extra.go):
// /users/retention, /users/share and /concurrency.
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { apiFetch } from '@/lib/api'
import { cleanParams, type Filters, type QueryParams } from './keys'
import { usePeriodParams } from './hooks'
import type { ISODate, ISOTime, MediaFamily, RefUser } from './types'

const V2 = '/api/v2'

/** Weekly cohorts: users active in `week` (Monday), and how many are active again k weeks later. */
export interface RetentionRow {
  week: ISODate
  cohort: number
  /** Index k = S+k; null when that week is after the range. retained[0] = cohort. */
  retained: (number | null)[]
}
export interface UserRetention { offsets: number; rows: RetentionRow[] }

export interface ShareUser {
  id: string
  name: string
  watchSeconds: number
  /** Share of all watch time in the range (0..1). */
  share: number
  /** Share of each period's watch time (0..1), aligned on `periods`. */
  values: number[]
}
export interface UserShare { bucket: 'week' | 'month'; periods: ISODate[]; users: ShareUser[] }

export interface ConcurrencySession {
  user: RefUser
  itemId: string
  name: string
  parentName?: string
  kind: MediaFamily
  /** Clipped to the day, local offset. */
  start: ISOTime
  end: ISOTime
}
export interface ConcurrencyDay {
  date: ISODate
  peak: number
  peakAt: ISOTime | null
  slotMinutes: number
  /** Max simultaneous sessions per slot, from local midnight. */
  slots: number[]
  sessions: ConcurrencySession[]
}

const get = <T,>(path: string, params: QueryParams) => ({ signal }: { signal: AbortSignal }) => apiFetch<T>(path, { params, signal })

export const qkExtra = {
  retention: (p: QueryParams) => ['v2', 'users-extra', 'retention', p] as const,
  share: (p: QueryParams) => ['v2', 'users-extra', 'share', p] as const,
  concurrency: (p: QueryParams) => ['v2', 'concurrency', p] as const,
}

export function useUserRetention(filters?: Filters) {
  const p = usePeriodParams(filters)
  return useQuery({ queryKey: qkExtra.retention(p), queryFn: get<UserRetention>(`${V2}/users/retention`, p), placeholderData: keepPreviousData })
}

export function useUserShare(filters?: Filters) {
  const p = usePeriodParams(filters)
  return useQuery({ queryKey: qkExtra.share(p), queryFn: get<UserShare>(`${V2}/users/share`, p), placeholderData: keepPreviousData })
}

/** Sessions of one local day (default: today on the server); the period is not used. */
export function useConcurrency(date?: string, filters?: Filters, opts: { refetchInterval?: number } = {}) {
  const p = cleanParams({ date, ...filters })
  return useQuery({
    queryKey: qkExtra.concurrency(p),
    queryFn: get<ConcurrencyDay>(`${V2}/concurrency`, p),
    placeholderData: keepPreviousData,
    refetchInterval: opts.refetchInterval,
  })
}
