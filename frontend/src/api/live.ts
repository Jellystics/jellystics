// Live Jellyfin sessions: GET /sessions/current (initial fetch) kept up to
// date by the socket `sessions` event (same payload, jellyfin.SessionInfo).
import { useCallback } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { apiFetch } from '@/lib/api'
import { useSocketEvent } from '@/lib/socket'

export interface JellyfinSession {
  Id: string
  UserId: string
  UserName: string
  Client: string
  DeviceName: string
  ApplicationVersion?: string
  NowPlayingItem?: {
    Id: string
    Name: string
    Type: string
    SeriesName?: string | null
    SeriesId?: string | null
    AlbumId?: string | null
    Album?: string | null
    AlbumArtist?: string | null
    RunTimeTicks?: number | null
    IndexNumber?: number | null
    ParentIndexNumber?: number | null
    MediaSources?: { Bitrate?: number | null }[] | null
  } | null
  PlayState?: { IsPaused: boolean; PositionTicks?: number | null; PlayMethod?: string | null } | null
  TranscodingInfo?: { Bitrate?: number | null; IsVideoDirect?: boolean; IsAudioDirect?: boolean; HardwareAccelerationType?: string | null } | null
  RemoteEndPoint?: string | null
}

export const qkLive = { sessions: ['live', 'sessions'] as const }

const playing = (data: unknown): JellyfinSession[] =>
  Array.isArray(data) ? (data as JellyfinSession[]).filter((s) => s && s.NowPlayingItem).sort((a, b) => a.Id.localeCompare(b.Id)) : []

/** Sessions currently playing something (initial REST fetch + live socket updates). */
export function useLiveSessions() {
  const qc = useQueryClient()
  useSocketEvent('sessions', useCallback((data: unknown) => {
    if (Array.isArray(data)) qc.setQueryData(qkLive.sessions, playing(data))
  }, [qc]))
  return useQuery({
    queryKey: qkLive.sessions,
    queryFn: async ({ signal }) => playing(await apiFetch<unknown>('/sessions/current', { signal })),
    staleTime: 30_000,
  })
}

/** Outgoing bitrate of a session in bit/s (transcode output, else the source bitrate). */
export function sessionBitrate(s: JellyfinSession): number | null {
  return s.TranscodingInfo?.Bitrate ?? s.NowPlayingItem?.MediaSources?.[0]?.Bitrate ?? null
}

/** Primary image of a Jellyfin item through the backend proxy. */
export function itemImageUrl(id: string, width = 240): string {
  return `/proxy/Items/Images/Primary/?id=${encodeURIComponent(id)}&fillWidth=${width}&quality=90`
}
