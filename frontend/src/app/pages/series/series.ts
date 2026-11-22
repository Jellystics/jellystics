import type { CadenceBucket, ReleaseLagBucket, SeriesDetail, SeriesEpisode } from '@/api'

export const CADENCE: CadenceBucket[] = ['<1min', '1-15min', '15-60min', 'sameDay', 'nextDay', '2-7d', '>7d']
export const RELEASE_LAG: ReleaseLagBucket[] = ['<24h', '1-7d', '1-4w', '1-6m', '6-12m', '>1y']

/** Counts in the fixed bucket order (missing buckets = 0). */
export function fixedBuckets<B extends string>(order: B[], rows: { bucket: B; count: number }[] | undefined): number[] {
  const m = new Map((rows ?? []).map((r) => [r.bucket, r.count]))
  return order.map((b) => m.get(b) ?? 0)
}

export interface SeasonView { number: number; special: boolean; episodes: SeriesEpisode[]; retention: number[] }

/**
 * Seasons aligned with their retention row (viewers per episode). Specials
 * (season 0 or `special: true`) are kept for the episode list but flagged so
 * the retention chart can leave them out.
 */
export function seasonViews(data: Pick<SeriesDetail, 'seasons' | 'retention'>): SeasonView[] {
  return data.seasons.map((s, i) => ({
    number: s.number,
    special: s.number === 0 || !!s.special,
    episodes: s.episodes,
    retention: data.retention[i] ?? s.episodes.map((e) => e.viewers),
  }))
}
