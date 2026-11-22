import type { SearchItem } from '@/api'

/** Typed page route of a search result (undefined when its parent is unknown). */
export function searchItemPath(it: SearchItem): string | undefined {
  const id = encodeURIComponent(it.id)
  switch (it.kind) {
    case 'movie': return `/movies/${id}`
    case 'series': return `/series/${id}`
    case 'album': return `/albums/${id}`
    case 'artist': return `/artists/${id}`
    case 'episode': return it.parentId ? `/series/${encodeURIComponent(it.parentId)}/episodes/${id}` : undefined
    case 'track': return it.parentId ? `/albums/${encodeURIComponent(it.parentId)}?track=${id}` : undefined
  }
}
