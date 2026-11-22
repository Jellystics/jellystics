// Single source of the app's entity routes, media families and play methods.

const e = encodeURIComponent

export type EntityKind = 'movie' | 'series' | 'album' | 'artist' | 'episode' | 'track'
export type MediaFamily = 'movie' | 'series' | 'music'
export type PlayMethodKind = 'direct' | 'stream' | 'transcode'

/**
 * Typed page of an entity, or undefined when it cannot be resolved here:
 * an episode without its series, a track without its album, an unknown kind.
 */
export function entityRoute(kind: string, id: string, parentId?: string | null): string | undefined {
  switch (kind) {
    case 'movie': return `/movies/${e(id)}`
    case 'series': return `/series/${e(id)}`
    case 'album': return `/albums/${e(id)}`
    case 'artist': return `/artists/${e(id)}`
    case 'episode': return parentId ? `/series/${e(parentId)}/episodes/${e(id)}` : undefined
    case 'track': return parentId ? `/albums/${e(parentId)}?track=${e(id)}` : undefined
  }
  return undefined
}

/** Item fallback: /items/:id resolves the kind server-side, then redirects to the typed page. */
export const itemPath = (id: string) => `/items/${e(id)}`

/** Typed page of an entity, falling back to /items/:id when it cannot be resolved here. */
export const entityPath = (kind: string, id: string, parentId?: string | null): string => entityRoute(kind, id, parentId) ?? itemPath(id)

export const userPath = (id: string) => `/users/${e(id)}`

/** Family of an entity kind, for type colours and badges (episode → series, album/track/artist → music). */
export const familyOf = (kind: string): MediaFamily => (kind === 'movie' ? 'movie' : kind === 'series' || kind === 'episode' ? 'series' : 'music')

/** Family of a Jellyfin media type (Movie / Episode / Audio). */
export const mediaTypeFamily = (type: string | null | undefined): MediaFamily => (type === 'Episode' ? 'series' : type === 'Audio' ? 'music' : 'movie')

/** Normalises a stored or live PlayMethod (DirectPlay / DirectStream / Transcode / direct …). */
export function playMethodOf(raw: string | null | undefined): PlayMethodKind | null {
  switch ((raw ?? '').toLowerCase()) {
    case 'directplay': case 'direct': return 'direct'
    case 'directstream': case 'stream': return 'stream'
    case 'transcode': return 'transcode'
  }
  return null
}
