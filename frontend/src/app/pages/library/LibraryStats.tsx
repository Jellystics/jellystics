import { useTranslation } from 'react-i18next'
import { AppLink, BarList, Card, CardBody, CardHeader, EmptyState, SegBar, StatStrip, StatTile } from '@/ui'
import { ChartFrame, Columns, LineChart } from '@/charts'
import { useFormat } from '@/lib/format'
import type { LibraryDetail } from '@/api'
import { useLibraryDecades, useLibraryGrowth, type DecadeRow, type GrowthBucket } from '@/api/entityExtras'
import { useLibraryFirstViewDelay, type FirstViewBucket } from '@/api/plugin'
import { PluginSinceNote } from '@/app/widgets/PluginWidgets'
import { usePeriod } from '@/lib/useUrlState'
import { entityPath } from '@/lib/routes'
import { SubQuery } from './EntityParts'
import styles from './entity.module.css'
import local from './library.module.css'

function Decades({ rows }: { rows: DecadeRow[] }) {
  const { t } = useTranslation()
  const f = useFormat()
  const views = rows.reduce((a, r) => a + r.views, 0) || 1
  const catalog = rows.reduce((a, r) => a + r.catalog, 0) || 1
  const max = Math.max(0.0001, ...rows.flatMap((r) => [r.views / views, r.catalog / catalog]))
  return (
    <div className={local.decades} role="img" aria-label={t('app.library.decades.title')}>
      {rows.map((r) => {
        const v = r.views / views, c = r.catalog / catalog
        return (
          <div key={r.decade} className={local.decade}>
            <span className={`num ${local.decadeLabel}`}>{t('app.library.decades.label', { decade: r.decade })}</span>
            <span className={local.bars}>
              <span className={local.barViews} style={{ width: `${(v / max) * 100}%` }} title={`${t('app.library.decades.views')} : ${f.percent(v)}`} />
              <span className={local.barCatalog} style={{ width: `${(c / max) * 100}%` }} title={`${t('app.library.decades.catalog')} : ${f.percent(c)}`} />
            </span>
          </div>
        )
      })}
    </div>
  )
}

function useBucketLabel(bucket: GrowthBucket) {
  const f = useFormat()
  return (date: string) => {
    const d = new Date(`${date}T00:00:00`)
    if (bucket === 'year') return String(d.getFullYear())
    if (bucket === 'month') return new Intl.DateTimeFormat(f.locale, { month: 'short', year: '2-digit' }).format(d)
    return f.day(d)
  }
}

function Growth({ id }: { id: string }) {
  const { t } = useTranslation()
  const f = useFormat()
  const [period] = usePeriod()
  const q = useLibraryGrowth(id)
  const label = useBucketLabel(q.data?.bucket ?? 'day')
  const points = q.data?.points ?? []
  const labels = points.map((p) => label(p.date))
  const events = q.data?.source === 'events'
  const added = { name: t('app.library.growth.added'), color: 'var(--accent)', values: points.map((p) => p.added) }
  const first = { name: t('app.library.growth.firstViews'), color: 'var(--t-music)', values: points.map((p) => p.firstViews) }
  // Removals are only known from the plugin's library events.
  const removed = { name: t('app.plugin.growth.removed'), color: 'var(--serious)', values: points.map((p) => p.removed ?? 0) }
  const showRemoved = events && removed.values.some((v) => v > 0)
  const series = showRemoved ? [added, removed, first] : [added, first]
  const empty = points.every((p) => p.added === 0 && p.firstViews === 0 && !(p.removed > 0))
  const since = q.data?.preciseFrom ?? null
  // Estimated buckets (catalog dates) are dashed; precise ones solid.
  const firstPrecise = events && points.some((p) => p.precise) ? points.findIndex((p) => p.precise) : points.length
  const note = !events
    ? t('app.plugin.growth.catalog')
    : since && since > period.from
      ? t('app.plugin.growth.partial', { date: f.day(new Date(`${since}T12:00:00`)) })
      : t('app.plugin.growth.precise')
  return (
    <ChartFrame
      title={t('app.library.growth.title')} subtitle={t('app.library.growth.subtitle')} csvName={`library-${id}-growth`}
      legend={series.map((s) => ({ label: s.name, color: s.color, line: true }))}
      table={{
        columns: [t('app.library.growth.date'), ...series.map((s) => s.name), t('app.plugin.growth.source')],
        rows: points.map((p, i) => [labels[i], ...series.map((s) => s.values[i]), p.precise ? t('app.plugin.growth.sourceEvents') : t('app.plugin.growth.sourceCatalog')]),
      }}
    >
      <SubQuery query={q} height={200} empty={empty ? <EmptyState title={t('app.library.growth.empty')} /> : null}>
        <LineChart labels={labels} series={series} area={false} height={200} preciseFrom={{ index: firstPrecise, label: t('app.plugin.growth.marker') }} xEvery={Math.max(1, Math.ceil(points.length / 8))} />
        <p className={local.note}>{note}</p>
      </SubQuery>
    </ChartFrame>
  )
}

const FIRST_VIEW_BUCKETS: FirstViewBucket[] = ['<1h', '1-24h', '1-7d', '1-4w', '>4w']

/** Delay between a real addition (plugin library events) and the first view. Hidden without events for the library. */
function FirstViewDelayCard({ id }: { id: string }) {
  const { t } = useTranslation()
  const f = useFormat()
  const d = useLibraryFirstViewDelay(id).data
  if (!d?.collectedSince) return null
  const counts = FIRST_VIEW_BUCKETS.map((b) => d.buckets.find((x) => x.bucket === b)?.count ?? 0)
  const labels = FIRST_VIEW_BUCKETS.map((b) => t(`app.plugin.firstView.b.${b}`))
  return (
    <ChartFrame
      title={t('app.plugin.firstView.title')} subtitle={t('app.plugin.firstView.subtitle')} csvName={`library-${id}-first-view-delay`}
      table={{ columns: [t('app.plugin.firstView.delay'), t('app.plugin.firstView.count')], rows: counts.map((n, i) => [labels[i], n]) }}
    >
      {d.added === 0 ? <EmptyState title={t('app.plugin.firstView.empty')} /> : (
        <>
          <Columns labels={labels} values={counts} name={t('app.plugin.firstView.count')} emphasis={counts.indexOf(Math.max(...counts))} height={150} />
          <p className={local.note}>
            {t('app.plugin.firstView.summary', { added: f.number(d.added), viewed: f.number(d.viewed) })}
            {d.medianSeconds != null && ` · ${t('app.plugin.firstView.median', { value: f.duration(d.medianSeconds) })}`}
            {d.notViewed > 0 && ` · ${t('app.plugin.firstView.notViewed', { count: d.notViewed })}`}
          </p>
        </>
      )}
      <PluginSinceNote since={d.collectedSince} />
    </ChartFrame>
  )
}

function DecadesCard({ id }: { id: string }) {
  const { t } = useTranslation()
  const f = useFormat()
  const q = useLibraryDecades(id)
  const rows = q.data ?? []
  const views = rows.reduce((a, r) => a + r.views, 0) || 1
  const catalog = rows.reduce((a, r) => a + r.catalog, 0) || 1
  return (
    <ChartFrame
      title={t('app.library.decades.title')} subtitle={t('app.library.decades.subtitle')} csvName={`library-${id}-decades`}
      legend={[{ label: t('app.library.decades.views'), color: 'var(--accent)' }, { label: t('app.library.decades.catalog'), color: 'var(--seq-2)' }]}
      table={{
        columns: [
          t('app.library.decades.decade'), t('app.library.decades.viewsCount'), t('app.library.decades.views'),
          t('app.library.decades.catalogCount'), t('app.library.decades.catalog'),
        ],
        rows: rows.map((r) => [t('app.library.decades.label', { decade: r.decade }), r.views, f.percent(r.views / views), r.catalog, f.percent(r.catalog / catalog)]),
      }}
    >
      <SubQuery query={q} height={160} empty={rows.length === 0 ? <EmptyState title={t('app.library.decades.empty')} /> : null}>
        <Decades rows={rows} />
      </SubQuery>
    </ChartFrame>
  )
}

export function LibraryStats({ id, data }: { id: string; data: LibraryDetail }) {
  const { t } = useTranslation()
  const f = useFormat()
  const m = data.methods
  const methodTotal = m.direct + m.stream + m.transcode
  const ttw = data.timeToWatch
  return (
    <div className={styles.stack}>
      <div className={styles.g2}>
        <Card>
          <CardHeader title={t('app.library.genres')} subtitle={t('app.library.genresSub')} />
          <CardBody>
            {data.genres.length === 0 ? <EmptyState title={t('app.library.noGenres')} /> : (
              <BarList
                labelHeader={t('app.library.genre')} valueLabel={t('app.entity.watchTime')} format={f.duration}
                rows={data.genres.map((g) => ({ key: g.name, label: g.name, value: g.watchSeconds, tooltip: [[t('app.entity.views'), f.number(g.views)]] }))}
              />
            )}
          </CardBody>
        </Card>
        <div className={styles.stack}>
          <Card>
            <CardHeader title={t('app.library.methods')} subtitle={t('app.library.methodsSub')} />
            <CardBody>
              {methodTotal === 0 ? <EmptyState title={t('app.library.noViews')} /> : (
                <SegBar parts={[
                  { label: t('app.method.direct'), value: m.direct, color: 'var(--m-direct)', ink: 'var(--m-direct-ink)' },
                  { label: t('app.method.stream'), value: m.stream, color: 'var(--m-stream)', ink: 'var(--m-stream-ink)' },
                  { label: t('app.method.transcode'), value: m.transcode, color: 'var(--m-transcode)', ink: 'var(--m-transcode-ink)' },
                ]} />
              )}
            </CardBody>
          </Card>
          <Card>
            <CardHeader title={t('app.library.timeToWatch')} subtitle={t('app.library.timeToWatchSub')} />
            <StatStrip columns={3}>
              <StatTile label={t('app.library.ttwMedian')} value={ttw.medianDays == null ? '—' : f.number(ttw.medianDays)} unit={ttw.medianDays == null ? undefined : t('app.library.days')} />
              <StatTile label={t('app.library.ttwAvg')} value={ttw.avgDays == null ? '—' : f.number(ttw.avgDays)} unit={ttw.avgDays == null ? undefined : t('app.library.days')} />
              <StatTile label={t('app.library.ttwCount')} value={f.number(ttw.count)} />
            </StatStrip>
          </Card>
          <FirstViewDelayCard id={id} />
        </div>
      </div>
      <div className={styles.g2}>
        <DecadesCard id={id} />
        <Growth id={id} />
      </div>
      <Card>
        <CardHeader title={t('app.library.unwatched')} subtitle={t('app.library.unwatchedSub', { count: data.unwatched.count })} />
        <CardBody>
          {data.unwatched.list.length === 0 ? <EmptyState title={t('app.library.allWatched')} /> : (
            <ul className={styles.list}>
              {data.unwatched.list.map((u) => (
                <li key={u.id}>
                  <AppLink to={entityPath(u.kind, u.id)} className={styles.link}>{u.name}</AppLink>
                  <span className="num muted">{u.dateAdded ? t('app.library.added', { date: f.day(new Date(`${u.dateAdded}T00:00:00`)) }) : '—'}</span>
                </li>
              ))}
            </ul>
          )}
        </CardBody>
      </Card>
    </div>
  )
}
