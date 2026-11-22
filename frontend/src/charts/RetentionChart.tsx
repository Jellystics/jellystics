import { useTranslation } from 'react-i18next'
import { useFormat } from '@/lib/format'
import { useChartTooltip } from './ChartTooltip'
import { useChartWidth } from './useChartWidth'
import { linePath, niceMax, ticks as makeTicks } from './scale'
import styles from './charts.module.css'

export function RetentionChart({ seasons, seasonNumbers, onEpisodeClick, height = 200 }: {
  seasons: number[][]
  /** Display number of each season (labels); defaults to 1, 2, 3… */
  seasonNumbers?: number[]
  /** 1-based season index and episode number. */
  onEpisodeClick?: (season: number, episode: number) => void
  height?: number
}) {
  const { t } = useTranslation()
  const f = useFormat()
  const [ref, width] = useChartWidth()
  const tip = useChartTooltip()
  const m = { t: 18, r: 12, b: 28, l: 40 }
  const pts = seasons.flatMap((s, si) => s.map((v, ei) => ({ v, si, ei })))
  const n = pts.length
  const iw = width - m.l - m.r, ih = height - m.t - m.b
  const ymax = niceMax(Math.max(0, ...pts.map((p) => p.v)))
  const step = iw / Math.max(1, n - 1)
  const x = (i: number) => m.l + i * step, y = (v: number) => m.t + ih - (v / ymax) * ih
  const base = pts[0]?.v || 1
  const color = 'var(--t-series)'
  const d = linePath(pts.map((p, i) => [x(i), y(p.v)]))
  const last = n - 1
  let idx = 0
  const sn = (si: number) => seasonNumbers?.[si] ?? si + 1

  return (
    <div className={styles.chart} ref={ref}>
      <svg width={width} height={height} role="img" aria-label={t('app.chart.retention')}>
        {makeTicks(ymax, 3).map((v) => (
          <g key={v}>
            <line x1={m.l} x2={width - m.r} y1={y(v)} y2={y(v)} stroke="var(--grid)" />
            <text x={m.l - 8} y={y(v) + 3.5} textAnchor="end">{f.compact(v)}</text>
          </g>
        ))}
        {seasons.map((s, si) => {
          const x0 = x(idx), x1 = x(idx + s.length - 1)
          const el = (
            <g key={si}>
              {si > 0 && <line x1={x0 - step / 2} x2={x0 - step / 2} y1={m.t} y2={y(0)} stroke="var(--axis)" />}
              <text x={(x0 + x1) / 2} y={height - 8} textAnchor="middle">{`${t('app.chart.season')} ${sn(si)}`}</text>
            </g>
          )
          idx += s.length
          return el
        })}
        {n > 0 && (
          <>
            <path d={`${d}L${x(last)},${y(0)}L${x(0)},${y(0)}Z`} fill={color} opacity={0.1} />
            <path d={d} fill="none" stroke={color} strokeWidth={2} strokeLinejoin="round" />
          </>
        )}
        {pts.map((p, i) => (
          <g key={i} className={styles.hz}>
            <line className={styles.xhair} x1={x(i)} x2={x(i)} y1={m.t} y2={y(0)} stroke="var(--axis)" />
            <circle className={styles.pt} cx={x(i)} cy={y(p.v)} r={4} fill={color} stroke="var(--surface)" strokeWidth={2} />
            <rect
              x={x(i) - step / 2} y={m.t} width={step} height={ih} fill="transparent"
              aria-label={t('app.chart.episodeShort', { season: sn(p.si), episode: p.ei + 1 })}
              style={onEpisodeClick ? { cursor: 'pointer' } : undefined}
              onClick={onEpisodeClick ? () => onEpisodeClick(p.si + 1, p.ei + 1) : undefined}
              {...tip.hit(t('app.chart.episodeShort', { season: sn(p.si), episode: p.ei + 1 }), [
                { color, label: t('app.chart.viewers'), value: f.number(p.v) },
                { label: t('app.chart.vsFirstEpisode'), value: f.percent(p.v / base) },
              ])}
            />
          </g>
        ))}
        {n > 0 && (
          <>
            <circle cx={x(last)} cy={y(pts[last].v)} r={4} fill={color} stroke="var(--surface)" strokeWidth={2} />
            <text className={styles.strong} x={width - m.r} y={m.t - 4} textAnchor="end">
              {`${f.percent(pts[last].v / base)} ${t('app.chart.retentionLabel')}`}
            </text>
          </>
        )}
      </svg>
      {tip.node}
    </div>
  )
}
