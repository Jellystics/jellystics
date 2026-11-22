import type { TFunction } from 'i18next'
import { itemImageUrl, type JellyfinSession } from '@/api/live'
import type { LiveSession, MediaType } from '@/ui'
import { methodOf } from '../history/links'

const TICKS = 10_000_000
const seg = encodeURIComponent

export function sessionFamily(s: JellyfinSession): MediaType {
  const type = s.NowPlayingItem?.Type
  return type === 'Episode' ? 'series' : type === 'Audio' ? 'music' : 'movie'
}

/** Maps a Jellyfin session to the SessionCard model (links to the typed entity routes). */
export function toLiveSession(s: JellyfinSession, t: TFunction): LiveSession {
  const it = s.NowPlayingItem!
  const type = sessionFamily(s)
  let itemHref = `/movies/${seg(it.Id)}`
  let subtitle = s.Client
  let title = it.Name
  let artId = it.Id
  if (type === 'series') {
    itemHref = it.SeriesId ? `/series/${seg(it.SeriesId)}/episodes/${seg(it.Id)}` : `/items/${seg(it.Id)}`
    title = it.SeriesName || it.Name
    const ep = it.ParentIndexNumber != null && it.IndexNumber != null ? `${t('app.live.episodeShort', { season: it.ParentIndexNumber, episode: it.IndexNumber })} · ` : ''
    subtitle = it.SeriesName ? `${ep}${it.Name}` : s.Client
    artId = it.SeriesId || it.Id
  } else if (type === 'music') {
    itemHref = it.AlbumId ? `/albums/${seg(it.AlbumId)}?track=${seg(it.Id)}` : `/items/${seg(it.Id)}`
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
    method: methodOf(s.PlayState?.PlayMethod) ?? (s.TranscodingInfo ? 'transcode' : 'direct'),
    paused: !!s.PlayState?.IsPaused,
    positionSec: (s.PlayState?.PositionTicks ?? 0) / TICKS,
    runtimeSec: (it.RunTimeTicks ?? 0) / TICKS,
    artSrc: itemImageUrl(artId, 160),
  }
}
