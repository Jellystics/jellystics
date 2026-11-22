import type { TFunction } from 'i18next'
import { itemImageUrl, type JellyfinSession } from '@/api/live'
import type { LiveSession, MediaType } from '@/ui'
import { entityPath, mediaTypeFamily, playMethodOf } from '@/lib/routes'

const TICKS = 10_000_000

export function sessionFamily(s: JellyfinSession): MediaType {
  return mediaTypeFamily(s.NowPlayingItem?.Type)
}

/** Maps a Jellyfin session to the SessionCard model (links to the typed entity routes). */
export function toLiveSession(s: JellyfinSession, t: TFunction): LiveSession {
  const it = s.NowPlayingItem!
  const type = sessionFamily(s)
  let itemHref = entityPath('movie', it.Id)
  let subtitle = s.Client
  let title = it.Name
  let artId = it.Id
  if (type === 'series') {
    itemHref = entityPath('episode', it.Id, it.SeriesId)
    title = it.SeriesName || it.Name
    const ep = it.ParentIndexNumber != null && it.IndexNumber != null ? `${t('app.live.episodeShort', { season: it.ParentIndexNumber, episode: it.IndexNumber })} · ` : ''
    subtitle = it.SeriesName ? `${ep}${it.Name}` : s.Client
    artId = it.SeriesId || it.Id
  } else if (type === 'music') {
    itemHref = entityPath('track', it.Id, it.AlbumId)
    subtitle = [it.AlbumArtist, it.Album].filter(Boolean).join(' · ') || s.Client
    artId = it.AlbumId || it.Id
  }
  return {
    id: s.Id,
    title,
    subtitle,
    itemHref,
    type,
    user: { id: s.UserId, name: s.UserName },
    device: s.DeviceName || s.Client,
    method: playMethodOf(s.PlayState?.PlayMethod) ?? (s.TranscodingInfo ? 'transcode' : 'direct'),
    paused: !!s.PlayState?.IsPaused,
    positionSec: (s.PlayState?.PositionTicks ?? 0) / TICKS,
    runtimeSec: (it.RunTimeTicks ?? 0) / TICKS,
    artSrc: itemImageUrl(artId, 160),
  }
}
