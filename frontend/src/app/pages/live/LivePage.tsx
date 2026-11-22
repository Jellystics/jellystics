import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Radio } from 'lucide-react'
import { formatInTimeZone } from 'date-fns-tz'
import { useConcurrency } from '@/api/usersExtra'
import { sessionBitrate, useLiveSessions, type JellyfinSession } from '@/api/live'
import { ChartFrame, Columns, SessionTimeline } from '@/charts'
import { useFormat } from '@/lib/format'
import { useServerConfig } from '@/lib/serverConfig'
import { Card, CardBody, CardHeader, EmptyState, ErrorState, MethodBadge, PageHeader, SessionCard, Skeleton, StatStrip, StatTile } from '@/ui'
import { methodOf, wallClock } from '../history/links'
import { timelineRows } from './timeline'
import { toLiveSession } from './toLiveSession'
import l from '../overview/layout.module.css'

const pad = (n: number) => String(n).padStart(2, '0')
const slotLabel = (i: number, minutes: number) => `${pad(Math.floor((i * minutes) / 60))}:${pad((i * minutes) % 60)}`

/** Mb/s with one decimal under 10. */
function mbps(bps: number, f: ReturnType<typeof useFormat>): string {
  const v = bps / 1_000_000
  return v < 10 ? (Math.round(v * 10) / 10).toLocaleString(f.locale) : f.number(v)
}

export default function LivePage() {
  const { t } = useTranslation()
  const f = useFormat()
  const live = useLiveSessions()
  const conc = useConcurrency(undefined, undefined, { refetchInterval: 60_000 })
  const tz = useServerConfig().data?.timezone ?? Intl.DateTimeFormat().resolvedOptions().timeZone
  const now = formatInTimeZone(new Date(), tz, 'HH:mm')
  const nowHour = Number(now.slice(0, 2)) + Number(now.slice(3)) / 60

  const sessions = useMemo(() => live.data ?? [], [live.data])
  const day = conc.data && !Array.isArray(conc.data) && Array.isArray(conc.data.slots) ? conc.data : undefined
  const transcodes = sessions.filter((s) => methodOf(s.PlayState?.PlayMethod) === 'transcode')
  const hw = transcodes.filter((s) => s.TranscodingInfo?.HardwareAccelerationType && s.TranscodingInfo.HardwareAccelerationType !== 'none').length
  const bitrates = sessions.map((s) => ({ s, bps: sessionBitrate(s) })).filter((b): b is { s: JellyfinSession; bps: number } => !!b.bps)
  const totalBps = bitrates.reduce((a, b) => a + b.bps, 0)
  const peak = Math.max(day?.peak ?? 0, sessions.length)
  const timeline = useMemo(() => timelineRows(day, sessions, nowHour), [day, sessions, nowHour])

  const minutes = day?.slotMinutes || 15
  const perHour = Math.round(60 / minutes)
  const firstActive = day ? day.slots.findIndex((v) => v > 0) : -1
  const fromSlot = Math.max(0, Math.min(8 * perHour, firstActive >= 0 ? Math.floor(firstActive / perHour) * perHour : 8 * perHour))
  const slots = day ? day.slots.slice(fromSlot) : []

  return (
    <div className={l.page}>
      <PageHeader title={t('app.live.title')} subtitle={t('app.live.subtitle', { count: sessions.length })} />
      <Card>
        <StatStrip columns={4}>
          <StatTile label={t('app.live.playing')} value={live.isPending ? '—' : f.number(sessions.length)} />
          <StatTile label={t('app.live.transcodes')} value={live.isPending ? '—' : f.number(transcodes.length)} foot={transcodes.length ? t('app.live.hardware', { count: hw }) : undefined} />
          <StatTile label={t('app.live.bandwidth')} value={bitrates.length ? mbps(totalBps, f) : '—'} unit={bitrates.length ? t('app.live.mbps') : undefined} />
          <StatTile label={t('app.live.peak')} value={f.number(peak)} foot={day?.peakAt ? t('app.live.peakAt', { time: wallClock(day.peakAt) }) : undefined} />
        </StatStrip>
      </Card>

      {live.isError && !live.data ? (
        <Card><ErrorState title={t('app.live.error')} onRetry={() => live.refetch()} /></Card>
      ) : live.isPending ? (
        <div className={l.g2}><Skeleton height={96} radius="lg" /><Skeleton height={96} radius="lg" /></div>
      ) : sessions.length === 0 ? (
        <Card><EmptyState icon={<Radio size={20} strokeWidth={1.6} />} title={t('app.live.empty')} description={t('app.live.emptyHint')} /></Card>
      ) : (
        <div className={l.g2}>{sessions.map((s) => <SessionCard key={s.Id} session={toLiveSession(s, t)} />)}</div>
      )}

      {bitrates.length > 0 && (
        <Card>
          <CardHeader title={t('app.live.bandwidthTitle')} subtitle={t('app.live.bandwidthSubtitle')} />
          <CardBody>
            <ul className={`${l.stack} ${l.plainList}`}>
              {bitrates.sort((a, b) => b.bps - a.bps).map(({ s, bps }) => {
                const m = methodOf(s.PlayState?.PlayMethod)
                return (
                  <li key={s.Id} className={l.row}>
                    {m && <MethodBadge method={m} />}
                    <span className={`${l.ellipsis} ${l.grow}`}>{toLiveSession(s, t).title} · {s.UserName}</span>
                    <span className="num">{mbps(bps, f)} {t('app.live.mbps')}</span>
                  </li>
                )
              })}
            </ul>
          </CardBody>
        </Card>
      )}

      {conc.isError && !day ? (
        <Card><ErrorState title={t('app.live.concurrencyError')} onRetry={() => conc.refetch()} /></Card>
      ) : !day ? (
        <Skeleton height={200} radius="lg" />
      ) : (
        <>
          <ChartFrame
            title={t('app.live.simultaneous')}
            subtitle={t('app.live.simultaneousSubtitle', { minutes })}
            csvName={`live-concurrency-${day.date}`}
            table={{ columns: [t('app.live.time'), t('app.live.playing')], rows: day.slots.map((v, i) => [slotLabel(i, minutes), v]) }}
          >
            <Columns
              labels={slots.map((_, i) => slotLabel(i + fromSlot, minutes))}
              values={slots}
              name={t('app.live.playing')}
              xEvery={perHour * 2}
              height={140}
              emphasis={slots.indexOf(Math.max(0, ...slots))}
            />
          </ChartFrame>
          <ChartFrame
            title={t('app.live.timeline')}
            subtitle={t('app.live.timelineSubtitle')}
            csvName={`live-timeline-${day.date}`}
            table={{
              columns: [t('app.live.user'), t('app.live.content'), t('app.live.start'), t('app.live.end')],
              rows: day.sessions.map((s) => [s.user.name, s.parentName ? `${s.parentName} · ${s.name}` : s.name, wallClock(s.start), wallClock(s.end)]),
            }}
          >
            {timeline.rows.length === 0
              ? <EmptyState title={t('app.live.timelineEmpty')} />
              : <SessionTimeline fromHour={timeline.fromHour} toHour={timeline.toHour} rows={timeline.rows} />}
          </ChartFrame>
        </>
      )}
    </div>
  )
}
