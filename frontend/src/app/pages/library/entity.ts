// Helpers shared by the phase 4 entity pages (libraries, library, movie,
// series, episode, artist, album).
import { ApiError } from '@/lib/api'
import { itemImage } from '@/lib/images'
import type { LibraryRef, LibraryType } from '@/api'
import { useSetCrumbLabel, useSetCrumbLabels, type CrumbLabels } from '@/app/shell/crumbs'

/**
 * Breadcrumb of an entity page: the name replaces the last crumb and the
 * library (plus any extra named label) refines the intermediate ones.
 */
export function useEntityCrumbs(name: string | undefined, library: LibraryRef | null | undefined, extra?: CrumbLabels) {
  useSetCrumbLabel(name)
  useSetCrumbLabels({ library: library?.name, libraryId: library?.id, ...extra })
}

/** History state of the row links opening a side panel (closing then goes back). */
export const PANEL_STATE = { panel: true }

/** Jellyfin primary image through the backend proxy. */
export const itemArt = (id: string, width = 320) => itemImage(id, width)

export const isNotFound = (e: unknown) => e instanceof ApiError && e.status === 404

/** Media family of a library type. */
export const libraryFamily = (t: LibraryType) => (t === 'movies' ? 'movie' : t === 'tvshows' ? 'series' : 'music') as 'movie' | 'series' | 'music'

/** "20 nov. · 21:10" (year added when not the current one). */
export function formatWhen(iso: string, locale = 'fr', now = new Date()): string {
  const d = new Date(iso)
  const day = new Intl.DateTimeFormat(locale, { day: 'numeric', month: 'short', ...(d.getFullYear() !== now.getFullYear() ? { year: 'numeric' } : {}) }).format(d)
  const time = new Intl.DateTimeFormat(locale, { hour: '2-digit', minute: '2-digit', hour12: false }).format(d)
  return `${day} · ${time}`
}

/** Share helper: part / whole, null when whole is 0. */
export const share = (part: number, whole: number) => (whole > 0 ? part / whole : null)
