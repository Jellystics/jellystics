import { useTranslation } from 'react-i18next'
import { ChartFrame, Funnel, StopHistogram } from '@/charts'
import { useFormat } from '@/lib/format'
import type { Funnel as FunnelData } from '@/api'

/** Clicks → views → half → completed funnel in a ChartFrame. */
export function FunnelCard({ funnel, title, subtitle, csvName }: { funnel: FunnelData; title: string; subtitle?: string; csvName: string }) {
  const { t } = useTranslation()
  const f = useFormat()
  const stages = [
    { label: t('app.entity.funnel.clicks'), value: funnel.clicks },
    { label: t('app.entity.funnel.views'), value: funnel.views },
    { label: t('app.entity.funnel.half'), value: funnel.half },
    // Finished views: the same metric as the "finished views" tile, read against views.
    {
      label: t('app.entity.funnel.completed'), value: funnel.completed,
      note: funnel.views > 0 ? t('app.plugin.funnel.ofViews', { pct: f.percent(funnel.completed / funnel.views), done: f.number(funnel.completed), views: f.number(funnel.views) }) : undefined,
    },
  ]
  return (
    <ChartFrame title={title} subtitle={subtitle} csvName={csvName} table={{ columns: [t('app.entity.funnel.stage'), t('app.entity.col.views')], rows: stages.map((s) => [s.label, s.value]) }}>
      <Funnel stages={stages} />
    </ChartFrame>
  )
}

/** Stop positions histogram (bins over the runtime) in a ChartFrame. */
export function StopsCard({ bins, runtimeSeconds, title, subtitle, csvName }: { bins: number[]; runtimeSeconds: number; title: string; subtitle?: string; csvName: string }) {
  const { t } = useTranslation()
  const f = useFormat()
  const minutes = Math.max(1, Math.round(runtimeSeconds / 60))
  const binLabel = (i: number) => `${f.position(Math.round((i * runtimeSeconds) / bins.length))}–${f.position(Math.round(((i + 1) * runtimeSeconds) / bins.length))}`
  return (
    <ChartFrame title={title} subtitle={subtitle} csvName={csvName} table={{ columns: [t('app.entity.stops.slice'), t('app.entity.stops.count')], rows: bins.map((v, i) => [binLabel(i), v]) }}>
      <StopHistogram bins={bins} runtimeMinutes={minutes} height={150} />
    </ChartFrame>
  )
}
