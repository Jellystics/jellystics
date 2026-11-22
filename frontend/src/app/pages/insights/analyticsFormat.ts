// Formatting and routing helpers of the analysis pages.

/** Route of a title (movie / series / album) or a played item (episode / track). */
export function entityPath(kind: string, id: string, parentId?: string | null): string | undefined {
  const e = encodeURIComponent
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

/** Family of a title kind, for type colours and badges. */
export const familyOf = (kind: string): 'movie' | 'series' | 'music' => (kind === 'movie' ? 'movie' : kind === 'series' || kind === 'episode' ? 'series' : 'music')

/** Bytes → "1,2 To" / "48 Go" / "350 Mo" (decimal units, like file managers). */
export function formatBytes(bytes: number, locale = 'fr'): string {
  const units = locale.startsWith('fr') ? ['o', 'Ko', 'Mo', 'Go', 'To'] : ['B', 'KB', 'MB', 'GB', 'TB']
  let v = bytes, i = 0
  while (v >= 1000 && i < units.length - 1) { v /= 1000; i++ }
  const digits = v < 10 && i > 0 ? 1 : 0
  return `${new Intl.NumberFormat(locale, { maximumFractionDigits: digits }).format(v)} ${units[i]}`
}

/** Bit/s → "8,5 Mb/s". */
export function formatBitrate(bps: number, locale = 'fr'): string {
  const mb = bps / 1_000_000
  return `${new Intl.NumberFormat(locale, { maximumFractionDigits: mb < 10 ? 1 : 0 }).format(mb)} Mb/s`
}

/** Hours with one decimal under 10 (y axes of watch-time charts). */
export function formatHours(seconds: number, locale = 'fr'): string {
  const h = seconds / 3600
  return `${new Intl.NumberFormat(locale, { maximumFractionDigits: h < 10 ? 1 : 0 }).format(h)} h`
}

/** "2026-11" → "nov." (+ year when it is not the current one). */
export function formatMonth(month: string, locale = 'fr', withYear = false): string {
  const [y, m] = month.split('-').map(Number)
  const d = new Date(y, m - 1, 1)
  return new Intl.DateTimeFormat(locale, { month: 'short', ...(withYear || y !== new Date().getFullYear() ? { year: '2-digit' } : {}) }).format(d)
}

