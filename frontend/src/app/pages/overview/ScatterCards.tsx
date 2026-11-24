import { useMemo, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { useLocation, useNavigate } from 'react-router-dom'
import { useOverviewLibrariesDaily, useOverviewPlays, type LibrariesDaily, type MediaFamily, type OverviewPlays, type PlayPoint } from '@/api'
import { ChartFrame, Scatter } from '@/charts'
import { formatDateTime, zonedDate } from '@/charts/timeScale'
import { useFormat } from '@/lib/format'
import { entityPath } from '@/lib/routes'
import { useServerConfig } from '@/lib/serverConfig'
import { Card, CardBody, CardHeader, EmptyState, ErrorState, Skeleton, withPeriod } from '@/ui'
import { dateOf } from '../history/links'
import { librarySeries } from './librarySeries'
import l from './layout.module.css'

const FAMILIES: MediaFamily[] = ['movie', 'series', 'music']

type QueryLike<T> = { data?: T; isError: boolean; refetch: () => unknown }

/** Loading / error shell around a card whose data is not there yet. */
function Pending<T>({ title, subtitle, query, children }: { title: string; subtitle?: string; query: QueryLike<T>; children: (d: T) => ReactNode }) {
  const { t } = useTranslation()
  if (query.data !== undefined) return <>{children(query.data)}</>
  return (
    <Card>
      <CardHeader title={title} subtitle={subtitle} />
      <CardBody>
        {query.isError
          ? <ErrorState title={t('app.error.generic')} onRetry={() => { query.refetch() }} />
          : <div aria-busy="true"><Skeleton height={240} radius="md" /></div>}
      </CardBody>
    </Card>
  )
}

const playTitle = (p: PlayPoint) => (p.item.parentName ? `${p.item.parentName} › ${p.item.name}` : p.item.name)

function useTimeZone() {
  return useServerConfig().data?.timezone ?? Intl.DateTimeFormat().resolvedOptions().timeZone
}

/** « Toutes les lectures »: one dot per play, x = start, y = watched minutes, colour = media type. */
export function PlaysScatterCard() {
  const { t } = useTranslation()
  const q = useOverviewPlays()
  const title = t('app.overview.plays.title'), subtitle = t('app.overview.plays.subtitle')
  return <Pending title={title} subtitle={subtitle} query={q}>{(d) => <PlaysScatter data={d} title={title} subtitle={subtitle} />}</Pending>
}

function PlaysScatter({ data, title, subtitle }: { data: OverviewPlays; title: string; subtitle: string }) {
  const { t } = useTranslation()
  const f = useFormat()
  const tz = useTimeZone()
  const navigate = useNavigate()
  const { search } = useLocation()
  // Oldest first, so the newest dots are drawn on top.
  const plays = useMemo(() => [...(data.points ?? [])].reverse(), [data.points])
  const present = FAMILIES.filter((k) => plays.some((p) => p.type === k))
  const cols = { date: t('app.overview.plays.date'), content: t('app.overview.plays.content'), user: t('app.overview.plays.user'), type: t('app.overview.plays.type'), minutes: t('app.overview.plays.minutes') }
  const points = useMemo(() => plays.map((p) => {
    const ms = Date.parse(p.startedAt)
    return {
      x: ms, y: Math.max(0, p.watchSeconds) / 60, label: playTitle(p), color: `var(--t-${p.type})`,
      rows: [
        { label: cols.user, value: p.user.name },
        { label: cols.date, value: formatDateTime(ms, f.locale, tz) },
        { label: t('app.overview.plays.duration'), value: f.duration(Math.max(0, p.watchSeconds)) },
        { color: `var(--t-${p.type})`, label: cols.type, value: t(`app.type.${p.type}`) },
      ],
    }
  }), [plays, f, tz, t, cols.user, cols.date, cols.type])

  return (
    <ChartFrame
      title={title}
      subtitle={subtitle}
      csvName="overview-plays"
      legend={present.map((k) => ({ label: t(`app.type.${k}`), color: `var(--t-${k})` }))}
      table={{
        columns: [cols.date, cols.content, cols.user, cols.type, cols.minutes],
        rows: plays.map((p) => [p.startedAt, playTitle(p), p.user.name, t(`app.type.${p.type}`), Math.round((Math.max(0, p.watchSeconds) / 60) * 10) / 10]),
      }}
    >
      {plays.length === 0 ? <EmptyState title={t('app.overview.plays.empty')} /> : (
        <>
          <Scatter
            points={points}
            timeAxis
            timeZone={tz}
            radius={4}
            xLabel={t('app.overview.plays.x')}
            yLabel={t('app.overview.plays.y')}
            yFormat={f.axis}
            onPointClick={(i) => { const it = plays[i].item; navigate(withPeriod(entityPath(it.kind, it.id, it.parentId), new URLSearchParams(search))) }}
          />
          {data.capped && <div className={l.note}>{t('app.overview.plays.capped', { value: f.number(data.limit) })}</div>}
        </>
      )}
    </ChartFrame>
  )
}

/** « Lectures par bibliothèque »: one dot per (day, library), y = views. */
export function LibrariesScatterCard() {
  const { t } = useTranslation()
  const q = useOverviewLibrariesDaily()
  const title = t('app.overview.libraries.title')
  return <Pending title={title} subtitle={t('app.overview.libraries.subtitle')} query={q}>{(d) => <LibrariesScatter data={d} title={title} />}</Pending>
}

function LibrariesScatter({ data, title }: { data: LibrariesDaily; title: string }) {
  const { t } = useTranslation()
  const f = useFormat()
  const tz = useTimeZone()
  const week = data.bucket === 'week'
  const { series, points } = useMemo(() => librarySeries(data, t('app.overview.libraries.other')), [data, t])
  const byId = new Map(series.map((s) => [s.id, s]))
  const when = (date: string) => (week ? t('app.overview.libraries.week', { date: f.day(dateOf(date)) }) : f.day(dateOf(date)))
  const views = t('app.overview.libraries.y')
  const dots = points.map((p) => {
    const s = byId.get(p.seriesId)!
    return {
      x: zonedDate(p.date, tz), y: p.views, label: s.name, color: s.color,
      rows: [{ label: t('app.overview.libraries.x'), value: when(p.date) }, { color: s.color, label: views, value: f.number(p.views) }],
    }
  })

  return (
    <ChartFrame
      title={title}
      subtitle={t(week ? 'app.overview.libraries.subtitleWeek' : 'app.overview.libraries.subtitle')}
      csvName="overview-plays-by-library"
      legend={series.map((s) => ({ label: s.name, color: s.color }))}
      table={{
        columns: [t('app.overview.libraries.x'), t('app.overview.libraries.library'), views],
        rows: points.map((p) => [p.date, byId.get(p.seriesId)!.name, p.views]),
      }}
    >
      {dots.length === 0 ? <EmptyState title={t('app.overview.libraries.empty')} /> : (
        <Scatter
          points={dots}
          timeAxis
          timeZone={tz}
          xLabel={t('app.overview.libraries.x')}
          yLabel={views}
          yFormat={f.compact}
        />
      )}
    </ChartFrame>
  )
}
