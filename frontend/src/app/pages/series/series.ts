import type { CadenceBucket, ReleaseLagBucket, SeriesDetail, SeriesEpisode } from '@/api'

export const CADENCE: CadenceBucket[] = ['<1min', '1-15min', '15-60min', 'sameDay', 'nextDay', '2-7d', '>7d']
export const RELEASE_LAG: ReleaseLagBucket[] = ['<24h', '1-7d', '1-4w', '1-6m', '6-12m', '>1y']

/** Counts in the fixed bucket order (missing buckets = 0). */
export function fixedBuckets<B extends string>(order: B[], rows: { bucket: B; count: number }[] | undefined): number[] {
  const m = new Map((rows ?? []).map((r) => [r.bucket, r.count]))
  return order.map((b) => m.get(b) ?? 0)
}

export interface SeasonView { number: number; special: boolean; episodes: SeriesEpisode[]; retention: number[] }

type RetentionRow = unknown

/** Viewers per episode of one retention row (plain numbers or tagged objects). */
function retentionValues(row: RetentionRow): { values: number[]; special: boolean } {
  const toNum = (v: unknown) => (typeof v === 'number' ? v : typeof v === 'object' && v ? Number((v as { viewers?: number; value?: number }).viewers ?? (v as { value?: number }).value ?? 0) : 0)
  if (Array.isArray(row)) return { values: row.map(toNum), special: false }
  if (row && typeof row === 'object') {
    const o = row as { values?: unknown[]; episodes?: unknown[]; special?: boolean }
    return { values: (o.values ?? o.episodes ?? []).map(toNum), special: !!o.special }
  }
  return { values: [], special: false }
}

/**
 * Seasons aligned with their retention row. Specials (season 0 or tagged
 * `special: true`) are kept for the episode list but flagged so the
 * retention chart can leave them out.
 */
export function seasonViews(data: Pick<SeriesDetail, 'seasons' | 'retention'>): SeasonView[] {
  return data.seasons.map((s, i) => {
    const r = retentionValues((data.retention as RetentionRow[])[i])
    const special = s.number === 0 || !!(s as { special?: boolean }).special || r.special
    return { number: s.number, special, episodes: s.episodes, retention: r.values.length ? r.values : s.episodes.map((e) => e.viewers) }
  })
}
