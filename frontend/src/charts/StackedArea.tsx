import { useTranslation } from 'react-i18next'
import { useChartTooltip } from './ChartTooltip'
import { useDefaultFormat } from './useDefaultFormat'
import { useChartWidth } from './useChartWidth'
import { niceMax, showXLabel, ticks as makeTicks } from './scale'
import styles from './charts.module.css'

export function StackedArea({ labels, series, yFormat, height = 220 }: {
  labels: string[]
  series: { name: string; color: string; values: number[] }[]
  yFormat?: (v: number) => string
  height?: number
}) {
  const { t } = useTranslation()
  const dflt = useDefaultFormat()
  const yf = yFormat ?? dflt
  const [ref, width] = useChartWidth()
  const tip = useChartTooltip()
  const m = { t: 10, r: 12, b: 26, l: 46 }
  const n = labels.length
  const iw = width - m.l - m.r, ih = height - m.t - m.b
  const totals = labels.map((_, i) => series.reduce((a, s) => a + (s.values[i] ?? 0), 0))
  const ymax = niceMax(Math.max(0, ...totals) * 1.05)
  const step = iw / Math.max(1, n - 1)
  const x = (i: number) => m.l + i * step, y = (v: number) => m.t + ih - (v / ymax) * ih
  const every = Math.max(1, Math.ceil(n / Math.max(1, Math.floor(iw / 60))))
  const acc = labels.map(() => 0)
  const areas = series.map((s) => {
    const lower = acc.slice()
    labels.forEach((_, i) => { acc[i] += s.values[i] ?? 0 })
    const top = acc.map((v, i) => `${x(i)},${y(v)}`), bot = lower.map((v, i) => `${x(i)},${y(v)}`).reverse()
    return { s, d: n ? `M${top.join('L')}L${bot.join('L')}Z` : '' }
  })

  return (
    <div className={styles.chart} ref={ref}>
      <svg width={width} height={height} role="img" aria-label={series.map((s) => s.name).join(', ') || t('app.chart.stackedArea')}>
        {makeTicks(ymax, 4).map((v) => (
          <g key={v}>
            <line x1={m.l} x2={width - m.r} y1={y(v)} y2={y(v)} stroke="var(--grid)" />
            <text x={m.l - 8} y={y(v) + 3.5} textAnchor="end">{yf(v)}</text>
          </g>
        ))}
        {areas.map(({ s, d }) => <path key={s.name} d={d} fill={s.color} opacity={0.85} stroke="var(--surface)" strokeWidth={1.5} />)}
        {labels.map((l, i) => showXLabel(i, n, every) && <text key={i} x={x(i)} y={height - 8} textAnchor="middle">{l}</text>)}
        {labels.map((l, i) => (
          <g key={i} className={styles.hz}>
            <line className={styles.xhair} x1={x(i)} x2={x(i)} y1={m.t} y2={y(0)} stroke="var(--ink-3)" />
            <rect
              x={x(i) - step / 2} y={m.t} width={step} height={ih} fill="transparent"
              {...tip.hit(l, series.map((s) => ({ color: s.color, label: s.name, value: yf(s.values[i] ?? 0) })).reverse())}
            />
          </g>
        ))}
      </svg>
      {tip.node}
    </div>
  )
}
