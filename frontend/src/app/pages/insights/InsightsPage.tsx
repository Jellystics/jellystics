import { useTranslation } from 'react-i18next'
import {
  useInsightsBinges, useInsightsCompletion, useInsightsGenres, useInsightsMoments, useInsightsPauses,
  useInsightsResumes, useInsightsRewatches, useInsightsRhythm, useInsightsSessions,
} from '@/api/analytics'
import { ChartFrame, Columns, SmallMultiples, StackedArea, WeekHeatmap } from '@/charts'
import { AppLink, Art, BarList, Card, CardBody, CardHeader, PageHeader, SegBar } from '@/ui'
import { useFormat } from '@/lib/format'
import { QueryCard } from '../_shared/shared'
import { entityPath, formatHours, formatMonth } from '../_shared/analyticsFormat'
import styles from '../_shared/analytics.module.css'

const TYPES = ['movie', 'series', 'music'] as const
const typeFill = (t: string) => `color-mix(in srgb, var(--t-${t}) 18%, transparent)`
const sum = (v: number[]) => v.reduce((a, b) => a + b, 0)

export default function InsightsPage() {
  const { t } = useTranslation()
  const f = useFormat()
  const completion = useInsightsCompletion()
  const sessions = useInsightsSessions()
  const resumes = useInsightsResumes()
  const rewatches = useInsightsRewatches()
  const genres = useInsightsGenres()
  const moments = useInsightsMoments()
  const pauses = useInsightsPauses()
  const binges = useInsightsBinges()
  const rhythm = useInsightsRhythm()

  const decimal = (n: number) => new Intl.NumberFormat(f.locale).format(n)
  const completionLabels = Array.from({ length: 10 }, (_, i) => `${i * 10}–${i * 10 + 10} %`)
  const sessionLabels = [
    '< 1 min', '1–5 min', '5–15 min', '15–30 min', '30–45 min', '45–60 min',
    `1–${decimal(1.5)} h`, `${decimal(1.5)}–2 h`, '> 2 h',
  ]
  const typeLabel = (k: (typeof TYPES)[number]) => t(`app.type.${k}`)
  const hours = Array.from({ length: 24 }, (_, h) => f.hour(h))

  return (
    <div className={styles.page}>
      <PageHeader title={t('app.insights.title')} subtitle={t('app.insights.subtitle')} />
      <div className={styles.grid}>
        <QueryCard title={t('app.insights.completion.title')} query={completion} isEmpty={(d) => d.units === 0}>
          {(d) => (
            <ChartFrame
              title={t('app.insights.completion.title')} subtitle={t('app.insights.completion.subtitle')} csvName="completion"
              table={{ columns: [t('app.insights.completion.bucket'), t('app.chart.views')], rows: completionLabels.map((l, i) => [l, d.distribution[i] ?? 0]) }}
            >
              <Columns labels={completionLabels} values={d.distribution} name={t('app.chart.views')} emphasis={9} xEvery={3} />
            </ChartFrame>
          )}
        </QueryCard>

        <QueryCard title={t('app.insights.byType.title')} query={completion} isEmpty={(d) => d.units === 0}>
          {(d) => (
            <Card>
              <CardHeader title={t('app.insights.byType.title')} subtitle={t('app.insights.byType.subtitle')} />
              <CardBody>
                <BarList
                  labelHeader={t('app.insights.type')} valueLabel={t('app.insights.byType.value')} max={1} format={f.percent}
                  rows={TYPES.filter((k) => d.byType[k] !== null).map((k) => ({ key: k, label: typeLabel(k), value: d.byType[k] ?? 0, fill: typeFill(k) }))}
                />
              </CardBody>
            </Card>
          )}
        </QueryCard>

        <QueryCard title={t('app.insights.binges.title')} query={binges} isEmpty={(d) => d.length === 0}>
          {(d) => (
            <Card>
              <CardHeader title={t('app.insights.binges.title')} subtitle={t('app.insights.binges.subtitle')} />
              <CardBody>
                <BarList
                  labelHeader={t('app.insights.binges.series')} valueLabel={t('app.insights.binges.value')}
                  rows={d.map((b) => ({ key: b.id, label: b.name, value: b.binges, to: entityPath('series', b.id), lead: <Art title={b.name} seed={b.id} /> }))}
                />
              </CardBody>
            </Card>
          )}
        </QueryCard>

        <QueryCard title={t('app.insights.pauses.title')} query={pauses} isEmpty={(d) => d.length === 0}>
          {(d) => (
            <Card>
              <CardHeader title={t('app.insights.pauses.title')} subtitle={t('app.insights.pauses.subtitle')} />
              <CardBody>
                <BarList
                  labelHeader={t('app.insights.content')} valueLabel={t('app.insights.pauses.value')} format={f.percent}
                  rows={d.map((p) => ({
                    key: `${p.kind}/${p.id}`, label: p.name, value: p.ratio, to: entityPath(p.kind, p.id),
                    tooltip: [[t('app.insights.pauses.paused'), f.duration(p.pausedSeconds)], [t('app.insights.pauses.watched'), f.duration(p.watchSeconds)]],
                  }))}
                />
              </CardBody>
            </Card>
          )}
        </QueryCard>

        <QueryCard title={t('app.insights.rhythm.title')} query={rhythm} wide isEmpty={(d) => sum(d.weekHour.flat()) === 0}>
          {(d) => {
            const days = f.weekdays()
            return (
              <ChartFrame
                title={t('app.insights.rhythm.title')} subtitle={t('app.insights.rhythm.subtitle')} csvName="week-rhythm"
                table={{ columns: ['', ...hours], rows: d.weekHour.map((row, i) => [days[i] ?? '', ...row]) }}
              >
                <WeekHeatmap matrix={d.weekHour} unit={t('app.chart.views')} />
              </ChartFrame>
            )
          }}
        </QueryCard>

        <QueryCard title={t('app.insights.sessions.title')} query={sessions} isEmpty={(d) => d.sessions === 0}>
          {(d) => (
            <ChartFrame
              title={t('app.insights.sessions.title')}
              subtitle={d.medianSeconds !== null ? t('app.insights.sessions.median', { value: f.duration(d.medianSeconds) }) : undefined}
              csvName="session-length"
              table={{ columns: [t('app.insights.sessions.length'), t('app.insights.sessions.value')], rows: sessionLabels.map((l, i) => [l, d.buckets[i] ?? 0]) }}
            >
              <Columns labels={sessionLabels} values={d.buckets} name={t('app.insights.sessions.value')} emphasis={d.buckets.indexOf(Math.max(...d.buckets))} xEvery={2} />
            </ChartFrame>
          )}
        </QueryCard>

        <QueryCard title={t('app.insights.resumes.title')} query={resumes} isEmpty={(d) => d.fromStart + d.resumed === 0}>
          {(d) => (
            <Card>
              <CardHeader title={t('app.insights.resumes.title')} subtitle={t('app.insights.resumes.subtitle')} />
              <CardBody>
                <SegBar parts={[
                  { label: t('app.insights.resumes.fromStart'), value: d.fromStart, color: 'var(--m-direct)', ink: 'var(--m-direct-ink)' },
                  { label: t('app.insights.resumes.resumed'), value: d.resumed, color: 'var(--m-transcode)', ink: 'var(--m-transcode-ink)' },
                ]} />
                <div className={styles.gap} />
                <BarList
                  labelHeader={t('app.insights.type')} valueLabel={t('app.insights.resumes.share')} max={1} format={f.percent}
                  rows={TYPES.filter((k) => d.byType[k] !== null).map((k) => ({ key: k, label: typeLabel(k), value: d.byType[k] ?? 0 }))}
                />
              </CardBody>
            </Card>
          )}
        </QueryCard>

        <QueryCard title={t('app.insights.rewatches.title')} query={rewatches} isEmpty={(d) => d.length === 0}>
          {(d) => (
            <Card>
              <CardHeader title={t('app.insights.rewatches.title')} subtitle={t('app.insights.rewatches.subtitle')} />
              <CardBody flush>
                <table className={styles.table}>
                  <thead>
                    <tr><th scope="col">{t('app.insights.content')}</th><th scope="col">{t('app.insights.rewatches.user')}</th><th scope="col" className={styles.num}>{t('app.insights.rewatches.value')}</th></tr>
                  </thead>
                  <tbody>
                    {d.map((r) => {
                      const to = entityPath(r.kind, r.id, r.parentId)
                      return (
                        <tr key={`${r.id}/${r.user.id}`}>
                          <td>
                            <div className={styles.media}>
                              <Art title={r.parentName ?? r.name} seed={r.parentId ?? r.id} kind={r.kind === 'track' ? 'square' : 'poster'} />
                              <div>
                                {to ? <AppLink className={styles.title} to={to}>{r.name}</AppLink> : <span className={styles.title}>{r.name}</span>}
                                {r.parentName && <div className={styles.sub}>{r.parentName}</div>}
                              </div>
                            </div>
                          </td>
                          <td><AppLink to={`/users/${encodeURIComponent(r.user.id)}`}>{r.user.name}</AppLink></td>
                          <td className={styles.num}>{f.number(r.views)}</td>
                        </tr>
                      )
                    })}
                  </tbody>
                </table>
              </CardBody>
            </Card>
          )}
        </QueryCard>

        <QueryCard title={t('app.insights.genres.title')} query={genres} wide isEmpty={(d) => d.series.length === 0}>
          {(d) => {
            const colors = ['var(--t-movie)', 'var(--t-series)', 'var(--t-music)']
            const series = [
              { name: t('app.insights.genres.other'), color: 'var(--axis)', values: d.other },
              ...d.series.map((s, i) => ({ name: s.name, color: colors[i], values: s.values })).reverse(),
            ]
            const labels = d.months.map((m) => formatMonth(m, f.locale))
            return (
              <ChartFrame
                title={t('app.insights.genres.title')} subtitle={t('app.insights.genres.subtitle')} csvName="genres"
                legend={[...series].reverse().map((s) => ({ label: s.name, color: s.color }))}
                table={{ columns: [t('app.insights.genres.month'), ...series.map((s) => s.name)], rows: d.months.map((m, i) => [m, ...series.map((s) => Math.round((s.values[i] ?? 0) / 360) / 10)]) }}
              >
                <StackedArea labels={labels} series={series} yFormat={(v) => formatHours(v, f.locale)} />
              </ChartFrame>
            )
          }}
        </QueryCard>

        <QueryCard title={t('app.insights.moments.title')} query={moments} wide isEmpty={(d) => sum([...d.movie, ...d.series, ...d.music]) === 0}>
          {(d) => (
            <ChartFrame
              title={t('app.insights.moments.title')} subtitle={t('app.insights.moments.subtitle')} csvName="moments"
              table={{ columns: [t('app.insights.moments.hour'), ...TYPES.map(typeLabel)], rows: hours.map((h, i) => [h, d.movie[i] ?? 0, d.series[i] ?? 0, d.music[i] ?? 0]) }}
            >
              <SmallMultiples items={TYPES.map((k) => ({ name: typeLabel(k), color: `var(--t-${k})`, values: d[k] }))} />
            </ChartFrame>
          )}
        </QueryCard>
      </div>
    </div>
  )
}
