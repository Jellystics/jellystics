import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import type { ColumnDef } from '@tanstack/react-table'
import { AppLink, Art, BarList, Card, CardBody, CardHeader, DataTable, EmptyState, EntityHeader, StatStrip, StatTile } from '@/ui'
import { useFormat } from '@/lib/format'
import { useAlbum, type AlbumDetail } from '@/api'
import { useAlbumFullListens } from '@/api/entityExtras'
import { EntitySkeleton, LibraryKicker, QueryView, SkipCell, SubQuery } from '../library/EntityParts'
import { PANEL_STATE, entityPath, itemArt, useEntityCrumbs } from '../library/entity'
import { TrackPanel } from './TrackPanel'
import styles from '../library/entity.module.css'

type Track = AlbumDetail['tracks'][number]

function FullAlbums({ id }: { id: string }) {
  const { t } = useTranslation()
  const q = useAlbumFullListens(id)
  const rows = q.data ?? []
  return (
    <Card>
      <CardHeader title={t('app.album.full.title')} subtitle={t('app.album.full.subtitle')} />
      <CardBody>
        <SubQuery query={q} height={120} empty={rows.length === 0 ? <EmptyState title={t('app.album.full.empty')} /> : null}>
          <BarList
            labelHeader={t('app.album.full.album')} valueLabel={t('app.album.full.count')}
            rows={rows.map((a) => ({
              key: a.id, label: a.name, value: a.fullListens, to: a.id === id ? undefined : entityPath('album', a.id),
              lead: <Art title={a.name} kind="square" src={itemArt(a.id, 72)} seed={a.id} />,
            }))}
          />
        </SubQuery>
      </CardBody>
    </Card>
  )
}

function Album({ id, data }: { id: string; data: AlbumDetail }) {
  const { t } = useTranslation()
  const f = useFormat()
  const { item, totals } = data
  const trackPath = (tid: string) => `${entityPath('album', id)}?track=${encodeURIComponent(tid)}`
  const cols: ColumnDef<Track>[] = [
    {
      id: 'title', header: t('app.album.col.title'),
      cell: ({ row }) => <span className={styles.cellMedia}><span className={`num muted ${styles.trackNo}`}>{row.original.number ?? row.index + 1}</span><span className={styles.title}>{row.original.name}</span></span>,
    },
    { id: 'duration', header: t('app.album.col.duration'), meta: { numeric: true }, cell: ({ row }) => <span className="muted">{f.position(row.original.runtimeSeconds)}</span> },
    { id: 'listens', header: t('app.album.listens'), meta: { numeric: true }, cell: ({ row }) => f.number(row.original.views) },
    { id: 'skip', header: t('app.entity.skip'), cell: ({ row }) => <SkipCell rate={row.original.skipRate} /> },
    { id: 'listeners', header: t('app.album.listeners'), meta: { numeric: true }, cell: ({ row }) => f.number(row.original.listeners) },
  ]
  const names = Object.fromEntries(data.tracks.map((tr) => [tr.id, { name: tr.name, albumId: item.id, albumName: item.name, artistId: item.artistId, artistName: item.artistName }]))
  return (
    <div className={styles.stack}>
      <EntityHeader
        art={<Art title={item.name} kind="square" size="lg" src={itemArt(item.id)} seed={item.id} />}
        kicker={
          <LibraryKicker type="music" library={item.library}>
            {item.artistId && item.artistName ? <AppLink to={entityPath('artist', item.artistId)} className={styles.subLink}>{item.artistName}</AppLink> : item.artistName}
          </LibraryKicker>
        }
        title={item.name}
        meta={[item.year ? String(item.year) : null, t('app.album.tracksMeta', { count: item.tracks, duration: f.duration(item.runtimeSeconds) })].filter(Boolean)}
      />
      <Card>
        <StatStrip columns={4}>
          <StatTile label={t('app.album.listens')} value={f.number(totals.listens)} />
          <StatTile label={t('app.album.fullListens')} value={totals.fullListenRatio == null ? '—' : f.percent(totals.fullListenRatio)} foot={t('app.album.wholeAlbum', { count: totals.fullAlbumListens })} />
          <StatTile label={t('app.album.listeners')} value={f.number(totals.listeners)} />
          <StatTile label={t('app.entity.skipRate')} value={totals.skipRate == null ? '—' : f.percent(totals.skipRate)} />
        </StatStrip>
      </Card>
      <Card>
        <CardHeader title={t('app.album.tracks')} subtitle={t('app.album.tracksSub')} />
        <DataTable dense columns={cols} data={data.tracks} rowHref={(tr) => trackPath(tr.id)} rowState={PANEL_STATE} empty={<EmptyState title={t('app.album.noTracks')} />} />
      </Card>
      <FullAlbums id={id} />
      <TrackPanel names={names} />
    </div>
  )
}

export default function AlbumPage() {
  const { id = '' } = useParams()
  const q = useAlbum(id)
  useEntityCrumbs(q.data?.item.name, q.data?.item.library)
  return <QueryView query={q} skeleton={<EntitySkeleton />}>{(data) => <Album id={id} data={data} />}</QueryView>
}
