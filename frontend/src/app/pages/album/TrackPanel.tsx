import { useLocation, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { AppLink, Art, Card, EmptyState, ErrorState, SidePanel, Skeleton, StatStrip, StatTile } from '@/ui'
import { useFormat } from '@/lib/format'
import { useSearchParam } from '@/lib/useUrlState'
import { useTrack } from '@/api'
import { UserCell } from '../library/EntityParts'
import { entityPath } from '@/lib/routes'
import { itemArt } from '../library/entity'
import styles from '../library/entity.module.css'

/**
 * Track drawer bound to `?track=` (album and artist pages). Names come from
 * the hosting page, which already knows them; stats from /tracks/:id.
 */
export function TrackPanel({ names }: { names: Record<string, { name: string; albumId?: string | null; albumName?: string | null; artistId?: string | null; artistName?: string | null }> }) {
  const { t } = useTranslation()
  const f = useFormat()
  const [trackId, setTrack] = useSearchParam('track')
  const location = useLocation()
  const navigate = useNavigate()
  const q = useTrack(trackId)
  const known = trackId ? names[trackId] : undefined
  const data = q.data && q.data.item.id === trackId ? q.data : undefined
  const albumId = known?.albumId ?? data?.item.albumId ?? null
  const artistId = known?.artistId ?? data?.item.artistId ?? null
  const albumName = known?.albumName ?? data?.item.albumName ?? null
  const artistName = known?.artistName ?? data?.item.artistName ?? null
  const title = known?.name ?? data?.item.name ?? t('app.album.track')
  const subtitle = [
    artistId && artistName ? <AppLink key="ar" to={entityPath('artist', artistId)}>{artistName}</AppLink> : artistName,
    albumId && albumName ? <AppLink key="al" to={entityPath('album', albumId)}>{albumName}</AppLink> : albumName,
  ].filter(Boolean)
  // Opened from a row link (pushed entry flagged PANEL_STATE): closing goes
  // back instead of stacking another entry; a deep link just drops ?track=.
  const close = () => {
    if ((location.state as { panel?: boolean } | null)?.panel) navigate(-1)
    else setTrack(undefined)
  }
  return (
    <SidePanel
      open={!!trackId}
      onOpenChange={(o) => { if (!o) close() }}
      title={title}
      subtitle={subtitle.length ? <>{subtitle.map((s, i) => <span key={i}>{i > 0 && ' · '}{s}</span>)}</> : undefined}
      lead={<Art title={albumName ?? title} kind="square" src={albumId ? itemArt(albumId, 96) : undefined} seed={albumId ?? trackId} />}
    >
      {q.isError && !data ? <ErrorState title={t('app.entity.error')} onRetry={() => { void q.refetch() }} /> : (
        <>
          <Card>
            {data ? (
              <StatStrip columns={3}>
                <StatTile label={t('app.album.listens')} value={f.number(data.totals.views)} foot={t('app.entity.clicksFoot', { count: data.totals.clicks })} />
                <StatTile label={t('app.entity.skip')} value={data.totals.skipRate == null ? '—' : f.percent(data.totals.skipRate)} />
                <StatTile label={t('app.album.listeners')} value={f.number(data.totals.listeners)} />
              </StatStrip>
            ) : <div className={styles.skCard} aria-busy="true"><Skeleton height={64} radius="md" /></div>}
          </Card>
          <section aria-label={t('app.album.recent')}>
            <h3 className={styles.title}>{t('app.album.recent')}</h3>
            {!data ? <div className={styles.stack}>{[0, 1, 2].map((i) => <Skeleton key={i} height={28} />)}</div>
              : data.recent.length === 0 ? <EmptyState title={t('app.album.noRecent')} />
                : (
                  <ul className={styles.list}>
                    {data.recent.map((r, i) => (
                      <li key={`${r.user.id}-${r.startedAt}-${i}`}>
                        <UserCell user={r.user} />
                        <span className="num muted">{f.relative(new Date(r.startedAt))} · {f.duration(r.watchSeconds)}</span>
                      </li>
                    ))}
                  </ul>
                )}
          </section>
        </>
      )}
    </SidePanel>
  )
}
