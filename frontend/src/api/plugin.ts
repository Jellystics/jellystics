// Jellystics plugin sync (backend/internal/handler/plugin_status.go, admin
// only) and the data quality breakdown (backend/internal/handler/v2_quality.go).
import { keepPreviousData, useMutation, useQuery, useQueryClient, type QueryClient } from '@tanstack/react-query'
import { apiFetch } from '@/lib/api'
import { get, V2 } from './client'
import { qk, type Filters } from './keys'
import { usePeriodParams } from './hooks'
import type { ISODate, ISOTime } from './types'

/** Manifest URL of the plugin repository, added in Jellyfin → Plugins → Repositories. */
export const PLUGIN_MANIFEST_URL = 'https://raw.githubusercontent.com/Jellystics/jellystics-plugin-playbackreporting/master/manifest.json'

export interface PluginStatus {
  detected: boolean
  pluginVersion: string | null
  schemaVersion: number | null
  supportedSchemaVersion: number
  cursorPlays: number
  lastCheckedAt: ISOTime | null
  lastSuccessAt: ISOTime | null
  lastError: string | null
  /** The plugin synced within the last 2 minutes: it is the source of truth for plays. */
  healthy: boolean
  /** Plays recorded by the plugin stored in Jellystics. */
  playsSynced: number
}

export type PushState = 'idle' | 'running' | 'done' | 'error' | 'interrupted'
export interface PushStatus {
  state: PushState
  total: number
  sent: number
  imported: number
  skipped: number
  rejected: number
  startedAt: ISOTime | null
  finishedAt: ISOTime | null
  error: string | null
  /** The next start continues after the last batch the plugin acknowledged. */
  resumable: boolean
}

export const pluginKeys = {
  status: ['plugin', 'status'] as const,
  push: ['plugin', 'push'] as const,
}

export function usePluginStatus() {
  return useQuery({ queryKey: pluginKeys.status, queryFn: ({ signal }) => apiFetch<PluginStatus>('/api/plugin/status', { signal }), refetchInterval: 30_000 })
}

/** Polls every 2 s while a push runs. */
export function usePushStatus() {
  return useQuery({
    queryKey: pluginKeys.push,
    queryFn: ({ signal }) => apiFetch<PushStatus>('/api/plugin/push-history/status', { signal }),
    refetchInterval: (q) => (q.state.data?.state === 'running' ? 2_000 : false),
  })
}

export function useStartPush() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => apiFetch<PushStatus>('/api/plugin/push-history', { method: 'POST' }),
    onSuccess: (st) => { qc.setQueryData(pluginKeys.push, st); qc.invalidateQueries({ queryKey: pluginKeys.status }) },
  })
}

/** After a resync / finished push: the sync state and the data quality change. */
export function invalidatePluginData(qc: QueryClient) {
  qc.invalidateQueries({ queryKey: pluginKeys.status })
  qc.invalidateQueries({ queryKey: ['v2', 'data-quality'] })
}

export function useResync() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => apiFetch('/api/plugin/resync', { method: 'POST' }),
    onSuccess: () => { invalidatePluginData(qc) },
  })
}

/** Source keys, from the most to the least precise. */
export const DATA_SOURCES = ['plugin', 'tracker', 'playbackReporting', 'otherImports', 'restore', 'other'] as const
export type DataSource = (typeof DATA_SOURCES)[number]

export interface DataQualityCounts {
  /** Plays of at least 30 s (as every v2 endpoint). */
  views: number
  /** Every playback attempt; the detail counters below are over clicks. */
  clicks: number
  watchSeconds: number
  /** Imported plays whose impossible duration was repaired: a lower bound (see the backend). */
  repaired: number
  /** Plays with playback details (tracks, bitrate, local / remote, pauses). */
  captured: number
  /** Plays with a known stop position. */
  positions: number
  /** Plays with the audio / subtitle track played. */
  streams: number
  transcodes: number
  /** Transcoded plays whose codecs / reasons are known. */
  transcodeDetailed: number
}
export interface DataQualitySource extends DataQualityCounts {
  key: DataSource
  /** 0..1 of the period. */
  viewShare: number
  watchShare: number
}
export interface DataQuality {
  totals: DataQualityCounts
  sources: DataQualitySource[]
  pluginCollectedSince: ISODate | null
  collectedSince: ISODate | null
  /** Plays (all time) without a usable date, excluded from every statistic. */
  undated: number
}

export function useDataQuality(filters?: Filters) {
  const p = usePeriodParams(filters)
  return useQuery({ queryKey: qk.dataQuality(p), queryFn: get<DataQuality>(`${V2}/data-quality`, p), placeholderData: keepPreviousData })
}
