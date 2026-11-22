import { useState } from 'react'
import { useChartTooltip } from './ChartTooltip'
import { useDefaultFormat } from './useDefaultFormat'
import { useChartWidth } from './useChartWidth'
import { linePath, niceMax, showXLabel, ticks } from './scale'
import styles from './charts.module.css'

export function LineChart({ labels, series, compare, area = true, height = 220, yFormat: yFormatProp, xEvery = 5, onPointClick }: {
  labels: string[]
  series: { name: string; color: string; values: number[] }[]
  compare?: { name: string; values: number[] }
  area?: boolean
  height?: number
  yFormat?: (v: number) => string
  xEvery?: number
  onPointClick?: (index: number) => void
}) {
  const defaultFmt = useDefaultFormat()
  const yFormat = yFormatProp ?? defaultFmt
  const [ref, width] = useChartWidth()
  const tip = useChartTooltip()
  const [hover, setHover] = useState<number | null>(null)
  const m = { t: 10, r: 12, b: 26, l: 46 }
  const iw = width - m.l - m.r, ih = height - m.t - m.b
  const all = series.flatMap((s) => s.values).concat(compare?.values ?? [])
  const ymax = niceMax(Math.max(0, ...all) * 1.05)
  const x = (i: number) => m.l + (labels.length <= 1 ? iw / 2 : (i * iw) / (labels.length - 1))
  const y = (v: number) => m.t + ih - (v / ymax) * ih
  const pts = (vals: number[]) => vals.map((v, i) => [x(i), y(v)] as [number, number])
  const step = iw / Math.max(1, labels.length - 1)
  const name = series.map((s) => s.name).join(', ')

  return (
    <div className={styles.chart} ref={ref}>
      <svg width={width} height={height} role="img" aria-label={name} onMouseLeave={() => setHover(null)}>
        {ticks(ymax).map((v) => (
          <g key={v}>
            <line x1={m.l} x2={width - m.r} y1={y(v)} y2={y(v)} stroke={v === 0 ? 'var(--axis)' : 'var(--grid)'} />
            <text x={m.l - 8} y={y(v) + 3.5} textAnchor="end">{yFormat(v)}</text>
          </g>
        ))}
        {labels.map((l, i) => showXLabel(i, labels.length, xEvery) && <text key={i} x={x(i)} y={height - 8} textAnchor="middle">{l}</text>)}
        {compare && <path d={linePath(pts(compare.values))} fill="none" stroke="var(--axis)" strokeWidth={1.5} />}
        {series.map((s) => {
          const p = pts(s.values)
          const last = p[p.length - 1]
          return (
            <g key={s.name}>
              {area && p.length > 0 && <path d={`${linePath(p)}L${last[0]},${y(0)}L${p[0][0]},${y(0)}Z`} fill={s.color} opacity={0.1} />}
              <path d={linePath(p)} fill="none" stroke={s.color} strokeWidth={2} strokeLinejoin="round" strokeLinecap="round" />
              {hover !== null && p[hover] && <circle cx={p[hover][0]} cy={p[hover][1]} r={4} fill={s.color} stroke="var(--surface)" strokeWidth={2} />}
              {last && <circle cx={last[0]} cy={last[1]} r={4} fill={s.color} stroke="var(--surface)" strokeWidth={2} />}
            </g>
          )
        })}
        {hover !== null && <line x1={x(hover)} x2={x(hover)} y1={m.t} y2={y(0)} stroke="var(--axis)" />}
        {labels.map((l, i) => {
          const rows = series.map((s) => ({ color: s.color, label: s.name, value: yFormat(s.values[i]) }))
          if (compare) rows.push({ color: 'var(--axis)', label: compare.name, value: yFormat(compare.values[i]) })
          return (
            <rect
              key={i} x={x(i) - step / 2} y={m.t} width={step} height={ih} fill="transparent" aria-label={l}
              style={onPointClick ? { cursor: 'pointer' } : undefined}
              onClick={onPointClick ? () => onPointClick(i) : undefined}
              {...tip.hit(l, rows, () => setHover(i))}
              onMouseLeave={() => { tip.hide(); setHover(null) }}
              onBlur={() => { tip.hide(); setHover(null) }}
            />
          )
        })}
      </svg>
      {tip.node}
    </div>
  )
}
