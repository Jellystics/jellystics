import { useTranslation } from 'react-i18next'
import { itemImage } from '@/lib/images'
import { useParams } from 'react-router-dom'
import { ArrowDown, ArrowUp, ChevronRight, UserX } from 'lucide-react'
import { useRecap, type Recap } from '@/api/analytics'
import { ApiError } from '@/lib/api'
import { ChartFrame, Columns } from '@/charts'
import { AppLink, Art, BarList, ButtonLink, Card, CardBody, CardHeader, EmptyState, ErrorState, Skeleton, StatStrip, StatTile } from '@/ui'
import { useFormat } from '@/lib/format'
import { useSetCrumbTail } from '@/app/shell/crumbs'
import { useServerConfig } from '@/lib/serverConfig'
import { NoData } from '../_shared/shared'
import { entityPath } from '@/lib/routes'
import { formatHours, formatMonth } from '../_shared/analyticsFormat'
import shared from '../_shared/analytics.module.css'
import styles from './RecapPage.module.css'
import { recapOptions } from './recapOptions'

const PERIOD_RE = /^\d{4}(-(0[1-9]|1[0-2]))?$/
const typeFill = (t: string) => `color-mix(in srgb, var(--t-${t}) 18%, transparent)`

export default function RecapPage() {
  const { t } = useTranslation()
  const f = useFormat()
  const { id = '', period = '' } = useParams()
  const valid = PERIOD_RE.test(period)
  const recap = useRecap(id, valid ? period : undefined)
  const label = (p: string) => (p.length === 4 ? p : new Intl.DateTimeFormat(f.locale, { month: 'long', year: 'numeric' }).format(new Date(Number(p.slice(0, 4)), Number(p.slice(5, 7)) - 1, 1)))
  // Users › Léa › Recap 2026 (the user name replaces the generic "user" crumb).
  useSetCrumbTail(valid ? [recap.data?.user.name, t('app.recap.crumb', { period: label(period) })] : null)
  const tz = useServerConfig().data?.timezone

  const switcher = valid && (
    <nav className={styles.switch} aria-label={t('app.recap.periods')}>
      {recapOptions(period, new Date(), tz).map((p) => (
        <ButtonLink key={p} size="sm" variant={p === period ? 'secondary' : 'ghost'} aria-current={p === period ? 'page' : undefined}
          to={`/users/${encodeURIComponent(id)}/recap/${p}`}>
          {p.length === 4 ? p : formatMonth(p, f.locale)}
        </ButtonLink>
      ))}
    </nav>
  )

  if (!valid) return <EmptyState headingLevel={1} title={t('app.recap.badPeriod')} description={t('app.recap.badPeriodHint')} />
  if (recap.data === undefined) {
    if (recap.isError) {
      if (recap.error instanceof ApiError && recap.error.status === 404) {
        return <EmptyState headingLevel={1} icon={<UserX size={20} strokeWidth={1.6} />} title={t('app.recap.notFound')} />
      }
      return <ErrorState title={t('app.error.generic')} onRetry={() => { recap.refetch() }} />
    }
    return (
      <div className={shared.page} aria-busy="true">
        <Card><CardBody><div className={styles.heroSkeleton}><Skeleton height={48} width={220} radius="md" /><Skeleton width={320} /></div></CardBody></Card>
        <div className={shared.grid3}>{[0, 1, 2].map((i) => <Card key={i}><CardBody><div className={shared.tilesSkeleton}><Skeleton height={120} radius="md" /></div></CardBody></Card>)}</div>
      </div>
    )
  }
  return <RecapView data={recap.data} label={label(period)} switcher={switcher} />
}

function RecapView({ data: d, label, switcher }: { data: Recap; label: string; switcher: React.ReactNode }) {
  const { t } = useTranslation()
  const f = useFormat()
  const hours = (s: number) => formatHours(s, f.locale)
  const bucketLabel = (k: string) => (d.granularity === 'month' ? formatMonth(k, f.locale, false) : String(Number(k.slice(8, 10))))
  const empty = d.totals.views === 0 && d.totals.watchSeconds === 0
  const best = d.buckets.reduce((a, b, i) => (b.watchSeconds > (d.buckets[a]?.watchSeconds ?? 0) ? i : a), 0)

  return (
    <div className={shared.page}>
      <Card>
        <div className={styles.hero}>
          <h1 className="sr-only">{`${t('app.crumb.recap')} · ${d.user.name} · ${label}`}</h1>
          <div className={shared.muted}>
            <AppLink to={`/users/${encodeURIComponent(d.user.id)}`}>{d.user.name}</AppLink>{` · ${label}`}
          </div>
          <div className={styles.heroValue}>{t('app.recap.hoursValue', { value: f.number(Math.round(d.totals.watchSeconds / 3600)) })}</div>
          <div className={styles.heroSub}>
            {t('app.recap.heroSub', { days: f.number(Math.floor(d.totals.watchSeconds / 86400)), views: f.number(d.totals.views), active: f.number(d.totals.activeDays) })}
          </div>
          {switcher}
        </div>
      </Card>

      {empty ? <Card><CardBody><NoData /></CardBody></Card> : (
        <>
          <div className={shared.grid3}>
            <Card>
              <CardHeader title={t('app.recap.topSeries')} />
              <CardBody>
                {d.topSeries.length ? (
                  <BarList valueLabel="" format={hours} rows={d.topSeries.map((s) => ({ key: s.id, label: s.name, value: s.watchSeconds, to: entityPath('series', s.id), lead: <Art title={s.name} seed={s.id} src={itemImage(s.id)} />, fill: typeFill('series') }))} />
                ) : <p className={shared.muted}>{t('app.recap.none')}</p>}
              </CardBody>
            </Card>
            <Card>
              <CardHeader title={t('app.recap.topMovies')} />
              <CardBody>
                {d.topMovies.length ? (
                  <BarList valueLabel="" format={(v) => t('app.recap.views', { count: v })} rows={d.topMovies.map((m) => ({ key: m.id, label: m.name, value: m.views, to: entityPath('movie', m.id), lead: <Art title={m.name} seed={m.id} src={itemImage(m.id)} />, fill: typeFill('movie') }))} />
                ) : <p className={shared.muted}>{t('app.recap.none')}</p>}
              </CardBody>
            </Card>
            <Card>
              <CardHeader title={t('app.recap.topArtists')} />
              <CardBody>
                {d.topArtists.length ? (
                  <BarList valueLabel="" format={hours} rows={d.topArtists.map((a) => ({ key: a.id, label: a.name, value: a.watchSeconds, to: entityPath('artist', a.id), lead: <Art title={a.name} seed={a.id} src={itemImage(a.id)} kind="square" />, fill: typeFill('music') }))} />
                ) : <p className={shared.muted}>{t('app.recap.none')}</p>}
              </CardBody>
            </Card>
          </div>

          <ChartFrame
            title={d.granularity === 'month' ? t('app.recap.byMonth') : t('app.recap.byDay')} subtitle={t('app.recap.hours')} csvName={`recap-${d.period}`}
            table={{ columns: [t('app.recap.bucket'), t('app.recap.hours'), t('app.chart.views')], rows: d.buckets.map((b) => [b.key, Math.round(b.watchSeconds / 360) / 10, b.views]) }}
          >
            <Columns labels={d.buckets.map((b) => bucketLabel(b.key))} values={d.buckets.map((b) => b.watchSeconds)} name={t('app.recap.hours')}
              yFormat={hours} emphasis={best} xEvery={d.granularity === 'month' ? 1 : 5} />
          </ChartFrame>

          <Card>
            <StatStrip columns={3}>
              <StatTile label={d.granularity === 'month' ? t('app.recap.bestMonth') : t('app.recap.bestDay')}
                value={d.records.bestBucket ? hours(d.records.bestBucket.watchSeconds) : t('app.recap.noValue')}
                foot={d.records.bestBucket ? (d.granularity === 'month' ? formatMonth(d.records.bestBucket.key, f.locale, true) : f.day(new Date(`${d.records.bestBucket.key}T12:00:00`))) : undefined} />
              <StatTile label={t('app.recap.binge')} value={d.records.binge ? t('app.recap.episodes', { count: d.records.binge.episodes }) : t('app.recap.noValue')}
                foot={d.records.binge ? (
                  <span><AppLink to={entityPath('series', d.records.binge.seriesId)}>{d.records.binge.seriesName}</AppLink>{` · ${f.day(new Date(`${d.records.binge.date}T12:00:00`))}`}</span>
                ) : undefined} />
              <StatTile label={t('app.recap.streak')} value={t('app.recap.days', { count: d.records.longestStreak })} />
            </StatStrip>
          </Card>

          <div className={shared.grid}>
            <ChartFrame
              title={t('app.recap.artistDiscovery.title')} subtitle={t('app.recap.artistDiscovery.subtitle')} csvName={`recap-artists-${d.period}`}
              table={{ columns: [t('app.recap.bucket'), t('app.recap.artistDiscovery.value')], rows: d.buckets.map((b) => [b.key, b.newArtists]) }}
            >
              <Columns labels={d.buckets.map((b) => bucketLabel(b.key))} values={d.buckets.map((b) => b.newArtists)} name={t('app.recap.artistDiscovery.value')}
                emphasis={d.buckets.reduce((a, b, i) => (b.newArtists > (d.buckets[a]?.newArtists ?? 0) ? i : a), 0)} xEvery={d.granularity === 'month' ? 1 : 5} />
            </ChartFrame>

            <Card>
              <CardHeader title={t('app.recap.ranking.title')} subtitle={t('app.recap.ranking.subtitle')} />
              <CardBody flush>
                <table className={shared.table}>
                  <thead className={shared.srHead}>
                    <tr>
                      <th scope="col"><span className="sr-only">{t('app.recap.ranking.rank')}</span></th>
                      <th scope="col"><span className="sr-only">{t('app.recap.ranking.content')}</span></th>
                      <th scope="col"><span className="sr-only">{t('app.recap.ranking.evolution')}</span></th>
                      <th scope="col"><span className="sr-only">{t('app.chart.views')}</span></th>
                    </tr>
                  </thead>
                  <tbody>
                    {d.ranking.map((r) => {
                      const delta = r.previousRank === null ? null : r.previousRank - r.rank
                      const Icon = delta === null || delta === 0 ? ChevronRight : delta > 0 ? ArrowUp : ArrowDown
                      const tone = delta === null ? styles.new : delta > 0 ? styles.up : delta < 0 ? styles.down : styles.flat
                      return (
                        <tr key={`${r.kind}/${r.id}`}>
                          <td className={`${shared.num} ${shared.muted}`}>{r.rank}</td>
                          <td>
                            <div className={shared.media}>
                              <Art title={r.name} seed={r.id} src={itemImage(r.id)} kind={r.kind === 'album' ? 'square' : 'poster'} />
                              <div>
                                <AppLink className={shared.title} to={entityPath(r.kind, r.id)}>{r.name}</AppLink>
                                {r.parentName && <div className={shared.sub}>{r.parentName}</div>}
                              </div>
                            </div>
                          </td>
                          <td>
                            <span className={`${styles.delta} ${tone}`}>
                              {delta === null ? t('app.recap.ranking.new') : (
                                <>
                                  <Icon size={12} aria-hidden="true" />
                                  {delta === 0 ? '=' : Math.abs(delta)}
                                  <span className={styles.srOnly}>{t(delta > 0 ? 'app.recap.ranking.up' : delta < 0 ? 'app.recap.ranking.down' : 'app.recap.ranking.same')}</span>
                                </>
                              )}
                            </span>
                          </td>
                          <td className={shared.num}>{f.number(r.views)}</td>
                        </tr>
                      )
                    })}
                  </tbody>
                </table>
              </CardBody>
            </Card>
          </div>
        </>
      )}
    </div>
  )
}
