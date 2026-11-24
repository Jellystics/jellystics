// API of the « Mes statistiques » page: ./api/<path> relative to the page, i.e. <basePath>/Jellystics/Me/api/...,
// relayed by the Jellystics plugin to Jellystics /api/me/<path> with the user id taken from the Jellyfin token.
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { authHeader, type Session } from './session'
import type { MeRange } from './period'
import type { MeDevices, MeHabits, MeHistory, MeOverview, MeProfile, MeScatter, MeTop, MeTypes } from './types'

/** Why the statistics cannot be shown. */
export type UnavailableReason = 'signedOut' | 'notLinked' | 'unreachable' | 'relinking' | 'forbidden' | 'unknownUser' | 'error'

export class MeApiError extends Error {
  constructor(public status: number, public reason: UnavailableReason, public body: unknown) { super(`HTTP ${status}`) }
}

const errorCode = (body: unknown) => {
  const e = body && typeof body === 'object' ? (body as { error?: unknown }).error : undefined
  return typeof e === 'string' ? e.toLowerCase().replace(/[\s_-]/g, '') : ''
}

/**
 * Reason of a failed call. The plugin answers 503 {error:"notLinked"} before Jellystics linked it, 502/504
 * {error:"unreachable"} when Jellystics does not answer; Jellystics answers 401 {error:"invalid secret"} when the
 * link is stale and 403 {error:"unknown user"} for a user it does not know yet. The plugin answers 502
 * {error:"secretMismatch"} while Jellystics re-links it (next sync). A bare 401 comes from Jellyfin itself: the
 * token is missing or expired.
 */
export function errorReason(status: number, body: unknown): UnavailableReason {
  const code = errorCode(body)
  if (code === 'notlinked') return 'notLinked'
  if (code === 'secretmismatch') return 'relinking'
  if (code === 'unreachable' || code === 'timeout') return 'unreachable'
  if (status === 401) return code === 'invalidsecret' ? 'notLinked' : 'signedOut'
  if (status === 403) return code === 'unknownuser' ? 'unknownUser' : 'forbidden'
  if (status === 503) return 'notLinked'
  if (status === 502 || status === 504) return 'unreachable'
  return 'error'
}

type Params = Record<string, string | number | boolean | undefined | null>

export async function meFetch<T>(session: Session, path: string, params: Params = {}, signal?: AbortSignal): Promise<T> {
  const qs = new URLSearchParams()
  Object.entries(params).forEach(([k, v]) => { if (v !== undefined && v !== null && v !== '' && v !== false) qs.set(k, v === true ? '1' : String(v)) })
  const url = `./api/${path}${qs.toString() ? `?${qs}` : ''}`
  const res = await fetch(url, { headers: { Accept: 'application/json', Authorization: authHeader(session.token) }, signal })
  const text = await res.text()
  let data: unknown
  let parsed = true
  try { data = text ? JSON.parse(text) : undefined } catch { data = text; parsed = false }
  if (!res.ok) throw new MeApiError(res.status, errorReason(res.status, data), data)
  // A 2xx page that is not JSON is the plugin's fallback HTML or a misroute: the relay is not in place.
  if (!parsed || data === undefined) throw new MeApiError(res.status, 'notLinked', data)
  return data as T
}

const retry = (count: number, err: unknown) => !(err instanceof MeApiError && err.reason !== 'error') && count < 1
const rangeParams = (r: MeRange) => ({ from: r.from, to: r.to, compare: r.compare })

function useMe<T>(session: Session, path: string, params: Params, opts: { keep?: boolean } = {}) {
  return useQuery({
    queryKey: ['me', session.userId, path, params],
    queryFn: ({ signal }) => meFetch<T>(session, path, params, signal),
    staleTime: 60_000,
    retry,
    placeholderData: opts.keep ? keepPreviousData : undefined,
  })
}

export const useMeProfile = (s: Session) => useMe<MeProfile>(s, 'profile', {})
export const useMeOverview = (s: Session, r: MeRange) => useMe<MeOverview>(s, 'overview', rangeParams(r))
export const useMeTop = (s: Session, r: MeRange) => useMe<MeTop>(s, 'top', { from: r.from, to: r.to })
export const useMeTypes = (s: Session, r: MeRange) => useMe<MeTypes>(s, 'types', { from: r.from, to: r.to })
export const useMeHabits = (s: Session, r: MeRange) => useMe<MeHabits>(s, 'habits', { from: r.from, to: r.to })
export const useMeDevices = (s: Session, r: MeRange) => useMe<MeDevices>(s, 'devices', { from: r.from, to: r.to })
export const useMeScatter = (s: Session, r: MeRange) => useMe<MeScatter>(s, 'scatter', { from: r.from, to: r.to })
export const useMeHistory = (s: Session, r: MeRange, page: number, pageSize: number) =>
  useMe<MeHistory>(s, 'history', { from: r.from, to: r.to, page, pageSize, sort: 'startedAt', desc: true }, { keep: true })
