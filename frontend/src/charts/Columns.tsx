import { useChartTooltip } from './ChartTooltip'
import { useDefaultFormat } from './useDefaultFormat'
import { useChartWidth } from './useChartWidth'
import { barPath, niceMax, ticks as makeTicks, clamp01 } from './scale'
import styles from './charts.module.css'

export function Columns({ labels, values, name, color = 'var(--accent)', emphasis = -1, height = 180, yFormat: yFormatProp, xEvery = 1, showXLabel, ticks = true, onBarClick }: {
  labels: string[]
  values: number[]
  name: string
  color?: string
  emphasis?: number
  height?: number
  yFormat?: (v: number) => string
  xEvery?: number
  /** Decides per index whether its x label is drawn; takes precedence over xEvery. */
  showXLabel?: (i: number) => boolean
  ticks?: boolean
  onBarClick?: (i: number) => void
}) {
  const defaultFmt = useDefaultFormat()
  const yFormat = yFormatProp ?? defaultFmt
  const [ref, width] = useChartWidth()
  const tip = useChartTooltip()
  const m = { t: 18, r: 8, b: 24, l: ticks ? 40 : 8 }
  const iw = width - m.l - m.r, ih = height - m.t - m.b
  const ymax = niceMax(Math.max(0, ...values) * 1.05)
  const band = iw / Math.max(1, values.length), bw = Math.min(24, band - 2)
  const y = (v: number) => m.t + ih - (clamp01(v / ymax)) * ih
  const hasEmphasis = emphasis >= 0

  return (
    <div className={styles.chart} ref={ref}>
      <svg width={width} height={height} role="img" aria-label={name}>
        {ticks && makeTicks(ymax).map((v) => (
          <g key={v}>
            <line x1={m.l} x2={width - m.r} y1={y(v)} y2={y(v)} stroke="var(--grid)" />
            <text x={m.l - 8} y={y(v) + 3.5} textAnchor="end">{yFormat(v)}</text>
          </g>
        ))}
        {values.map((v, i) => {
          const cx = m.l + band * i + band / 2
          const c = hasEmphasis && i !== emphasis ? `color-mix(in srgb, ${color} 35%, var(--surface-3))` : color
          return (
            <g key={i}>
              <path d={barPath(cx - bw / 2, y(v), bw, y(0) - y(v))} fill={c} />
              {i === emphasis && <text className={styles.strong} x={cx} y={y(v) - 6} textAnchor="middle">{yFormat(v)}</text>}
              {(showXLabel ? showXLabel(i) : i % xEvery === 0) && <text x={cx} y={height - 6} textAnchor="middle">{labels[i]}</text>}
              <rect
                x={m.l + band * i} y={m.t} width={band} height={ih} fill="transparent"
                {...tip.hit(labels[i], [{ color: c, label: name, value: yFormat(v) }], undefined, onBarClick ? () => onBarClick(i) : undefined)}
              />
            </g>
          )
        })}
        <line x1={m.l} x2={width - m.r} y1={y(0)} y2={y(0)} stroke="var(--axis)" />
      </svg>
      {tip.node}
    </div>
  )
}
