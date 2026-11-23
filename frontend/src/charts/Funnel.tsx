import { useTranslation } from 'react-i18next'
import { useFormat } from '@/lib/format'
import { useChartTooltip } from './ChartTooltip'
import { useChartWidth } from './useChartWidth'
import { barPathH } from './scale'
import styles from './charts.module.css'

const RAMP = ['var(--seq-6)', 'var(--seq-5)', 'var(--seq-4)', 'var(--seq-3)']

/** `note` replaces the share of the first stage (e.g. "80 % of views · 4/5"). */
export function Funnel({ stages }: { stages: { label: string; value: number; note?: string }[] }) {
  const { t } = useTranslation()
  const f = useFormat()
  const [ref, width] = useChartWidth()
  const tip = useChartTooltip()
  const max = stages[0]?.value || 1
  const longest = Math.max(0, ...stages.map((s) => s.label.length))
  // Label column sized to the longest label (≈7px/char), capped so bars keep room on narrow containers.
  const rowH = 34, lw = Math.min(Math.round(width * 0.4), Math.max(90, longest * 7 + 8)), height = stages.length * rowH

  return (
    <div className={styles.chart} ref={ref}>
      <svg width={width} height={height} role="img" aria-label={t('app.chart.funnel')}>
        {stages.map((s, i) => {
          const w = Math.max(0, ((width - lw - 90) * s.value) / max), yy = i * rowH + 6
          const color = RAMP[Math.min(i, RAMP.length - 1)]
          const share = f.percent(s.value / max)
          return (
            <g key={i}>
              <text className={styles.strong} x={0} y={yy + 15}>{s.label}</text>
              <path
                d={barPathH(lw, yy, Math.max(2, w), 22)} fill={color}
                {...tip.hit(s.label, [{ color, label: t('app.chart.plays'), value: f.number(s.value) }, { label: t('app.chart.vsFirst'), value: share }])}
              />
              <text className={styles.strong} x={lw + Math.max(2, w) + 8} y={yy + 15}>
                {f.number(s.value)}
                {i > 0 && <tspan className={styles.light} dx={8}>{s.note ?? share}</tspan>}
              </text>
            </g>
          )
        })}
      </svg>
      {tip.node}
    </div>
  )
}
