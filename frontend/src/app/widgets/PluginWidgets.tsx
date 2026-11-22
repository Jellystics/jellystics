// Widgets fed by the precise data of the Jellystics plugin. Each one hides
// itself when the plugin has no data for the period (never a row of zeros)
// and says "data collected from …" when the plugin started inside it.
import type { ColumnDef } from '@tanstack/react-table'
import { useTranslation } from 'react-i18next'
import { Info } from 'lucide-react'
import type { PlayedToEnd, SkipStats } from '@/api'
import { useUserLogins, type ItemMarks, type LoginDevice } from '@/api/plugin'
import { useFormat } from '@/lib/format'
import { Card, CardBody, CardHeader, DataTable, EmptyState, SegBar, StatStrip, StatTile, Tooltip } from '@/ui'
import { usePluginPartial } from './plugin'
import s from './PluginWidgets.module.css'

/** "Data collected from …" when the plugin data starts after the period start. */
export function PluginSinceNote({ since }: { since: string | null | undefined }) {
  const { t } = useTranslation()
  const f = useFormat()
  const partial = usePluginPartial(since)
  if (!partial || !since) return null
  return <p className={s.note}>{t('app.analytics.collectedSince', { date: f.day(new Date(`${since}T12:00:00`)) })}</p>
}

/**
 * "Played to the end" tile: share of closed views Jellyfin reported as played
 * to completion; views without that report (history, session tracking) are
 * estimated from the stop position, and the foot says how many.
 */
export function PlayedToEndTile({ data }: { data: PlayedToEnd }) {
  const { t } = useTranslation()
  const f = useFormat()
  const estimated = data.plays - data.measured
  const foot = estimated > 0
    ? t('app.plugin.playedToEnd.footMixed', { measured: f.number(data.measured), estimated: f.number(estimated) })
    : t('app.plugin.playedToEnd.footMeasured', { count: data.measured })
  return (
    <StatTile
      label={t('app.plugin.playedToEnd.label')}
      value={data.ratio == null ? '—' : f.percent(data.ratio)}
      foot={
        <Tooltip content={t('app.plugin.playedToEnd.tip')}>
          <span tabIndex={0} className={s.tipFoot}>{foot}<Info size={12} aria-hidden="true" /></span>
        </Tooltip>
      }
    />
  )
}

/** Skipped time to the second (seconds matter for an intro skip). */
const skipped = (sec: number) => {
  const n = Math.round(sec)
  return n < 60 ? `${n} s` : `${Math.floor(n / 60)} min ${String(n % 60).padStart(2, '0')} s`
}

/** Forward skips / intro skipping of closed plugin plays (episode, series). Hidden without measured plays. */
export function SkipsCard({ skips, since }: { skips: SkipStats | undefined; since: string | null | undefined }) {
  const { t } = useTranslation()
  const f = useFormat()
  if (!skips || skips.plays === 0) return null
  return (
    <Card>
      <CardHeader title={t('app.plugin.skips.title')} subtitle={t('app.plugin.skips.plays', { count: skips.plays })} />
      <StatStrip columns={3}>
        <StatTile label={t('app.plugin.skips.intro')} value={skips.introRate == null ? '—' : f.percent(skips.introRate)} foot={t('app.plugin.skips.introFoot', { count: skips.intro })} />
        <StatTile label={t('app.plugin.skips.skipRate')} value={skips.skipRate == null ? '—' : f.percent(skips.skipRate)} foot={t('app.plugin.skips.skipFoot', { count: skips.skipped })} />
        <StatTile label={t('app.plugin.skips.avgSkipped')} value={skips.avgSkippedSeconds == null ? '—' : skipped(skips.avgSkippedSeconds)} foot={t('app.plugin.skips.avgSkippedFoot')} />
      </StatStrip>
      <CardBody>
        <p className={s.note}>{t('app.plugin.skips.estimate')}</p>
        <PluginSinceNote since={since} />
      </CardBody>
    </Card>
  )
}

/** Manual vs automatic "played" marks of a movie / episode (current state, useItemMarks). Render only when hasMarks. */
export function MarksCard({ data: d }: { data: ItemMarks }) {
  const { t } = useTranslation()
  const f = useFormat()
  const marked = d.manual + d.automatic + d.other
  return (
    <Card>
      <CardHeader title={t('app.plugin.marks.title')} subtitle={t('app.plugin.marks.subtitle')} />
      <CardBody>
        {marked > 0 && (
          <SegBar parts={[
            { label: t('app.plugin.marks.manualLabel'), value: d.manual, color: 'var(--accent)', display: f.number(d.manual) },
            { label: t('app.plugin.marks.automaticLabel'), value: d.automatic, color: 'var(--t-music)', display: f.number(d.automatic) },
            { label: t('app.plugin.marks.otherLabel'), value: d.other, color: 'var(--seq-2)', display: f.number(d.other) },
          ].filter((p) => p.value > 0)} />
        )}
        <ul className={s.facts}>
          <li>
            <b>{t('app.plugin.marks.manual', { count: d.manual })}</b>
            {d.manualWithoutView > 0 && <> · {t('app.plugin.marks.manualWithoutView', { count: d.manualWithoutView })}</>}
          </li>
          <li>{t('app.plugin.marks.automatic', { count: d.automatic })}</li>
          {d.other > 0 && <li>{t('app.plugin.marks.other', { count: d.other })}</li>}
          {d.unplayed > 0 && <li>{t('app.plugin.marks.unplayed', { count: d.unplayed })}</li>}
          <li>{t('app.plugin.marks.watched', { count: d.watched })}</li>
        </ul>
        <p className={s.note}>{t('app.plugin.marks.since', { date: f.day(new Date(`${d.collectedSince}T12:00:00`)) })}</p>
      </CardBody>
    </Card>
  )
}

/** Logins of a user (plugin): count, last login, devices, failed attempts (admins). Hidden before any login was collected. */
export function LoginsCard({ userId }: { userId: string }) {
  const { t } = useTranslation()
  const f = useFormat()
  const d = useUserLogins(userId).data
  const partial = usePluginPartial(d?.collectedSince)
  if (!d?.collectedSince) return null
  const cols: ColumnDef<LoginDevice>[] = [
    { id: 'device', header: t('app.userPage.devices.device'), cell: ({ row: { original: r } }) => r.device || '—' },
    { id: 'client', header: t('app.userPage.devices.client'), cell: ({ row: { original: r } }) => <span className="ink2">{[r.client, r.version].filter(Boolean).join(' ') || '—'}</span> },
    { id: 'count', header: t('app.plugin.logins.count'), meta: { numeric: true }, cell: ({ row: { original: r } }) => f.number(r.count) },
    { id: 'last', header: t('app.plugin.logins.lastAt'), cell: ({ row: { original: r } }) => <span className="num muted">{f.relative(new Date(r.lastAt))}</span> },
  ]
  return (
    <Card>
      <CardHeader title={t('app.plugin.logins.title')} subtitle={t('app.plugin.logins.subtitle')} />
      <StatStrip columns={d.failed == null ? 2 : 3}>
        <StatTile label={t('app.plugin.logins.count')} value={f.number(d.count)} />
        <StatTile label={t('app.plugin.logins.lastLogin')} value={d.lastLogin ? f.relative(new Date(d.lastLogin)) : t('app.plugin.logins.never')} />
        {d.failed != null && <StatTile label={t('app.plugin.logins.failed')} value={f.number(d.failed)} foot={t('app.plugin.logins.failedFoot')} />}
      </StatStrip>
      <CardBody flush>
        <DataTable dense columns={cols} data={d.devices} empty={<EmptyState title={t('app.plugin.logins.empty')} />} />
      </CardBody>
      {partial && <CardBody><PluginSinceNote since={d.collectedSince} /></CardBody>}
    </Card>
  )
}
