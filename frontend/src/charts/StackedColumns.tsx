import { useTranslation } from 'react-i18next'
import { useChartTooltip } from './ChartTooltip'
import { useDefaultFormat } from './useDefaultFormat'
import { useChartWidth } from './useChartWidth'
import { barPath, niceMax, showXLabel, ticks } from './scale'
import styles from './charts.module.css'

export function StackedColumns({ labels, series, height = 220, yFormat: yFormatProp, xEvery = 5 }: {
  labels: string[]
  series: { name: string; color: string; values: number[] }[]
  height?: number
  yFormat?: (v: number) => string
  xEvery?: number
}) {
  const { t } = useTranslation()
  const defaultFmt = useDefaultFormat()
  const yFormat = yFormatProp ?? defaultFmt
  const [ref, width] = useChartWidth()
  const tip = useChartTooltip()
  const m = { t: 10, r: 8, b: 26, l: 46 }
  const iw = width - m.l - m.r, ih = height - m.t - m.b
  const totals = labels.map((_, i) => series.reduce((a, s) => a + (s.values[i] ?? 0), 0))
  const ymax = niceMax(Math.max(0, ...totals) * 1.05)
  const band = iw / Math.max(1, labels.length), bw = Math.min(24, band - 2)
  const y = (v: number) => m.t + ih - (v / ymax) * ih

  return (
    <div className={styles.chart} ref={ref}>
      <svg width={width} height={height} role="img" aria-label={series.map((s) => s.name).join(', ')}>
        {ticks(ymax).map((v) => (
          <g key={v}>
            <line x1={m.l} x2={width - m.r} y1={y(v)} y2={y(v)} stroke="var(--grid)" />
            <text x={m.l - 8} y={y(v) + 3.5} textAnchor="end">{yFormat(v)}</text>
          </g>
        ))}
        {labels.map((l, i) => {
          const cx = m.l + band * i + band / 2, x0 = cx - bw / 2
          let acc = 0
          // topmost non-empty series gets the rounded end
          let topIdx = -1
          series.forEach((s, si) => { if (s.values[i]) topIdx = si })
          return (
            <g key={i}>
              {series.map((s, si) => {
                const v = s.values[i]
                if (!v) return null
                const y1 = y(acc + v), h = y(acc) - y1 - (acc > 0 ? 2 : 0)
                acc += v
                return si === topIdx
                  ? <path key={s.name} d={barPath(x0, y1, bw, h)} fill={s.color} />
                  : <rect key={s.name} x={x0} y={y1} width={bw} height={Math.max(0, h)} fill={s.color} />
              })}
              <rect
                x={m.l + band * i} y={m.t} width={band} height={ih} fill="transparent" aria-label={l}
                {...tip.hit(l, [
                  ...series.map((s) => ({ color: s.color, label: s.name, value: yFormat(s.values[i] ?? 0) })),
                  { color: null, label: t('app.chart.total'), value: yFormat(totals[i]) },
                ])}
              />
              {showXLabel(i, labels.length, xEvery) && <text x={cx} y={height - 8} textAnchor="middle">{l}</text>}
            </g>
          )
        })}
        <line x1={m.l} x2={width - m.r} y1={y(0)} y2={y(0)} stroke="var(--axis)" />
      </svg>
      {tip.node}
    </div>
  )
}
