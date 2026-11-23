import { useMemo, useRef, useState, type KeyboardEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { useChartTooltip, type TipRow } from './ChartTooltip'
import { useDefaultFormat } from './useDefaultFormat'
import { useChartWidth } from './useChartWidth'
import { niceMax, ticks as makeTicks, clamp01 } from './scale'
import { formatDateTime, timeTicks } from './timeScale'
import styles from './charts.module.css'

export type ScatterPoint = {
  x: number
  y: number
  label: string
  color?: string
  /** Tooltip rows; default: the x and y values. */
  rows?: TipRow[]
}

/**
 * Dot plot. By default x runs 0..max; with `timeAxis` x is an epoch-ms
 * instant, the domain runs min..max (or `xDomain`) and ticks are dates in
 * `timeZone` (the server's display zone). Colours come per point, so several
 * series share one plot (legend: ChartFrame). With `onPointClick` the points
 * are buttons; the keyboard reaches them through one tab stop and the arrow keys.
 */
export function Scatter({ points, xLabel, yLabel, xFormat, yFormat, note, height = 240, timeAxis = false, timeZone, xDomain, radius = 5, onPointClick }: {
  points: ScatterPoint[]
  xLabel: string
  yLabel: string
  xFormat?: (v: number) => string
  yFormat?: (v: number) => string
  note?: string
  height?: number
  timeAxis?: boolean
  timeZone?: string
  xDomain?: [number, number]
  radius?: number
  onPointClick?: (index: number) => void
}) {
  const { i18n } = useTranslation()
  const locale = i18n.resolvedLanguage || 'fr'
  const dflt = useDefaultFormat()
  const xf = xFormat ?? (timeAxis ? (v: number) => formatDateTime(v, locale, timeZone) : dflt)
  const yf = yFormat ?? dflt
  const [ref, width] = useChartWidth()
  const tip = useChartTooltip()
  const svgRef = useRef<SVGSVGElement>(null)
  const m = { t: 16, r: 12 + (timeAxis ? radius : 0), b: 34, l: 46 }
  const iw = Math.max(1, width - m.l - m.r), ih = height - m.t - m.b

  const [x0, x1] = useMemo((): [number, number] => {
    if (xDomain) return xDomain
    if (!timeAxis) return [0, niceMax(Math.max(0, ...points.map((p) => p.x)))]
    if (points.length === 0) return [0, 1]
    let lo = Infinity, hi = -Infinity
    for (const p of points) { if (p.x < lo) lo = p.x; if (p.x > hi) hi = p.x }
    if (lo === hi) return [lo - 43_200_000, hi + 43_200_000]
    return [lo, hi]
  }, [points, timeAxis, xDomain])
  const ymax = niceMax(Math.max(0, ...points.map((p) => p.y)))
  const x = (v: number) => m.l + clamp01((v - x0) / (x1 - x0 || 1)) * iw
  const y = (v: number) => m.t + ih - (clamp01(v / ymax)) * ih

  const timeTickList = useMemo(() => {
    if (!timeAxis) return null
    const tt = timeTicks(x0, x1, Math.max(2, Math.floor(iw / 84)), locale, timeZone)
    return tt.ticks.map((v) => ({ v, label: tt.format(v) }))
  }, [timeAxis, x0, x1, iw, locale, timeZone])
  const xTicks = timeTickList ?? makeTicks(x1, 4).map((v) => ({ v, label: xf(v) }))

  // Roving focus in x order: one tab stop, arrows move between points.
  const order = useMemo(() => points.map((_, i) => i).sort((a, b) => points[a].x - points[b].x || a - b), [points])
  const [focusIdx, setFocusIdx] = useState<number | null>(null)
  const current = focusIdx !== null && focusIdx < points.length ? focusIdx : order[0]
  const moveFocus = (from: number, delta: number) => {
    const pos = order.indexOf(from)
    const next = order[Math.min(order.length - 1, Math.max(0, pos + delta))]
    if (next === undefined || next === from) return
    setFocusIdx(next)
    svgRef.current?.querySelector<SVGElement>(`[data-pt="${next}"]`)?.focus()
  }
  const keys: Record<string, number> = { ArrowRight: 1, ArrowDown: 1, ArrowLeft: -1, ArrowUp: -1, PageDown: 10, PageUp: -10 }

  return (
    <div className={styles.chart} ref={ref}>
      <svg ref={svgRef} width={width} height={height} role="img" aria-label={`${yLabel} / ${xLabel}`}>
        {makeTicks(ymax, 4).map((v) => (
          <g key={v}>
            <line x1={m.l} x2={width - m.r} y1={y(v)} y2={y(v)} stroke={v === 0 ? 'var(--axis)' : 'var(--grid)'} />
            <text x={m.l - 8} y={y(v) + 3.5} textAnchor="end">{yf(v)}</text>
          </g>
        ))}
        {xTicks.map(({ v, label }, i) => {
          // Keep edge labels inside the plot.
          const anchor = timeAxis && x(v) - m.l < 24 ? 'start' : timeAxis && m.l + iw - x(v) < 24 ? 'end' : 'middle'
          return <text key={`${v}-${i}`} x={x(v)} y={height - 18} textAnchor={anchor}>{label}</text>
        })}
        <text x={m.l + iw / 2} y={height - 3} textAnchor="middle">{xLabel}</text>
        <text x={m.l} y={m.t - 4}>{yLabel}</text>
        {note && <text className={styles.strong} x={width - m.r - 4} y={y(0) - 6} textAnchor="end">{note}</text>}
        {points.map((p, i) => {
          const hit = tip.hit(p.label, p.rows ?? [{ label: xLabel, value: xf(p.x) }, { label: yLabel, value: yf(p.y) }], undefined, onPointClick ? () => onPointClick(i) : undefined)
          const onKeyDown = (e: KeyboardEvent) => {
            const d = keys[e.key]
            if (d !== undefined) { e.preventDefault(); moveFocus(i, d); return }
            if (e.key === 'Home' || e.key === 'End') { e.preventDefault(); moveFocus(i, e.key === 'Home' ? -order.length : order.length); return }
            hit.onKeyDown?.(e)
          }
          return (
            <circle
              key={i} data-pt={i} cx={x(p.x)} cy={y(p.y)} r={radius} fill={p.color ?? 'var(--accent)'}
              stroke="var(--surface)" strokeWidth={radius >= 5 ? 2 : 1}
              {...hit}
              tabIndex={i === current ? 0 : -1}
              onKeyDown={onKeyDown}
              onFocus={(e) => { setFocusIdx(i); hit.onFocus(e) }}
            />
          )
        })}
      </svg>
      {tip.node}
    </div>
  )
}
