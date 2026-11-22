// Helpers shared by the phase 4 entity pages (libraries, library, movie,
// series, episode, artist, album).
import { ApiError } from '@/lib/api'
import type { HistoryItem, LibraryType } from '@/api'

/** Jellyfin primary image through the backend proxy. */
export const itemArt = (id: string, width = 320) => `/proxy/Items/Images/Primary/?id=${encodeURIComponent(id)}&fillWidth=${width}&quality=90`

export const isNotFound = (e: unknown) => e instanceof ApiError && e.status === 404

export type EntityKind = 'movie' | 'series' | 'album' | 'artist' | 'episode' | 'track'

/** Typed route of an entity (episodes need their series, tracks their album). */
export function entityPath(kind: EntityKind, id: string, parentId?: string | null): string {
  const e = encodeURIComponent
  switch (kind) {
    case 'movie': return `/movies/${e(id)}`
    case 'series': return `/series/${e(id)}`
    case 'album': return `/albums/${e(id)}`
    case 'artist': return `/artists/${e(id)}`
    case 'episode': return parentId ? `/series/${e(parentId)}/episodes/${e(id)}` : `/items/${e(id)}`
    case 'track': return parentId ? `/albums/${e(parentId)}?track=${e(id)}` : `/items/${e(id)}`
  }
}

/** Route of the item of a history row. */
export const historyItemPath = (it: HistoryItem) => entityPath(it.kind, it.id, it.parentId)

/** Media family of a library type. */
export const libraryFamily = (t: LibraryType) => (t === 'movies' ? 'movie' : t === 'tvshows' ? 'series' : 'music') as 'movie' | 'series' | 'music'

/** File size: 412 Go / 18,2 Go / 950 Mo (decimal units, locale digits). */
export function formatBytes(bytes: number, locale = 'fr'): string {
  const units = locale.startsWith('fr') ? ['o', 'Ko', 'Mo', 'Go', 'To'] : ['B', 'KB', 'MB', 'GB', 'TB']
  let v = Math.max(0, bytes), i = 0
  while (v >= 1000 && i < units.length - 1) { v /= 1000; i++ }
  const digits = v >= 100 || i === 0 ? 0 : 1
  return `${new Intl.NumberFormat(locale, { maximumFractionDigits: digits }).format(v)} ${units[i]}`
}

/** "20 nov. · 21:10" (year added when not the current one). */
export function formatWhen(iso: string, locale = 'fr', now = new Date()): string {
  const d = new Date(iso)
  const day = new Intl.DateTimeFormat(locale, { day: 'numeric', month: 'short', ...(d.getFullYear() !== now.getFullYear() ? { year: 'numeric' } : {}) }).format(d)
  const time = new Intl.DateTimeFormat(locale, { hour: '2-digit', minute: '2-digit', hour12: false }).format(d)
  return `${day} · ${time}`
}

/** Map a raw PlayMethod (DirectPlay…) or v2 method to the badge method. */
export function playMethod(m: string): 'direct' | 'stream' | 'transcode' | null {
  const s = m.toLowerCase()
  if (s === 'directplay' || s === 'direct') return 'direct'
  if (s === 'directstream' || s === 'stream') return 'stream'
  if (s === 'transcode') return 'transcode'
  return null
}

/** Share helper: part / whole, null when whole is 0. */
export const share = (part: number, whole: number) => (whole > 0 ? part / whole : null)
