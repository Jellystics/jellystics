import { useTranslation } from 'react-i18next'
import { Columns } from './Columns'

export function StopHistogram({ bins, runtimeMinutes, height = 160 }: { bins: number[]; runtimeMinutes: number; height?: number }) {
  const { t } = useTranslation()
  const labels = bins.map((_, i) => `${Math.round((i * runtimeMinutes) / bins.length)} min`)
  return (
    <Columns
      labels={labels} values={bins} name={t('app.chart.stops')} height={height}
      xEvery={Math.max(1, Math.ceil(bins.length / 6))} emphasis={bins.length - 1}
    />
  )
}
