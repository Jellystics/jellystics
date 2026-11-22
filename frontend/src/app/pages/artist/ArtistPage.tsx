import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import type { ColumnDef } from '@tanstack/react-table'
import { AppLink, Art, Card, CardHeader, DataTable, EmptyState, EntityHeader, StatStrip, StatTile } from '@/ui'
import { ChartFrame, Columns } from '@/charts'
import { useFormat } from '@/lib/format'
import { useArtist, type ArtistDetail, type TrackStat } from '@/api'
import { EntitySkeleton, LibraryKicker, QueryView, SkipCell } from '../library/EntityParts'
import { entityPath } from '@/lib/routes'
import { PANEL_STATE, itemArt, useEntityCrumbs } from '../library/entity'
import { TrackPanel } from '../album/TrackPanel'
import styles from '../library/entity.module.css'

function Artist({ id, data }: { id: string; data: ArtistDetail }) {
  const { t } = useTranslation()
  const f = useFormat()
  const { item, totals } = data
  const trackPath = (tid: string) => `${entityPath('artist', id)}?track=${encodeURIComponent(tid)}`
  const cols: ColumnDef<TrackStat>[] = [
    {
      id: 'title', header: t('app.album.col.title'),
      cell: ({ row }) => <span className={styles.cellMedia}><span className={`num muted ${styles.trackNo}`}>{row.index + 1}</span><span className={styles.title}>{row.original.name}</span></span>,
    },
    { id: 'album', header: t('app.artist.col.album'), cell: ({ row }) => row.original.albumId ? <AppLink to={entityPath('album', row.original.albumId)} className={styles.subLink}>{row.original.albumName}</AppLink> : <span className="muted">{row.original.albumName || '—'}</span> },
    { id: 'listens', header: t('app.album.listens'), meta: { numeric: true }, cell: ({ row }) => f.number(row.original.views) },
    { id: 'clicks', header: t('app.entity.clicks'), meta: { numeric: true }, cell: ({ row }) => <span className="muted">{f.number(row.original.clicks)}</span> },
    { id: 'skip', header: t('app.entity.skip'), cell: ({ row }) => <SkipCell rate={row.original.skipRate} /> },
    { id: 'listeners', header: t('app.album.listeners'), meta: { numeric: true }, cell: ({ row }) => f.number(row.original.listeners) },
  ]
  const hours = data.skipByHour.length === 24 ? data.skipByHour : Array.from({ length: 24 }, (_, h) => data.skipByHour[h] ?? 0)
  const hourLabels = hours.map((_, h) => f.hour(h))
  const peak = hours.indexOf(Math.max(...hours))
  const names = Object.fromEntries(data.topTracks.map((tr) => [tr.id, { name: tr.name, albumId: tr.albumId, albumName: tr.albumName, artistId: item.id, artistName: item.name }]))
  return (
    <div className={styles.stack}>
      <EntityHeader
        art={<Art title={item.name} kind="square" size="lg" src={itemArt(item.id)} seed={item.id} />}
        kicker={<LibraryKicker type="music" library={item.library} />}
        title={item.name}
        meta={[t('app.artist.counts', { albums: item.albums, tracks: item.tracks }), item.genres.length ? item.genres.slice(0, 3).join(', ') : null].filter(Boolean)}
      />
      <Card>
        <StatStrip columns={4}>
          <StatTile label={t('app.album.listens')} value={f.number(totals.listens)} />
          <StatTile label={t('app.artist.listenTime')} value={f.duration(totals.watchSeconds)} />
          <StatTile label={t('app.album.listeners')} value={f.number(totals.listeners)} />
          <StatTile label={t('app.entity.skipRate')} value={totals.skipRate == null ? '—' : f.percent(totals.skipRate)} foot={t('app.artist.skipFoot')} />
        </StatStrip>
      </Card>
      <Card>
        <CardHeader title={t('app.artist.topTracks')} subtitle={t('app.artist.topTracksSub')} />
        <DataTable columns={cols} data={data.topTracks} rowHref={(tr) => trackPath(tr.id)} rowState={PANEL_STATE} empty={<EmptyState title={t('app.artist.noTracks')} />} />
      </Card>
      <section aria-label={t('app.artist.albums')} className={styles.stack}>
        <h2 className="sr-only">{t('app.artist.albums')}</h2>
        {data.albums.length === 0 ? <Card><EmptyState title={t('app.artist.noAlbums')} /></Card> : (
          <div className={styles.cards}>
            {data.albums.map((a) => (
              <AppLink key={a.id} to={entityPath('album', a.id)} className={styles.tile}>
                <Art title={a.name} kind="square" size="md" src={itemArt(a.id, 160)} seed={a.id} />
                <span className={styles.main}>
                  <span className={styles.tileName}>{a.name}</span>
                  <span className={styles.tileMeta}>{[a.year, t('app.artist.albumMeta', { count: a.tracks, listens: f.number(a.listens) })].filter(Boolean).join(' · ')}</span>
                </span>
              </AppLink>
            ))}
          </div>
        )}
      </section>
      <ChartFrame
        title={t('app.artist.skipByHour')} subtitle={t('app.artist.skipByHourSub')} csvName={`artist-${id}-skip-by-hour`}
        table={{ columns: [t('app.artist.hour'), t('app.entity.skipRate')], rows: hours.map((v, h) => [hourLabels[h], f.percent(v)]) }}
      >
        {hours.some((v) => v > 0)
          ? <Columns labels={hourLabels} values={hours} name={t('app.entity.skipRate')} emphasis={peak} xEvery={4} height={160} yFormat={f.percent} />
          : <EmptyState title={t('app.artist.noSkips')} />}
      </ChartFrame>
      <TrackPanel names={names} />
    </div>
  )
}

export default function ArtistPage() {
  const { id = '' } = useParams()
  const q = useArtist(id)
  useEntityCrumbs(q.data?.item.name, q.data?.item.library)
  return <QueryView query={q} skeleton={<EntitySkeleton />}>{(data) => <Artist id={id} data={data} />}</QueryView>
}
