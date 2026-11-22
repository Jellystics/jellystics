import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { CalendarClock } from 'lucide-react'
import { useOverview, type GroupKind, type Overview } from '@/api'
import { itemImageUrl } from '@/api/live'
import { ChartFrame, Columns, LineChart, WeekHeatmap } from '@/charts'
import { useFormat } from '@/lib/format'
import { useSearchParam } from '@/lib/useUrlState'
import {
  AppLink, Art, Avatar, BarList, Card, CardBody, CardHeader, EmptyState, ErrorState, MediaTypeBadge, PageHeader,
  SegBar, SegmentedControl, Skeleton, StatStrip, StatTile,
} from '@/ui'
import { dateOf, groupFamily, groupHref, userHref } from '../history/links'
import { pointsDelta, ratioDelta } from './delta'
import { LiveCard } from './LiveCard'
import l from './layout.module.css'

type Metric = 'time' | 'views' | 'clicks'
const METRICS: Metric[] = ['time', 'views', 'clicks']
type TopFilter = 'all' | GroupKind

const pick = (m: Metric) => (d: { watchSeconds: number; views: number; clicks: number }) =>
  m === 'time' ? d.watchSeconds / 3600 : m === 'views' ? d.views : d.clicks

function OverviewSkeleton() {
  return (
    <>
      <Card><div className={l.chartPad}><div className={l.spacer} /><Skeleton height={64} radius="md" /><div className={l.spacer} /><Skeleton height={200} radius="md" /></div></Card>
      <div className={l.wide}><Skeleton height={280} radius="lg" /><Skeleton height={280} radius="lg" /></div>
      <div className={l.g3}><Skeleton height={240} radius="lg" /><Skeleton height={240} radius="lg" /><Skeleton height={240} radius="lg" /></div>
    </>
  )
}

export default function OverviewPage() {
  const { t } = useTranslation()
  const f = useFormat()
  const q = useOverview()
  const [metricParam, setMetric] = useSearchParam('metric', 'time')
  const metric: Metric = METRICS.includes(metricParam as Metric) ? (metricParam as Metric) : 'time'
  const data = q.data && !Array.isArray(q.data) && q.data.totals ? q.data : undefined

  return (
    <div className={l.page}>
      <PageHeader
        title={t('app.overview.title')}
        subtitle={data ? t('app.overview.subtitle', { views: f.number(data.totals.views), users: data.totals.activeUsers }) : undefined}
      />
      {!data && q.isError ? (
        <Card><ErrorState title={t('app.overview.error')} onRetry={() => q.refetch()} /></Card>
      ) : !data ? (
        <OverviewSkeleton />
      ) : (
        <OverviewBody data={data} metric={metric} setMetric={setMetric} />
      )}
    </div>
  )
}

function OverviewBody({ data, metric, setMetric }: { data: Overview; metric: Metric; setMetric: (m: string) => void }) {
  const { t } = useTranslation()
  const f = useFormat()
  const [top, setTop] = useState<TopFilter>('all')
  const { totals, previous } = data
  const daily = data.daily ?? []
  const labels = daily.map((d) => f.day(dateOf(d.date)))
  const values = daily.map(pick(metric))
  const prevValues = data.previousDaily?.map(pick(metric))
  const metricLabel = t(`app.overview.metric.${metric}`)
  const yFormat = metric === 'time' ? (v: number) => t('app.overview.hoursValue', { value: f.number(v) }) : f.compact
  const xEvery = Math.max(1, Math.ceil(daily.length / 7))

  const topRows = useMemo(() => (data.topItems ?? [])
    .filter((it) => top === 'all' || it.kind === top)
    .slice(0, 8)
    .map((it) => ({
      key: `${it.kind}:${it.id}`, label: it.name, value: it.views, to: groupHref(it.kind, it.id),
      lead: <Art title={it.name} kind={it.kind === 'album' ? 'square' : 'poster'} src={itemImageUrl(it.artId || it.id, 64)} />,
      badge: top === 'all' ? <MediaTypeBadge type={groupFamily(it.kind)} /> : undefined,
    })), [data.topItems, top])

  const typeSeconds = daily.reduce((a, d) => ({
    movie: a.movie + (d.byType?.movie?.watchSeconds ?? 0),
    series: a.series + (d.byType?.series?.watchSeconds ?? 0),
    music: a.music + (d.byType?.music?.watchSeconds ?? 0),
  }), { movie: 0, series: 0, music: 0 })
  const typeTotal = typeSeconds.movie + typeSeconds.series + typeSeconds.music
  const methods = data.byMethod ?? { direct: 0, stream: 0, transcode: 0 }
  const methodTotal = methods.direct + methods.stream + methods.transcode
  const peak = data.concurrencyPeak ?? []
  const peakMax = peak.reduce((a, p, i) => (p.peak > (peak[a]?.peak ?? 0) ? i : a), -1)
  const weekHour = data.weekHour?.length === 7 ? data.weekHour : Array.from({ length: 7 }, () => Array<number>(24).fill(0))
  const weekdays = f.weekdays()

  const tile = (m: Metric, value: string, unit?: string) => (
    <StatTile
      key={m}
      label={t(`app.overview.metric.${m}`)}
      value={value}
      unit={unit}
      delta={previous ? ratioDelta(pick(m)(totals), pick(m)(previous), f) : undefined}
      trend={daily.length > 1 ? daily.map(pick(m)) : undefined}
      selected={metric === m}
      onSelect={() => setMetric(m)}
    />
  )

  return (
    <>
      <Card>
        <StatStrip columns={5}>
          {tile('time', f.number(totals.watchSeconds / 3600), t('app.overview.hoursUnit'))}
          {tile('views', f.number(totals.views))}
          {tile('clicks', f.number(totals.clicks))}
          <StatTile label={t('app.overview.activeUsers')} value={f.number(totals.activeUsers)} delta={previous ? ratioDelta(totals.activeUsers, previous.activeUsers, f) : undefined} />
          <StatTile
            label={t('app.overview.completion')}
            value={totals.completion === null ? '—' : f.number(totals.completion * 100)}
            unit={totals.completion === null ? undefined : '%'}
            delta={previous ? pointsDelta(totals.completion, previous.completion, t) : undefined}
          />
        </StatStrip>
      </Card>

      <ChartFrame
        title={t('app.overview.perDay', { metric: metricLabel })}
        csvName={`overview-${metric}`}
        legend={[
          { label: t('app.overview.currentPeriod'), color: 'var(--accent)', line: true },
          ...(prevValues ? [{ label: t('app.overview.previousPeriod'), color: 'var(--axis)', line: true }] : []),
        ]}
        table={{
          columns: [t('app.overview.date'), metricLabel, ...(prevValues ? [t('app.overview.previousPeriod')] : [])],
          rows: daily.map((d, i) => [d.date, Math.round(values[i] * 100) / 100, ...(prevValues ? [Math.round((prevValues[i] ?? 0) * 100) / 100] : [])]),
        }}
      >
        <LineChart
          labels={labels}
          series={[{ name: metricLabel, color: 'var(--accent)', values }]}
          compare={prevValues ? { name: t('app.overview.previousPeriod'), values: prevValues } : undefined}
          yFormat={yFormat}
          xEvery={xEvery}
          height={200}
        />
      </ChartFrame>

      <div className={l.wide}>
        <Card>
          <CardHeader
            title={t('app.overview.top.title')}
            subtitle={t('app.overview.top.subtitle')}
            action={
              <SegmentedControl<TopFilter>
                aria-label={t('app.overview.top.filter')}
                value={top}
                onValueChange={setTop}
                options={[
                  { value: 'all', label: t('app.overview.top.all') },
                  { value: 'movie', label: t('app.overview.top.movies') },
                  { value: 'series', label: t('app.overview.top.series') },
                  { value: 'album', label: t('app.overview.top.music') },
                ]}
              />
            }
          />
          <CardBody>
            {topRows.length === 0
              ? <EmptyState title={t('app.overview.top.empty')} />
              : <BarList rows={topRows} labelHeader={t('app.overview.top.content')} valueLabel={t('app.overview.metric.views')} />}
          </CardBody>
        </Card>
        <LiveCard />
      </div>

      <div className={l.g3}>
        <Card>
          <CardHeader title={t('app.overview.users.title')} subtitle={t('app.overview.users.subtitle')} />
          <CardBody>
            {(data.topUsers ?? []).length === 0 ? <EmptyState title={t('app.overview.users.empty')} /> : (
              <BarList
                labelHeader={t('app.overview.users.user')}
                valueLabel={t('app.overview.metric.time')}
                format={f.duration}
                rows={data.topUsers.slice(0, 8).map((u) => ({ key: u.id, label: u.name, value: u.watchSeconds, to: userHref(u.id), lead: <Avatar id={u.id} name={u.name} size="sm" /> }))}
              />
            )}
          </CardBody>
        </Card>
        <Card>
          <CardHeader title={t('app.overview.byType.title')} subtitle={t('app.overview.byType.subtitle')} />
          <CardBody>
            {typeTotal === 0 && methodTotal === 0 ? <EmptyState title={t('app.overview.byType.empty')} /> : (
              <div className={l.stack}>
                <SegBar parts={(['series', 'movie', 'music'] as const).map((k) => ({ label: t(`app.type.${k}`), value: typeSeconds[k], color: `var(--t-${k})`, display: f.duration(typeSeconds[k]) }))} />
                <div className={l.note}>{t('app.overview.byType.method')}</div>
                <SegBar parts={(['direct', 'stream', 'transcode'] as const).map((k) => ({ label: t(`app.method.${k}`), value: methods[k], color: `var(--m-${k})`, ink: `var(--m-${k}-ink)` }))} />
              </div>
            )}
          </CardBody>
        </Card>
        <ChartFrame
          title={t('app.overview.when.title')}
          subtitle={t('app.overview.when.subtitle')}
          csvName="overview-week-hour"
          table={{ columns: [t('app.overview.when.day'), ...Array.from({ length: 24 }, (_, h) => f.hour(h))], rows: weekHour.map((row, d) => [weekdays[d], ...row]) }}
        >
          <WeekHeatmap matrix={weekHour} unit={t('app.overview.metric.views')} />
        </ChartFrame>
      </div>

      <div className={l.g2}>
        <ChartFrame
          title={t('app.overview.peak.title')}
          subtitle={t('app.overview.peak.subtitle')}
          csvName="overview-concurrency-peak"
          table={{ columns: [t('app.overview.date'), t('app.overview.peak.series')], rows: peak.map((p) => [p.date, p.peak]) }}
        >
          {peak.length === 0 ? <EmptyState title={t('app.overview.peak.empty')} /> : (
            <Columns
              labels={peak.map((p) => f.day(dateOf(p.date)))}
              values={peak.map((p) => p.peak)}
              name={t('app.overview.peak.series')}
              emphasis={peakMax}
              xEvery={Math.max(1, Math.ceil(peak.length / 6))}
              height={160}
            />
          )}
        </ChartFrame>
        <Card>
          <CardHeader title={t('app.overview.onThisDay.title')} subtitle={t('app.overview.onThisDay.subtitle')} />
          <CardBody>
            {(data.onThisDay ?? []).length === 0 ? (
              <EmptyState icon={<CalendarClock size={20} strokeWidth={1.6} />} title={t('app.overview.onThisDay.empty')} />
            ) : (
              <ul className={`${l.stack} ${l.plainList}`}>
                {data.onThisDay.map((r) => (
                  <li key={`${r.itemId}:${r.date}`} className={l.cellMedia}>
                    <Art title={r.name} kind={r.kind === 'album' ? 'square' : 'poster'} src={itemImageUrl(r.itemId, 64)} />
                    <div className={`${l.ellipsis} ${l.grow}`}>
                      <AppLink to={groupHref(r.kind, r.itemId)}>{r.name}</AppLink>
                      <div className={l.cellSub}>{[r.users.join(', '), dateOf(r.date).getFullYear()].filter(Boolean).join(' · ')}</div>
                    </div>
                    <MediaTypeBadge type={groupFamily(r.kind)} />
                  </li>
                ))}
              </ul>
            )}
          </CardBody>
        </Card>
      </div>
    </>
  )
}
