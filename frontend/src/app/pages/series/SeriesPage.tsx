import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { AlertTriangle } from 'lucide-react'
import type { ColumnDef } from '@tanstack/react-table'
import { AppLink, Art, AvatarStack, Badge, Card, DataTable, EmptyState, EntityHeader, ErrorState, SegmentedControl, StatStrip, StatTile, Tabs, Tooltip, withPeriod } from '@/ui'
import { ChartFrame, Columns, RetentionChart } from '@/charts'
import { useFormat } from '@/lib/format'
import { useSearchParam } from '@/lib/useUrlState'
import { useSeries, type SeriesDetail, type SeriesEpisode } from '@/api'
import { useSeriesHistory } from '@/api/entityExtras'
import { CompletionCell, EntitySkeleton, HistoryTable, LibraryKicker, QueryView, RatioCell, UserCell } from '../library/EntityParts'
import { SkipsCard } from '@/app/widgets/PluginWidgets'
import { entityPath } from '@/lib/routes'
import { formatWhen, itemArt, share, useEntityCrumbs } from '../library/entity'
import { CADENCE, RELEASE_LAG, fixedBuckets, seasonViews } from './series'
import styles from '../library/entity.module.css'

type Viewer = SeriesDetail['viewers'][number]
const PAGE_SIZE = 25

function SeriesHistoryTab({ id }: { id: string }) {
  const [page, setPage] = useSearchParam('page', '1')
  const p = Math.max(1, Number(page) || 1)
  const q = useSeriesHistory(id, { page: p, pageSize: PAGE_SIZE })
  const { t } = useTranslation()
  if (q.isError) return <ErrorState title={t('app.entity.error')} onRetry={() => { void q.refetch() }} />
  return (
    <HistoryTable
      rows={q.data?.rows ?? []} loading={q.isPending}
      pagination={q.data ? { page: p, pageSize: PAGE_SIZE, total: q.data.total, onPageChange: (n) => setPage(String(n)) } : undefined}
    />
  )
}

function Series({ id, data }: { id: string; data: SeriesDetail }) {
  const { t } = useTranslation()
  const f = useFormat()
  const navigate = useNavigate()
  const [sp] = useSearchParams()
  const [tab, setTab] = useSearchParam('tab', 'episodes')
  const seasons = seasonViews(data)
  const charted = seasons.filter((s) => !s.special && s.retention.length > 0)
  const [seasonParam, setSeason] = useSearchParam('season')
  const season = seasons.find((s) => String(s.number) === seasonParam) ?? seasons.find((s) => !s.special) ?? seasons[0]
  const { item, totals } = data
  const seasonLabel = (n: number) => (n === 0 ? t('app.series.specials') : t('app.series.season', { n }))
  const years = item.years.from ? (item.years.to && item.years.to !== item.years.from ? `${item.years.from} – ${item.years.to}` : String(item.years.from)) : null
  const clicksNoView = share(totals.clicks - totals.views, totals.clicks)

  const episodeCols: ColumnDef<SeriesEpisode>[] = [
    { id: 'n', header: t('app.series.col.number'), meta: { numeric: true }, cell: ({ row }) => <span className="muted">{row.original.number ?? '—'}</span> },
    {
      id: 'title', header: t('app.series.col.title'),
      cell: ({ row }) => (
        <span className={styles.cellMedia}>
          <Art title={row.original.name} kind="thumb" src={itemArt(row.original.id, 160)} seed={row.original.id} />
          <AppLink to={entityPath('episode', row.original.id, id)} className={styles.link}>{row.original.name}</AppLink>
        </span>
      ),
    },
    { id: 'views', header: t('app.entity.col.views'), meta: { numeric: true }, cell: ({ row }) => f.number(row.original.views) },
    { id: 'viewers', header: t('app.entity.col.viewers'), meta: { numeric: true }, cell: ({ row }) => f.number(row.original.viewers) },
    { id: 'completion', header: t('app.metric.completionAvg'), cell: ({ row }) => <CompletionCell value={row.original.avgCompletion} /> },
    {
      id: 'abandons', header: t('app.series.col.abandons'), meta: { numeric: true },
      cell: ({ row }) => row.original.abandons > 0
        ? <Badge tone="serious" icon={<AlertTriangle size={12} strokeWidth={1.6} />}>{f.number(row.original.abandons)}</Badge>
        : <span className="muted">—</span>,
    },
    ...((data.skips?.plays ?? 0) > 0 ? [{
      id: 'intro', header: t('app.plugin.skips.intro'),
      cell: ({ row }) => row.original.skipPlays > 0
        ? (
          <Tooltip content={t('app.plugin.skips.cellTip', { count: row.original.skipPlays, avg: row.original.avgSkippedSeconds == null ? '—' : f.duration(row.original.avgSkippedSeconds) })}>
            <span tabIndex={0} className={styles.tipTrigger}>
              <RatioCell value={row.original.introRate} width={64} tone="muted" label={t('app.plugin.skips.intro')} />
            </span>
          </Tooltip>
        )
        : <span className="muted">—</span>,
    } satisfies ColumnDef<SeriesEpisode>] : []),
  ]
  const viewerCols: ColumnDef<Viewer>[] = [
    { id: 'user', header: t('app.entity.col.user'), cell: ({ row }) => <UserCell user={row.original.user} /> },
    { id: 'episodes', header: t('app.series.col.episodesWatched'), meta: { numeric: true }, cell: ({ row }) => f.number(row.original.episodesWatched) },
    { id: 'last', header: t('app.entity.col.lastWatched'), cell: ({ row }) => <span className="num muted">{formatWhen(row.original.lastWatched, f.locale)}</span> },
  ]

  const openEpisode = (s: number, e: number) => {
    const ep = charted[s - 1]?.episodes[e - 1]
    if (ep) navigate(withPeriod(entityPath('episode', ep.id, id), sp))
  }
  const cadence = fixedBuckets(CADENCE, data.cadence)
  const lag = fixedBuckets(RELEASE_LAG, data.releaseLag)
  const cadenceLabels = CADENCE.map((b) => t(`app.series.cadence.b.${b}`))
  const lagLabels = RELEASE_LAG.map((b) => t(`app.series.lag.b.${b}`))
  const retentionTable = charted.flatMap((s) => s.episodes.map((e, i) => [t('app.entity.episodeCode', { season: s.number, episode: e.number ?? i + 1 }), e.name, s.retention[i] ?? 0]))

  return (
    <div className={styles.stack}>
      <EntityHeader
        art={<Art title={item.name} kind="poster" size="lg" src={itemArt(item.id)} seed={item.id} />}
        kicker={<LibraryKicker type="series" library={item.library} />}
        title={item.name}
        meta={[
          years,
          t('app.series.counts', { seasons: item.seasons, episodes: item.episodes }),
          item.genres.length ? item.genres.slice(0, 3).join(', ') : null,
          data.viewers.length ? <span className={styles.row}><AvatarStack users={data.viewers.map((v) => v.user)} max={5} />{t('app.series.viewersCount', { count: data.viewers.length })}</span> : null,
        ].filter(Boolean)}
      />
      <Card>
        <StatStrip columns={5}>
          <StatTile label={t('app.entity.watchTime')} value={f.duration(totals.watchSeconds)} />
          <StatTile label={t('app.entity.views')} value={f.number(totals.views)} foot={t('app.entity.viewersFoot', { count: totals.viewers })} />
          <StatTile label={t('app.entity.clicks')} value={f.number(totals.clicks)} foot={clicksNoView != null ? t('app.series.clicksNoView', { pct: f.percent(clicksNoView) }) : undefined} />
          <StatTile label={t('app.series.retention')} value={totals.retention == null ? '—' : f.percent(totals.retention)} foot={t('app.series.retentionFoot')} />
          <StatTile label={t('app.series.binges')} value={f.number(totals.binges)} foot={t('app.series.bingesFoot')} />
        </StatStrip>
      </Card>
      <ChartFrame
        title={t('app.series.retentionTitle')} subtitle={t('app.series.retentionSub')} csvName={`series-${id}-retention`}
        table={{ columns: [t('app.series.col.episode'), t('app.series.col.title'), t('app.entity.col.viewers')], rows: retentionTable }}
      >
        {charted.length ? <RetentionChart seasons={charted.map((s) => s.retention)} seasonNumbers={charted.map((s) => s.number)} onEpisodeClick={openEpisode} /> : <EmptyState title={t('app.series.noEpisodes')} />}
      </ChartFrame>
      <SkipsCard skips={data.skips} since={data.pluginSince} />
      <Tabs value={tab ?? 'episodes'} onValueChange={setTab} items={[
        { value: 'episodes', label: t('app.series.tab.episodes'), count: item.episodes },
        { value: 'viewers', label: t('app.entity.tab.viewers'), count: data.viewers.length },
        { value: 'history', label: t('app.entity.tab.history') },
      ]} />
      {tab === 'viewers' && <Card><DataTable columns={viewerCols} data={data.viewers} empty={<EmptyState title={t('app.entity.noViewers')} />} /></Card>}
      {tab === 'history' && <Card><SeriesHistoryTab id={id} /></Card>}
      {tab !== 'viewers' && tab !== 'history' && (
        <>
          {seasons.length > 1 && season && (
            <div className={styles.row}>
              <SegmentedControl
                aria-label={t('app.series.seasonPicker')} value={String(season.number)} onValueChange={setSeason}
                options={seasons.map((s) => ({ value: String(s.number), label: seasonLabel(s.number) }))}
              />
            </div>
          )}
          <Card>
            <DataTable
              columns={episodeCols} data={season?.episodes ?? []}
              empty={<EmptyState title={t('app.series.noEpisodes')} />}
            />
          </Card>
        </>
      )}
      <div className={styles.g2}>
        <ChartFrame
          title={t('app.series.cadence.title')} subtitle={t('app.series.cadence.subtitle')} csvName={`series-${id}-cadence`}
          table={{ columns: [t('app.series.cadence.gap'), t('app.series.cadence.count')], rows: cadence.map((v, i) => [cadenceLabels[i], v]) }}
        >
          {cadence.some((v) => v > 0)
            ? <Columns labels={cadenceLabels} values={cadence} name={t('app.series.cadence.count')} emphasis={cadence.indexOf(Math.max(...cadence))} height={170} />
            : <EmptyState title={t('app.series.cadence.empty')} />}
        </ChartFrame>
        <ChartFrame
          title={t('app.series.lag.title')} subtitle={t('app.series.lag.subtitle')} csvName={`series-${id}-release-lag`}
          table={{ columns: [t('app.series.lag.delay'), t('app.series.lag.count')], rows: lag.map((v, i) => [lagLabels[i], v]) }}
        >
          {lag.some((v) => v > 0)
            ? <Columns labels={lagLabels} values={lag} name={t('app.series.lag.count')} emphasis={lag.indexOf(Math.max(...lag))} height={170} />
            : <EmptyState title={t('app.series.lag.empty')} />}
        </ChartFrame>
      </div>
    </div>
  )
}

export default function SeriesPage() {
  const { id = '' } = useParams()
  const q = useSeries(id)
  useEntityCrumbs(q.data?.item.name, q.data?.item.library)
  return <QueryView query={q} skeleton={<EntitySkeleton tiles={5} />}>{(data) => <Series id={id} data={data} />}</QueryView>
}
