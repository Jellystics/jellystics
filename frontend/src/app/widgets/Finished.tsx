// "Finished views" figure shared by the admin entity pages and the « Mes statistiques » page.
// Kept apart from PluginWidgets so a consumer does not pull the plugin data hooks.
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Info } from 'lucide-react'
import type { PlayedToEnd } from '@/api/types'
import { useFormat } from '@/lib/format'
import { StatTile } from '@/ui/StatTile'
import { Tooltip } from '@/ui/Tooltip'
import s from './PluginWidgets.module.css'

/** Foot of a finished-views figure: how many units the plugin measured (Jellyfin end-of-playback report) vs estimated from the stop position. */
export function FinishedFoot({ data }: { data: PlayedToEnd }) {
  const { t } = useTranslation()
  const f = useFormat()
  const estimated = data.plays - data.measured
  const foot = estimated > 0
    ? t('app.plugin.playedToEnd.footMixed', { measured: f.number(data.measured), estimated: f.number(estimated) })
    : t('app.plugin.playedToEnd.footMeasured', { count: data.measured })
  return (
    <Tooltip content={t('app.plugin.playedToEnd.tip')}>
      <span tabIndex={0} className={s.tipFoot}>{foot}<Info size={12} aria-hidden="true" /></span>
    </Tooltip>
  )
}

/**
 * The "finished views" tile of an entity page: (user, item, day) units
 * finished per Jellyfin's report when the plugin measured it, else per the
 * stop position (>= 90 %). Same count as the funnel's last step.
 */
export function FinishedTile({ data, foot }: { data: PlayedToEnd; foot?: ReactNode }) {
  const { t } = useTranslation()
  const f = useFormat()
  return (
    <StatTile
      label={t('app.metric.finishedViews')}
      value={data.ratio == null ? '—' : f.percent(data.ratio)}
      foot={data.measured > 0 ? <FinishedFoot data={data} /> : foot}
    />
  )
}
