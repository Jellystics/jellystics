// Phase 5 analyses: response types and React Query hooks of /api/v2/insights/*,
// /discover/*, /playback/*, /recap/:userId/:period and /users/:id/languages
// (backend/internal/handler/v2_{insights,discover,playback,recap}.go).
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { get, seg, V2 } from './client'
import { usePeriodParams } from './hooks'
import { qk, type Filters } from './keys'
import type { ByType, GroupKind, ISODate, ISOTime, ItemKind, MethodSplit, RefUser } from './types'

// ── /insights ────────────────────────────────────────────────────────────────
export interface CompletionInsight {
  /** View units per 10 % completion bucket (index 9 = 90–100 %). */
  distribution: number[]
  /** Share of completed units (≥ 90 %) per family; null without data. */
  byType: ByType<number | null>
  units: number
}
export interface SessionsInsight {
  /** Playback rows per length bucket: < 1 min, 1–5, 5–15, 15–30, 30–45, 45–60 min, 1–1.5 h, 1.5–2 h, ≥ 2 h. */
  buckets: number[]
  sessions: number
  medianSeconds: number | null
}
export interface ResumesInsight { fromStart: number; resumed: number; byType: ByType<number | null> }
export interface RewatchRow {
  id: string
  kind: ItemKind
  name: string
  /** Series (episode) or album (track). */
  parentId: string | null
  parentName: string | null
  views: number
  user: RefUser
}
export interface GenresInsight {
  bucket: 'month'
  /** YYYY-MM months overlapping the range (leading empty months trimmed on ranges > 400 days). */
  months: string[]
  /** Top 3 genres by watch seconds (per month). */
  series: { name: string; values: number[] }[]
  other: number[]
}
/** Views per local hour (24 values) per family. */
export type MomentsInsight = ByType<number[]>
export interface PauseRow { id: string; kind: GroupKind; name: string; ratio: number; pausedSeconds: number; pauseCount: number; watchSeconds: number }
export interface BingeRow { id: string; name: string; binges: number; users: number }
export interface RhythmInsight { weekHour: number[][] }

// ── /discover ────────────────────────────────────────────────────────────────
export interface CatalogItem { id: string; kind: GroupKind; name: string; dateAdded: ISODate | null; sizeBytes: number }
export interface UnwatchedResult { count: number; catalog: number; sizeBytes: number; items: CatalogItem[] }
export interface AbandonedRow {
  id: string
  kind: 'movie' | 'series'
  name: string
  user: RefUser
  progress: number
  lastPlayed: ISOTime
  positionSeconds?: number
  episodesViewed?: number
  episodesTotal?: number
}
export interface AbandonedResult { count: number; items: AbandonedRow[] }
export interface AddedResult { count: number; seen: number; items: (CatalogItem & { seen: boolean })[] }
export interface StoragePoint { id: string; kind: GroupKind; name: string; sizeBytes: number; views: number }

// ── /playback ────────────────────────────────────────────────────────────────
export interface MethodDay extends MethodSplit { date: ISODate }
/** `daily`: one point per day, or per ISO week (date = its Monday) when bucket is "week" (ranges > 400 days). */
export interface PlaybackMethods { bucket: 'day' | 'week'; totals: MethodSplit; previous?: MethodSplit; daily: MethodDay[] }
export interface ClientRow {
  client: string
  clicks: number
  views: number
  transcodes: number
  transcodeRatio: number | null
  /** Clicks that never became a view (< 30 s). */
  falseStarts: number
  falseStartRatio: number | null
}
export interface LabelCount { label: string; views: number }
export interface ConversionRow { videoFrom: string; videoTo: string; audioFrom: string; audioTo: string; views: number }
export interface CodecPair { videoFrom: string; videoTo: string; views: number }
export interface PlaybackQuality {
  /** Local date of the first row captured with the new fields (migration 011); null = nothing yet. */
  collectedSince: ISODate | null
  /** Views of the range carrying captured fields. */
  captured: number
  views: number
  transcodes: number
  resolution: { served: Record<'4K' | '1080p' | '720p' | 'SD', number>; sourceUhd: number; downscaledUhd: number }
  hdr: { native: number; toneMapped: number; sdr: number }
  reasons: LabelCount[]
  /** Top 12 video + audio conversions. */
  conversions: ConversionRow[]
  /** Every video source → output codec pair (untruncated), for the codec matrix. */
  codecPairs: CodecPair[]
  streams: { audioOnly: number; videoOnly: number; both: number }
  hwAccel: LabelCount[]
  /** bit/s; buckets: < 2, 2–5, 5–10, 10–20, 20–40, ≥ 40 Mb/s. */
  bitrate: { samples: number; avg: number | null; p95: number | null; buckets: number[] }
}
export interface PlaybackNetwork {
  collectedSince: ISODate | null
  captured: number
  local: number
  remote: number
  unknown: number
  places: { country: string; city: string | null; views: number }[]
}
export interface UserLanguages {
  collectedSince: ISODate | null
  captured: number
  split: { originalSubtitled: number; original: number; dubbed: number; unknown: number }
  audio: LabelCount[]
  subtitles: LabelCount[]
  subtitleOff: number
}

// ── /recap ───────────────────────────────────────────────────────────────────
export interface RecapTotals { watchSeconds: number; views: number; activeDays: number }
export interface RecapTop { id: string; name: string; watchSeconds: number; views: number }
export interface RecapBucket { key: string; watchSeconds: number; views: number; newArtists: number }
export interface Recap {
  user: RefUser
  period: string
  granularity: 'month' | 'day'
  from: ISODate
  /** Exclusive. */
  to: ISODate
  totals: RecapTotals
  previous: RecapTotals
  topSeries: RecapTop[]
  topMovies: RecapTop[]
  topArtists: RecapTop[]
  buckets: RecapBucket[]
  records: {
    bestBucket: RecapBucket | null
    binge: { seriesId: string; seriesName: string; episodes: number; date: ISODate } | null
    longestStreak: number
  }
  ranking: { id: string; kind: GroupKind; name: string; parentName: string | null; views: number; rank: number; previousRank: number | null }[]
}

// ── hooks ────────────────────────────────────────────────────────────────────
function useSection<T>(section: 'insights' | 'discover' | 'playback', name: string, filters?: Filters) {
  const p = usePeriodParams(filters)
  return useQuery({ queryKey: qk[section](name, p), queryFn: get<T>(`${V2}/${section}/${name}`, p), placeholderData: keepPreviousData })
}

export const useInsightsCompletion = (f?: Filters) => useSection<CompletionInsight>('insights', 'completion', f)
export const useInsightsSessions = (f?: Filters) => useSection<SessionsInsight>('insights', 'sessions', f)
export const useInsightsResumes = (f?: Filters) => useSection<ResumesInsight>('insights', 'resumes', f)
export const useInsightsRewatches = (f?: Filters) => useSection<RewatchRow[]>('insights', 'rewatches', f)
export const useInsightsGenres = (f?: Filters) => useSection<GenresInsight>('insights', 'genres', f)
export const useInsightsMoments = (f?: Filters) => useSection<MomentsInsight>('insights', 'moments', f)
export const useInsightsPauses = (f?: Filters) => useSection<PauseRow[]>('insights', 'pauses', f)
export const useInsightsBinges = (f?: Filters) => useSection<BingeRow[]>('insights', 'binges', f)
export const useInsightsRhythm = (f?: Filters) => useSection<RhythmInsight>('insights', 'rhythm', f)

export const useDiscoverUnwatched = (f?: Filters) => useSection<UnwatchedResult>('discover', 'unwatched', f)
export const useDiscoverAbandoned = (f?: Filters) => useSection<AbandonedResult>('discover', 'abandoned', f)
export const useDiscoverAdded = (f?: Filters) => useSection<AddedResult>('discover', 'added', f)
export const useDiscoverStorage = (f?: Filters) => useSection<StoragePoint[]>('discover', 'storage', f)

export const usePlaybackMethods = (f?: Filters) => useSection<PlaybackMethods>('playback', 'methods', f)
export const usePlaybackClients = (f?: Filters) => useSection<ClientRow[]>('playback', 'clients', f)
export const usePlaybackQuality = (f?: Filters) => useSection<PlaybackQuality>('playback', 'quality', f)
export const usePlaybackNetwork = (f?: Filters) => useSection<PlaybackNetwork>('playback', 'network', f)

/**
 * Recap of one user for a period `YYYY` or `YYYY-MM` (the URL period is not
 * used). The previous recap stays on screen while switching periods of the
 * same user, never across users.
 */
export function useRecap(userId: string | undefined, period: string | undefined) {
  return useQuery({
    queryKey: qk.recap(userId ?? '', period ?? ''),
    queryFn: get<Recap>(`${V2}/recap/${seg(userId ?? '')}/${seg(period ?? '')}`, {}),
    enabled: !!userId && !!period,
    placeholderData: (prev, prevQuery) => (prevQuery?.queryKey[2] === (userId ?? '') ? prev : undefined),
  })
}

export function useUserLanguages(userId: string | undefined, filters?: Filters) {
  const p = usePeriodParams(filters)
  return useQuery({
    queryKey: qk.languages(userId ?? '', p),
    queryFn: get<UserLanguages>(`${V2}/users/${seg(userId ?? '')}/languages`, p),
    enabled: !!userId,
    placeholderData: keepPreviousData,
  })
}
