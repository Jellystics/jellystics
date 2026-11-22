import type { GroupKind, HistoryItem, MediaType as ApiMediaType } from '@/api'
import type { MediaType, PlayMethod } from '@/ui'

/** Route of a movie / series / album. */
export function groupHref(kind: GroupKind, id: string): string {
  const seg = encodeURIComponent(id)
  if (kind === 'series') return `/series/${seg}`
  if (kind === 'album') return `/albums/${seg}`
  return `/movies/${seg}`
}

/** Route of the item actually played (episode page, album with the track panel, movie). */
export function itemHref(item: HistoryItem): string {
  const id = encodeURIComponent(item.id)
  if (item.kind === 'episode') return item.parentId ? `/series/${encodeURIComponent(item.parentId)}/episodes/${id}` : `/items/${id}`
  if (item.kind === 'track') return item.parentId ? `/albums/${encodeURIComponent(item.parentId)}?track=${id}` : `/items/${id}`
  return `/movies/${id}`
}

export const familyOf = (t: ApiMediaType | string): MediaType => (t === 'Episode' ? 'series' : t === 'Audio' ? 'music' : 'movie')
export const groupFamily = (k: GroupKind): MediaType => (k === 'album' ? 'music' : k)

/** Normalises a stored PlayMethod (DirectPlay / DirectStream / Transcode / direct …). */
export function methodOf(raw: string | null | undefined): PlayMethod | null {
  switch ((raw ?? '').toLowerCase()) {
    case 'directplay': case 'direct': return 'direct'
    case 'directstream': case 'stream': return 'stream'
    case 'transcode': return 'transcode'
  }
  return null
}

export const userHref = (id: string) => `/users/${encodeURIComponent(id)}`

/** Wall-clock hour (fractional) of an ISO time with offset, as written by the server (its time zone). */
export function wallHours(iso: string): number {
  const m = /T(\d{2}):(\d{2})/.exec(iso)
  return m ? Number(m[1]) + Number(m[2]) / 60 : 0
}
/** Wall-clock HH:MM of an ISO time with offset. */
export function wallClock(iso: string): string {
  return /T(\d{2}:\d{2})/.exec(iso)?.[1] ?? ''
}
/** Local calendar date (YYYY-MM-DD) of an ISO time with offset. */
export const wallDate = (iso: string) => iso.slice(0, 10)
/** Parses YYYY-MM-DD as a local Date (for display formatting only). */
export function dateOf(s: string): Date {
  const [y, m, d] = s.slice(0, 10).split('-').map(Number)
  return new Date(y, (m || 1) - 1, d || 1)
}

/** YYYY-MM shifted by n months. */
export function shiftMonth(month: string, n: number): string {
  const [y, m] = month.split('-').map(Number)
  const d = new Date(Date.UTC(y, m - 1 + n, 1))
  return `${d.getUTCFullYear()}-${String(d.getUTCMonth() + 1).padStart(2, '0')}`
}
