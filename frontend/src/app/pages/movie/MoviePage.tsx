import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import type { ColumnDef } from '@tanstack/react-table'
import { Art, Card, DataTable, EmptyState, EntityHeader, StatStrip, StatTile, Tabs } from '@/ui'
import { ChartFrame, Columns } from '@/charts'
import { useFormat } from '@/lib/format'
import { useSearchParam } from '@/lib/useUrlState'
import { useMovie, type MovieDetail } from '@/api'
import { useMovieSittings } from '@/api/entityExtras'
import { useItemMarks } from '@/api/pluginData'
import { FinishedTile, MarksCard } from '@/app/widgets/PluginWidgets'
import { hasMarks } from '@/app/widgets/plugin'
import { CompletionCell, EntitySkeleton, HistoryTable, LibraryKicker, QueryView, SubQuery, UserCell } from '../library/EntityParts'
import { FunnelCard, StopsCard } from '../library/charts'
import { formatWhen, itemArt, useEntityCrumbs } from '../library/entity'
import styles from '../library/entity.module.css'

type Viewer = MovieDetail['viewers'][number]

function Sittings({ id }: { id: string }) {
  const { t } = useTranslation()
  const f = useFormat()
  const q = useMovieSittings(id)
  const buckets = q.data ?? []
  const total = buckets.reduce((a, b) => a + b.count, 0)
  const labels = buckets.map((b) => t(`app.movie.sittings.n${b.sittings}`))
  const shares = buckets.map((b) => (total ? b.count / total : 0))
  return (
    <ChartFrame
      title={t('app.movie.sittings.title')} subtitle={t('app.movie.sittings.subtitle')} csvName={`movie-${id}-sittings`}
      table={{ columns: [t('app.movie.sittings.col'), t('app.movie.sittings.count'), t('app.movie.sittings.share')], rows: buckets.map((b, i) => [labels[i], b.count, f.percent(shares[i])]) }}
    >
      <SubQuery query={q} height={160} empty={total === 0 ? <EmptyState title={t('app.movie.sittings.empty')} /> : null}>
        <Columns labels={labels} values={shares} name={t('app.movie.sittings.share')} emphasis={0} height={160} yFormat={f.percent} />
      </SubQuery>
    </ChartFrame>
  )
}

function Movie({ id, data }: { id: string; data: MovieDetail }) {
  const { t } = useTranslation()
  const f = useFormat()
  const [tab, setTab] = useSearchParam('tab', 'viewers')
  const { item, totals, funnel } = data
  const marks = useItemMarks(id).data
  const showMarks = hasMarks(marks)
  const meta = [
    item.year ? String(item.year) : null,
    item.runtimeSeconds ? f.duration(item.runtimeSeconds) : null,
    item.genres.length ? item.genres.slice(0, 3).join(', ') : null,
    [item.resolution, item.sizeBytes ? f.bytes(item.sizeBytes) : null].filter(Boolean).join(' · ') || null,
  ].filter((m): m is string => !!m)
  const viewerCols: ColumnDef<Viewer>[] = [
    { id: 'user', header: t('app.entity.col.user'), cell: ({ row }) => <UserCell user={row.original.user} /> },
    { id: 'views', header: t('app.entity.col.views'), meta: { numeric: true }, cell: ({ row }) => f.number(row.original.views) },
    { id: 'last', header: t('app.entity.col.lastWatched'), cell: ({ row }) => <span className="num muted">{formatWhen(row.original.lastWatched, f.locale)}</span> },
    { id: 'best', header: t('app.movie.bestCompletion'), cell: ({ row }) => <CompletionCell value={row.original.bestCompletion} /> },
  ]
  return (
    <div className={styles.stack}>
      <EntityHeader
        art={<Art title={item.name} kind="poster" size="lg" src={itemArt(item.id)} seed={item.id} />}
        kicker={<LibraryKicker type="movie" library={item.library} />}
        title={item.name}
        meta={meta}
      />
      <Card>
        <StatStrip columns={4}>
          <StatTile label={t('app.entity.views')} value={f.number(totals.views)} foot={t('app.entity.clicksFoot', { count: totals.clicks })} />
          <StatTile label={t('app.entity.viewers')} value={f.number(totals.viewers)} foot={f.duration(totals.watchSeconds)} />
          <FinishedTile data={data.playedToEnd} foot={t('app.movie.completedFoot', { done: f.number(funnel.completed), count: funnel.views })} />
          <StatTile label={t('app.movie.transcoded')} value={totals.transcodeRatio == null ? '—' : f.percent(totals.transcodeRatio)} />
        </StatStrip>
      </Card>
      <div className={styles.g2}>
        <FunnelCard funnel={funnel} title={t('app.movie.funnel')} subtitle={t('app.entity.funnelSub')}csvName={`movie-${id}-funnel`} />
        <StopsCard bins={data.stopPositions} runtimeSeconds={item.runtimeSeconds} title={t('app.movie.stops')} subtitle={t('app.movie.stopsSub')} csvName={`movie-${id}-stops`} />
      </div>
      {showMarks
        ? <div className={styles.g2}><Sittings id={id} /><MarksCard data={marks} /></div>
        : <Sittings id={id} />}
      <Tabs value={tab ?? 'viewers'} onValueChange={setTab} items={[
        { value: 'viewers', label: t('app.entity.tab.viewers'), count: data.viewers.length },
        { value: 'history', label: t('app.entity.tab.history'), count: data.history.length },
      ]} />
      <Card>
        {tab === 'history'
          ? <HistoryTable rows={data.history} showItem={false} />
          : <DataTable columns={viewerCols} data={data.viewers} empty={<EmptyState title={t('app.entity.noViewers')} />} />}
      </Card>
    </div>
  )
}

export default function MoviePage() {
  const { id = '' } = useParams()
  const q = useMovie(id)
  useEntityCrumbs(q.data?.item.name, q.data?.item.library)
  return <QueryView query={q} skeleton={<EntitySkeleton />}>{(data) => <Movie id={id} data={data} />}</QueryView>
}
