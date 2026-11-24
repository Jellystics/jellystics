// Sections of the « Mes statistiques » page. They reuse the dashboard's kit and charts; nothing here
// links to a dashboard route (the page lives inside Jellyfin, for any user).
import { useMemo, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import type { ColumnDef } from '@tanstack/react-table'
import { History as HistoryIcon, Info } from 'lucide-react'
import { ChartFrame } from '@/charts/ChartFrame'
import { LineChart } from '@/charts/LineChart'
import { Scatter } from '@/charts/Scatter'
import { WeekHeatmap } from '@/charts/WeekHeatmap'
import { formatDateTime } from '@/charts/timeScale'
import { useFormat } from '@/lib/format'
import { familyOf, mediaTypeFamily, playMethodOf } from '@/lib/routes'
import { Art } from '@/ui/Art'
import { Badge, MediaTypeBadge, MethodBadge, ViewBadge } from '@/ui/Badge'
import { BarList } from '@/ui/BarList'
import { Card, CardBody, CardHeader } from '@/ui/Card'
import { DataTable } from '@/ui/DataTable'
import { EmptyState } from '@/ui/EmptyState'
import { ErrorState } from '@/ui/ErrorState'
import { Meter } from '@/ui/Meter'
import { SegBar } from '@/ui/SegBar'
import { SegmentedControl } from '@/ui/SegmentedControl'
import { Skeleton } from '@/ui/Skeleton'
import { StatStrip } from '@/ui/StatStrip'
import { StatTile } from '@/ui/StatTile'
import { Tooltip } from '@/ui/Tooltip'
import { TypeKey } from '@/ui/TypeKey'
import { FinishedTile } from '@/app/widgets/Finished'
import { pointsDelta, ratioDelta } from '@/app/pages/overview/delta'
import { dateOf, wallClock, wallDate } from '@/app/pages/history/links'
import { MeApiError, useMeDevices, useMeHabits, useMeHistory, useMeOverview, useMeScatter, useMeTop, useMeTypes } from './api'
import type { MeRange } from './period'
import { itemImageUrl, type Session } from './session'
import type { Family, MeDevices, MeHistoryRow, MeProfile, PlayedToEnd } from './types'
import l from '@/app/pages/overview/layout.module.css'
import pw from '@/app/widgets/PluginWidgets.module.css'
import s from './me.module.css'

type Props = { session: Session; range: MeRange }
type Metric = 'time' | 'views' | 'clicks'
const FAMILIES: Family[] = ['series', 'movie', 'music']
const pick = (m: Metric) => (d: { watchSeconds: number; views: number; clicks: number }) =>
  m === 'time' ? d.watchSeconds / 3600 : m === 'views' ? d.views : d.clicks

type QueryLike<T> = { data?: T; isError: boolean; error: unknown; refetch: () => unknown }

/** Loading / error shell of a card whose data is not there yet. */
function Pending<T>({ title, subtitle, query, height = 220, children }: { title: string; subtitle?: string; query: QueryLike<T>; height?: number; children: (d: T) => ReactNode }) {
  const { t } = useTranslation()
  if (query.data !== undefined) return <>{children(query.data)}</>
  const reason = query.error instanceof MeApiError ? query.error.reason : 'error'
  return (
    <Card>
      <CardHeader title={title} subtitle={subtitle} />
      <CardBody>
        {query.isError
          ? <ErrorState title={t('app.me.sectionError')} description={reason === 'error' ? undefined : t(`app.me.unavailable.${reason}`)} onRetry={() => { query.refetch() }} />
          : <div aria-busy="true"><Skeleton height={height} radius="md" /></div>}
      </CardBody>
    </Card>
  )
}

/** Foot of the finished tile when nothing was measured by the plugin: the ratio is estimated from the stop position. */
function EstimatedFoot() {
  const { t } = useTranslation()
  return (
    <Tooltip content={t('app.plugin.playedToEnd.tip')}>
      <span tabIndex={0} className={pw.tipFoot}>{t('app.me.kpi.estimated')}<Info size={12} aria-hidden="true" /></span>
    </Tooltip>
  )
}

const finishedFoot = (d: PlayedToEnd) => (d.plays > 0 && d.measured === 0 ? <EstimatedFoot /> : undefined)

/** Watch time, views, clicks, finished views (measured / estimated like the dashboard), streak. */
export function KpiStrip({ session, range, profile }: Props & { profile: MeProfile }) {
  const { t } = useTranslation()
  const f = useFormat()
  const q = useMeOverview(session, range)
  const d = q.data
  if (!d) {
    if (q.isError) return <Pending title={t('app.me.kpi.title')} query={q}>{() => null}</Pending>
    return <Skeleton height={96} radius="lg" />
  }
  const { totals, previous } = d
  const delta = (m: Metric) => (previous ? ratioDelta(pick(m)(totals), pick(m)(previous), f) : undefined)
  return (
    <Card>
      <StatStrip columns={6}>
        <StatTile label={t('app.me.kpi.time')} {...f.watchKpi(totals.watchSeconds, t('app.me.kpi.hoursUnit'))} delta={delta('time')} />
        <StatTile label={t('app.me.kpi.views')} value={f.number(totals.views)} delta={delta('views')} />
        <StatTile label={t('app.me.kpi.clicks')} value={f.number(totals.clicks)} delta={delta('clicks')} />
        <FinishedTile data={d.playedToEnd} foot={finishedFoot(d.playedToEnd)} />
        <StatTile
          label={t('app.metric.completionAvg')}
          value={totals.completion === null ? '—' : f.number(totals.completion * 100)}
          unit={totals.completion === null ? undefined : '%'}
          delta={previous ? pointsDelta(totals.completion, previous.completion, t) : undefined}
        />
        <StatTile label={t('app.me.kpi.streak')} value={f.number(profile.streak.current)} unit={t('app.me.kpi.daysUnit')} foot={t('app.me.kpi.record', { count: profile.streak.best })} />
      </StatStrip>
    </Card>
  )
}

/** Watch time / views / clicks per day (per week over long ranges). */
export function DailyCard({ session, range }: Props) {
  const { t } = useTranslation()
  const f = useFormat()
  const [metric, setMetric] = useState<Metric>('time')
  const q = useMeOverview(session, range)
  const metricLabel = t(`app.me.kpi.${metric}`)
  const week = q.data?.bucket === 'week'
  const title = t(week ? 'app.me.daily.titleWeek' : 'app.me.daily.titleDay', { metric: metricLabel })
  const control = (
    <SegmentedControl<Metric>
      aria-label={t('app.me.daily.metric')}
      value={metric}
      onValueChange={setMetric}
      options={(['time', 'views', 'clicks'] as const).map((m) => ({ value: m, label: t(`app.me.kpi.${m}`) }))}
    />
  )
  return (
    <Pending title={title} query={q} height={200}>
      {(d) => {
        const daily = d.daily ?? []
        const values = daily.map(pick(metric))
        const labels = daily.map((p) => (week ? t('app.me.daily.week', { date: f.day(dateOf(p.date)) }) : f.day(dateOf(p.date))))
        const yFormat = metric === 'time' ? f.hoursAxis : f.compact
        const empty = values.every((v) => v === 0)
        return (
          <ChartFrame
            title={title}
            csvName={`my-stats-${metric}`}
            action={control}
            table={{ columns: [t('app.me.daily.date'), metricLabel], rows: daily.map((p, i) => [p.date, Math.round(values[i] * 100) / 100]) }}
          >
            {empty ? <EmptyState title={t('app.me.daily.empty')} /> : (
              <LineChart labels={labels} series={[{ name: metricLabel, color: 'var(--accent)', values }]} yFormat={yFormat} xEvery={Math.max(1, Math.ceil(daily.length / 7))} height={200} />
            )}
          </ChartFrame>
        )
      }}
    </Pending>
  )
}

type TopFilter = 'all' | 'movie' | 'series' | 'album'

/** Most viewed movies, series and albums, with their posters. */
export function TopCard({ session, range, basePath }: Props & { basePath: string }) {
  const { t } = useTranslation()
  const f = useFormat()
  const [filter, setFilter] = useState<TopFilter>('all')
  const q = useMeTop(session, range)
  const title = t('app.me.top.title'), subtitle = t('app.me.top.subtitle')
  return (
    <Pending title={title} subtitle={subtitle} query={q} height={280}>
      {(d) => {
        const rows = (d.items ?? [])
          .filter((it) => filter === 'all' || it.kind === filter)
          .sort((a, b) => b.views - a.views || b.watchSeconds - a.watchSeconds)
          .slice(0, 10)
          .map((it) => ({
            key: `${it.kind}:${it.id}`, label: it.name, value: it.views,
            lead: <Art title={it.name} kind={it.kind === 'album' ? 'square' : 'poster'} src={itemImageUrl(basePath, it.artId || it.id, 64)} />,
            badge: filter === 'all' ? <MediaTypeBadge type={familyOf(it.kind)} /> : undefined,
            tooltip: [[t('app.me.kpi.views'), f.number(it.views)], [t('app.me.kpi.time'), f.duration(it.watchSeconds)]] as [string, string][],
          }))
        return (
          <Card>
            <CardHeader
              title={title}
              subtitle={subtitle}
              action={
                <SegmentedControl<TopFilter>
                  aria-label={t('app.me.top.filter')}
                  value={filter}
                  onValueChange={setFilter}
                  options={[
                    { value: 'all', label: t('app.me.top.all') },
                    { value: 'movie', label: t('app.me.top.movies') },
                    { value: 'series', label: t('app.me.top.series') },
                    { value: 'album', label: t('app.me.top.music') },
                  ]}
                />
              }
            />
            <CardBody>
              {rows.length === 0
                ? <EmptyState title={t('app.me.top.empty')} />
                : <BarList rows={rows} labelHeader={t('app.me.top.content')} valueLabel={t('app.me.kpi.views')} />}
            </CardBody>
          </Card>
        )
      }}
    </Pending>
  )
}

/** Watch time per media type, finished ratio per type and playback methods. */
export function TypesCard({ session, range }: Props) {
  const { t } = useTranslation()
  const f = useFormat()
  const q = useMeTypes(session, range)
  const ov = useMeOverview(session, range)
  const title = t('app.me.types.title'), subtitle = t('app.me.types.subtitle')
  return (
    <Pending title={title} subtitle={subtitle} query={q}>
      {(d) => {
        const total = FAMILIES.reduce((a, k) => a + (d.byType?.[k]?.watchSeconds ?? 0), 0)
        const views = FAMILIES.reduce((a, k) => a + (d.byType?.[k]?.views ?? 0), 0)
        const methods = ov.data?.byMethod
        const methodTotal = methods ? methods.direct + methods.stream + methods.transcode : 0
        return (
          <Card>
            <CardHeader title={title} subtitle={subtitle} />
            <CardBody>
              {total === 0 && views === 0 ? <EmptyState title={t('app.me.types.empty')} /> : (
                <div className={l.stack}>
                  <SegBar parts={FAMILIES.map((k) => ({ label: t(`app.type.${k}`), value: d.byType[k]?.watchSeconds ?? 0, color: `var(--t-${k})`, display: f.duration(d.byType[k]?.watchSeconds ?? 0) }))} />
                  <table className={s.typeTable}>
                    <thead>
                      <tr><th>{t('app.me.types.type')}</th><th className={s.n}>{t('app.me.kpi.time')}</th><th className={s.n}>{t('app.me.kpi.views')}</th><th className={s.n}>{t('app.metric.finishedViews')}</th></tr>
                    </thead>
                    <tbody>
                      {FAMILIES.map((k) => {
                        const c = d.byType[k] ?? { watchSeconds: 0, views: 0, clicks: 0 }
                        const pte = d.playedToEnd?.[k]
                        return (
                          <tr key={k}>
                            <td><TypeKey type={k} label={t(`app.type.${k}`)} /></td>
                            <td className={s.n}>{f.duration(c.watchSeconds)}</td>
                            <td className={s.n}>{f.number(c.views)}</td>
                            <td className={s.n}>{pte?.ratio == null ? '—' : f.percent(pte.ratio)}</td>
                          </tr>
                        )
                      })}
                    </tbody>
                  </table>
                  {methods && methodTotal > 0 && (
                    <>
                      <div className={l.note}>{t('app.me.types.method')}</div>
                      <SegBar parts={(['direct', 'stream', 'transcode'] as const).map((k) => ({ label: t(`app.method.${k}`), value: methods[k], color: `var(--m-${k})`, ink: `var(--m-${k}-ink)`, display: f.number(methods[k]) }))} />
                    </>
                  )}
                </div>
              )}
            </CardBody>
          </Card>
        )
      }}
    </Pending>
  )
}

/** Views by weekday × hour. */
export function HabitsCard({ session, range }: Props) {
  const { t } = useTranslation()
  const f = useFormat()
  const q = useMeHabits(session, range)
  const title = t('app.me.habits.title')
  return (
    <Pending title={title} subtitle={t('app.me.habits.subtitle')} query={q} height={180}>
      {(d) => {
        const matrix = d.weekHour?.length === 7 ? d.weekHour : Array.from({ length: 7 }, () => Array<number>(24).fill(0))
        const days = f.weekdays()
        let best = { v: 0, d: -1, h: -1 }
        matrix.forEach((row, di) => row.forEach((v, h) => { if (v > best.v) best = { v, d: di, h } }))
        const subtitle = best.d >= 0 ? t('app.me.habits.peak', { day: days[best.d], hour: f.hour(best.h) }) : t('app.me.habits.subtitle')
        return (
          <ChartFrame
            title={title}
            subtitle={subtitle}
            csvName="my-stats-week-hour"
            table={{ columns: [t('app.me.habits.day'), ...Array.from({ length: 24 }, (_, h) => f.hour(h))], rows: matrix.map((row, di) => [days[di], ...row]) }}
          >
            {best.d < 0 ? <EmptyState title={t('app.me.habits.empty')} /> : <WeekHeatmap matrix={matrix} unit={t('app.me.kpi.views')} />}
          </ChartFrame>
        )
      }}
    </Pending>
  )
}

const playTitle = (p: { item: { name: string; parentName?: string } }) => (p.item.parentName ? `${p.item.parentName} › ${p.item.name}` : p.item.name)

/** One dot per play: start time × watched minutes, coloured by media type. */
export function PlaysCard({ session, range }: Props) {
  const { t } = useTranslation()
  const f = useFormat()
  const q = useMeScatter(session, range)
  const title = t('app.me.plays.title'), subtitle = t('app.me.plays.subtitle')
  const plays = useMemo(() => [...(q.data?.points ?? [])].reverse(), [q.data]) // oldest first: newest drawn on top
  const points = useMemo(() => plays.map((p) => {
    const ms = Date.parse(p.startedAt)
    return {
      x: ms, y: Math.max(0, p.watchSeconds) / 60, label: playTitle(p), color: `var(--t-${p.type})`,
      rows: [
        { label: t('app.me.plays.date'), value: formatDateTime(ms, f.locale) },
        { label: t('app.me.plays.duration'), value: f.duration(Math.max(0, p.watchSeconds)) },
        { color: `var(--t-${p.type})`, label: t('app.me.types.type'), value: t(`app.type.${p.type}`) },
      ],
    }
  }), [plays, f, t])
  return (
    <Pending title={title} subtitle={subtitle} query={q}>
      {(d) => {
        const present = FAMILIES.filter((k) => plays.some((p) => p.type === k))
        return (
          <ChartFrame
            title={title}
            subtitle={subtitle}
            csvName="my-stats-plays"
            legend={present.map((k) => ({ label: t(`app.type.${k}`), color: `var(--t-${k})` }))}
            table={{
              columns: [t('app.me.plays.date'), t('app.me.plays.content'), t('app.me.types.type'), t('app.me.plays.minutes')],
              rows: plays.map((p) => [p.startedAt, playTitle(p), t(`app.type.${p.type}`), Math.round((Math.max(0, p.watchSeconds) / 60) * 10) / 10]),
            }}
          >
            {plays.length === 0 ? <EmptyState title={t('app.me.plays.empty')} /> : (
              <>
                <Scatter points={points} timeAxis radius={4} xLabel={t('app.me.plays.x')} yLabel={t('app.me.plays.y')} yFormat={f.axis} />
                {d.capped && <div className={l.note}>{t('app.me.plays.capped', { value: f.number(d.limit) })}</div>}
              </>
            )}
          </ChartFrame>
        )
      }}
    </Pending>
  )
}

type Device = MeDevices['devices'][number]
type Client = MeDevices['clients'][number]

/** Devices and apps used over the period (never an IP address). */
export function DevicesCards({ session, range }: Props) {
  const { t } = useTranslation()
  const f = useFormat()
  const q = useMeDevices(session, range)
  const deviceCols: ColumnDef<Device>[] = [
    { id: 'device', header: t('app.me.devices.device'), cell: ({ row: { original: d } }) => d.device || '—' },
    { id: 'client', header: t('app.me.devices.client'), cell: ({ row: { original: d } }) => <span className="ink2">{[d.client, d.version].filter(Boolean).join(' ') || '—'}</span> },
    { id: 'views', header: t('app.me.kpi.views'), meta: { numeric: true }, cell: ({ row: { original: d } }) => f.number(d.views) },
    { id: 'time', header: t('app.me.kpi.time'), meta: { numeric: true }, cell: ({ row: { original: d } }) => <span className={s.nowrap}>{f.duration(d.watchSeconds)}</span> },
    { id: 'lastUsed', header: t('app.me.devices.lastUsed'), cell: ({ row: { original: d } }) => <span className={`muted ${s.nowrap}`}>{d.lastUsed ? f.relative(new Date(d.lastUsed)) : '—'}</span> },
  ]
  const ratio = (r: number | null) => (r == null ? '—' : f.percent(r))
  const clientCols: ColumnDef<Client>[] = [
    { id: 'client', header: t('app.me.devices.client'), cell: ({ row: { original: c } }) => c.client || '—' },
    { id: 'clicks', header: t('app.me.kpi.clicks'), meta: { numeric: true }, cell: ({ row: { original: c } }) => f.number(c.clicks) },
    { id: 'views', header: t('app.me.kpi.views'), meta: { numeric: true }, cell: ({ row: { original: c } }) => f.number(c.views) },
    { id: 'transcodes', header: t('app.me.clients.transcodes'), meta: { numeric: true }, cell: ({ row: { original: c } }) => ratio(c.transcodeRatio) },
    { id: 'falseStarts', header: t('app.me.clients.falseStarts'), meta: { numeric: true }, cell: ({ row: { original: c } }) => ratio(c.falseStartRatio) },
  ]
  const devTitle = t('app.me.devices.title'), cliTitle = t('app.me.clients.title')
  return (
    <div className={l.g2}>
      <Pending title={devTitle} subtitle={t('app.me.devices.subtitle')} query={q} height={160}>
        {(d) => (
          <Card>
            <CardHeader title={devTitle} subtitle={t('app.me.devices.subtitle')} />
            <CardBody flush><DataTable dense columns={deviceCols} data={d.devices ?? []} empty={<EmptyState title={t('app.me.devices.empty')} />} /></CardBody>
          </Card>
        )}
      </Pending>
      <Pending title={cliTitle} subtitle={t('app.me.clients.subtitle')} query={q} height={160}>
        {(d) => (
          <Card>
            <CardHeader title={cliTitle} subtitle={t('app.me.clients.subtitle')} />
            <CardBody flush><DataTable dense columns={clientCols} data={d.clients ?? []} empty={<EmptyState title={t('app.me.devices.empty')} />} /></CardBody>
          </Card>
        )}
      </Pending>
    </div>
  )
}

function historySubtitle(r: MeHistoryRow, t: (k: string, o?: Record<string, unknown>) => string): string {
  const it = r.item
  if (it.kind === 'episode') {
    const ep = it.season != null && it.episode != null ? t('app.history.episodeShort', { season: it.season, episode: it.episode }) : ''
    return [it.parentName, ep].filter(Boolean).join(' · ')
  }
  if (it.kind === 'track') return it.parentName ?? ''
  return t('app.type.movie')
}

/** Paginated history of the user's plays, newest first. */
export function HistoryCard({ session, range, basePath }: Props & { basePath: string }) {
  const { t } = useTranslation()
  const f = useFormat()
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(25)
  const q = useMeHistory(session, range, page, pageSize)
  const data = q.data && Array.isArray(q.data.rows) ? q.data : undefined
  const columns: ColumnDef<MeHistoryRow>[] = [
    {
      id: 'startedAt', header: t('app.history.col.date'),
      cell: ({ row: { original: r } }) => <span className={`num ${s.nowrap} muted`}>{f.day(dateOf(wallDate(r.startedAt)))} · {wallClock(r.startedAt)}</span>,
    },
    {
      id: 'content', header: t('app.history.col.content'),
      cell: ({ row: { original: r } }) => {
        const fam = mediaTypeFamily(r.mediaType)
        const art = r.item.kind === 'movie' ? r.item.id : r.item.parentId || r.item.id
        return (
          <div className={l.cellMedia}>
            <Art title={r.item.parentName || r.item.name} kind={fam === 'music' ? 'square' : 'poster'} src={itemImageUrl(basePath, art, 64)} />
            <div className={l.ellipsis}>
              <div>{r.item.name}</div>
              <div className={l.cellSub}><TypeKey type={fam} label={historySubtitle(r, t) || t(`app.type.${fam}`)} /></div>
            </div>
          </div>
        )
      },
    },
    { id: 'watchSeconds', header: t('app.history.col.watched'), meta: { numeric: true }, cell: ({ row: { original: r } }) => <span className={s.nowrap}>{f.duration(r.watchSeconds)}</span> },
    {
      id: 'completion', header: t('app.history.col.completion'),
      cell: ({ row: { original: r } }) => r.completion === null ? <span className="muted">—</span> : (
        <div className={s.completion}><Meter value={r.completion} thin label={t('app.history.col.completion')} /><span className="num muted">{f.percent(r.completion)}</span></div>
      ),
    },
    {
      id: 'device', header: t('app.history.col.device'),
      cell: ({ row: { original: r } }) => <div><div>{r.client || '—'}</div><div className={l.cellSub}>{r.device}</div></div>,
    },
    {
      id: 'method', header: t('app.history.col.method'),
      cell: ({ row: { original: r } }) => { const m = playMethodOf(r.method); return m ? <MethodBadge method={m} /> : <Badge>{r.method || '—'}</Badge> },
    },
    { id: 'view', header: () => <span className="sr-only">{t('app.history.col.view')}</span>, cell: ({ row: { original: r } }) => <ViewBadge view={r.isView} /> },
  ]
  const title = t('app.me.history.title')
  if (q.isError && !data) return <Pending title={title} query={q}>{() => null}</Pending>
  return (
    <Card>
      <CardHeader title={title} subtitle={data ? t('app.me.history.subtitle', { count: data.total }) : undefined} />
      <DataTable
        columns={columns}
        data={data?.rows ?? []}
        loading={!data}
        pagination={data ? {
          page: data.page, pageSize: data.pageSize, total: data.total, onPageChange: setPage,
          onPageSizeChange: (n) => { setPageSize(n); setPage(1) },
        } : undefined}
        empty={<EmptyState icon={<HistoryIcon size={20} strokeWidth={1.6} />} title={t('app.me.history.empty')} />}
      />
    </Card>
  )
}
