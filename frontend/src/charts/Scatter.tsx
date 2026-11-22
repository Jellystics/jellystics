import { useChartTooltip } from './ChartTooltip'
import { useDefaultFormat } from './useDefaultFormat'
import { useChartWidth } from './useChartWidth'
import { niceMax, ticks as makeTicks, clamp01 } from './scale'
import styles from './charts.module.css'

export function Scatter({ points, xLabel, yLabel, xFormat, yFormat, note, height = 240 }: {
  points: { x: number; y: number; label: string; color?: string }[]
  xLabel: string
  yLabel: string
  xFormat?: (v: number) => string
  yFormat?: (v: number) => string
  note?: string
  height?: number
}) {
  const dflt = useDefaultFormat()
  const xf = xFormat ?? dflt, yf = yFormat ?? dflt
  const [ref, width] = useChartWidth()
  const tip = useChartTooltip()
  const m = { t: 16, r: 12, b: 34, l: 46 }
  const iw = width - m.l - m.r, ih = height - m.t - m.b
  const xmax = niceMax(Math.max(0, ...points.map((p) => p.x))), ymax = niceMax(Math.max(0, ...points.map((p) => p.y)))
  const x = (v: number) => m.l + (v / xmax) * iw, y = (v: number) => m.t + ih - (clamp01(v / ymax)) * ih

  return (
    <div className={styles.chart} ref={ref}>
      <svg width={width} height={height} role="img" aria-label={`${yLabel} / ${xLabel}`}>
        {makeTicks(ymax, 4).map((v) => (
          <g key={v}>
            <line x1={m.l} x2={width - m.r} y1={y(v)} y2={y(v)} stroke="var(--grid)" />
            <text x={m.l - 8} y={y(v) + 3.5} textAnchor="end">{yf(v)}</text>
          </g>
        ))}
        {makeTicks(xmax, 4).map((v) => <text key={v} x={x(v)} y={height - 18} textAnchor="middle">{xf(v)}</text>)}
        <text x={m.l + iw / 2} y={height - 3} textAnchor="middle">{xLabel}</text>
        <text x={m.l} y={m.t - 4}>{yLabel}</text>
        {note && <text className={styles.strong} x={width - m.r - 4} y={y(0) - 6} textAnchor="end">{note}</text>}
        {points.map((p, i) => (
          <circle
            key={i} cx={x(p.x)} cy={y(p.y)} r={5} fill={p.color ?? 'var(--accent)'} stroke="var(--surface)" strokeWidth={2}
            {...tip.hit(p.label, [{ label: xLabel, value: xf(p.x) }, { label: yLabel, value: yf(p.y) }])}
          />
        ))}
      </svg>
      {tip.node}
    </div>
  )
}
