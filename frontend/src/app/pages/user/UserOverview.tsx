import { useTranslation } from 'react-i18next'
import type { ColumnDef } from '@tanstack/react-table'
import { AlertTriangle, Tv } from 'lucide-react'
import type { UserDetail } from '@/api'
import { itemImageUrl } from '@/api/live'
import { CalendarHeatmap, ChartFrame, Columns } from '@/charts'
import { useFormat } from '@/lib/format'
import { usePeriod } from '@/lib/useUrlState'
import { AppLink, Art, Badge, Card, CardBody, CardHeader, DataTable, EmptyState, Meter, StatStrip, StatTile } from '@/ui'
import { ratioDelta, pointsDelta } from '../overview/delta'
import { LanguagesCard } from '@/app/widgets/LanguagesCard'
import { FinishedFoot, LoginsCard } from '@/app/widgets/PluginWidgets'
import { hasPlayedToEnd } from '@/app/widgets/plugin'
import { ComparePanel } from './ComparePanel'
import { StreakCard } from './StreakCard'
import { activeDays, outdated, peakIndex } from './stats'
import l from '../overview/layout.module.css'
import s from './user.module.css'

type Device = UserDetail['devices'][number]

/** Overview tab of /users/:id. */
export function UserOverview({ user }: { user: UserDetail }) {
  const { t } = useTranslation()
  const f = useFormat()
  const [period] = usePeriod()
  const { totals, previous } = user
  const { active, days } = activeDays(user.calendar ?? [], period.from, period.to)
  const hours = user.hours?.length === 24 ? user.hours : Array<number>(24).fill(0)
  const peak = peakIndex(hours)
  const old = outdated(user.devices ?? [])
  const showPte = hasPlayedToEnd(user.playedToEnd)

  const deviceCols: ColumnDef<Device>[] = [
    { id: 'device', header: t('app.userPage.devices.device'), cell: ({ row: { original: d } }) => d.device || '—' },
    { id: 'client', header: t('app.userPage.devices.client'), cell: ({ row: { original: d } }) => <span className="ink2">{d.client || '—'}</span> },
    {
      id: 'version', header: t('app.userPage.devices.version'),
      cell: ({ row }) => old.has(row.index)
        ? <Badge tone="warning" icon={<AlertTriangle size={12} aria-hidden="true" />}>{t('app.userPage.devices.outdated', { version: row.original.version })}</Badge>
        : <span className={s.version}>{row.original.version || '—'}</span>,
    },
    { id: 'views', header: t('app.userPage.devices.views'), meta: { numeric: true }, cell: ({ row: { original: d } }) => f.number(d.views) },
  ]

  const scores = [
    ['regularity', user.profile.regularity],
    ['discovery', user.profile.discovery],
    ['concentration', user.profile.concentration],
    ['completion', user.profile.completion],
  ] as const
  // profile.completion is the share of finished views (≥ 90 %), not the average completion of the tiles.
  const scoreLabel = (k: (typeof scores)[number][0]) => (k === 'completion' ? t('app.metric.finishedViews') : t(`app.userPage.profile.${k}`))

  return (
    <div className={s.tabBody}>
      <Card>
        <StatStrip columns={5}>
          <StatTile label={t('app.userPage.tiles.time')} value={f.number(totals.watchSeconds / 3600)} unit={t('app.userPage.tiles.hoursUnit')} delta={previous ? ratioDelta(totals.watchSeconds, previous.watchSeconds, f) : undefined} />
          <StatTile label={t('app.userPage.tiles.views')} value={f.number(totals.views)} delta={previous ? ratioDelta(totals.views, previous.views, f) : undefined} />
          <StatTile label={t('app.userPage.tiles.activeDays')} value={f.number(active)} unit={`/ ${f.number(days)}`} />
          <StatTile label={t('app.userPage.tiles.streak')} value={f.number(user.streak.current)} unit={t('app.userPage.tiles.daysUnit')} foot={t('app.userPage.tiles.record', { count: user.streak.best })} />
          <StatTile
            label={t('app.metric.completionAvg')}
            value={totals.completion === null ? '—' : f.number(totals.completion * 100)}
            unit={totals.completion === null ? undefined : '%'}
            delta={previous ? pointsDelta(totals.completion, previous.completion, t) : undefined}
          />
        </StatStrip>
      </Card>

      <ChartFrame
        title={t('app.userPage.calendar.title')}
        subtitle={t('app.userPage.calendar.subtitle')}
        csvName={`user-${user.id}-calendar`}
        table={{ columns: [t('app.userPage.calendar.date'), t('app.userPage.tiles.time')], rows: (user.calendar ?? []).map((d) => [d.date, f.duration(d.watchSeconds)]) }}
      >
        <CalendarHeatmap days={(user.calendar ?? []).map((d) => ({ date: d.date, seconds: d.watchSeconds }))} />
      </ChartFrame>

      <div className={l.g2}>
        <Card>
          <CardHeader title={t('app.userPage.inProgress.title')} subtitle={t('app.userPage.inProgress.subtitle')} />
          <CardBody>
            {(user.inProgress ?? []).length === 0 ? <EmptyState icon={<Tv size={20} strokeWidth={1.6} />} title={t('app.userPage.inProgress.empty')} /> : (
              <div className={s.shelf}>
                {user.inProgress.slice(0, 6).map((p) => {
                  const next = p.nextSeason != null && p.nextEpisode != null ? t('app.userPage.inProgress.next', { season: p.nextSeason, episode: p.nextEpisode }) : t('app.userPage.inProgress.nextUnknown')
                  return (
                    <AppLink key={p.seriesId} to={`/series/${encodeURIComponent(p.seriesId)}/episodes/${encodeURIComponent(p.nextId)}`} className={s.shelfItem}>
                      <Art title={p.name} src={itemImageUrl(p.seriesId, 64)} />
                      <div className={`${l.grow} ${l.ellipsis}`}>
                        <div className={s.shelfTitle}>{p.name}</div>
                        <div className={s.shelfMeta}>{next} · {f.relative(new Date(p.lastPlayed))}</div>
                        <Meter value={p.progress} thin tone="series" label={t('app.userPage.inProgress.progress', { name: p.name })} />
                      </div>
                    </AppLink>
                  )
                })}
              </div>
            )}
          </CardBody>
        </Card>
        <ChartFrame
          title={t('app.userPage.clock.title')}
          subtitle={peak >= 0 ? t('app.userPage.clock.subtitle', { hour: f.hour(peak) }) : t('app.userPage.clock.subtitleEmpty')}
          csvName={`user-${user.id}-hours`}
          table={{ columns: [t('app.userPage.clock.hour'), t('app.userPage.clock.views')], rows: hours.map((v, h) => [f.hour(h), v]) }}
        >
          <Columns labels={hours.map((_, h) => f.hour(h))} values={hours} emphasis={peak} xEvery={6} height={170} name={t('app.userPage.clock.views')} />
        </ChartFrame>
      </div>

      <div className={l.g2}>
        <Card>
          <CardHeader title={t('app.userPage.profile.title')} subtitle={t('app.userPage.profile.subtitle')} />
          <CardBody>
            <div className={s.profile}>
              {scores.map(([k, v]) => (
                <div key={k} className={s.score}>
                  <div className={s.scoreHead}><span>{scoreLabel(k)}</span><b className="num">{v === null ? '—' : f.percent(v)}</b></div>
                  <Meter value={v ?? 0} label={scoreLabel(k)} />
                  <div className={s.scoreSub}>{k === 'completion' && showPte ? <FinishedFoot data={user.playedToEnd} /> : t(`app.userPage.profile.${k}Hint`)}</div>
                </div>
              ))}
            </div>
          </CardBody>
        </Card>
        <StreakCard user={user} />
      </div>

      <div className={l.g2}>
        <Card>
          <CardHeader title={t('app.userPage.devices.title')} subtitle={t('app.userPage.devices.subtitle')} />
          <CardBody flush>
            <DataTable dense columns={deviceCols} data={user.devices ?? []} empty={<EmptyState title={t('app.userPage.devices.empty')} />} />
          </CardBody>
        </Card>
        <ComparePanel user={user} />
      </div>

      <div className={l.g2}><LanguagesCard userId={user.id} /><LoginsCard userId={user.id} /></div>
    </div>
  )
}

