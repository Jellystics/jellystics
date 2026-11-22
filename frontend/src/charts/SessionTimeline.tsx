import { useTranslation } from 'react-i18next'
import { useFormat } from '@/lib/format'
import type { MediaType } from '@/ui/Badge'
import { useChartTooltip } from './ChartTooltip'
import { useChartWidth } from './useChartWidth'
import styles from './charts.module.css'

type Span = { start: number; end: number; type: MediaType; label: string }

export function SessionTimeline({ fromHour, toHour, rows }: { fromHour: number; toHour: number; rows: { user: string; spans: Span[] }[] }) {
  const { t } = useTranslation()
  const f = useFormat()
  const [ref, width] = useChartWidth()
  const tip = useChartTooltip()
  const lw = 90, rh = 26, top = 18, height = top + rows.length * rh + 6
  const range = Math.max(1, toHour - fromHour)
  const x = (h: number) => lw + ((Math.min(toHour, Math.max(fromHour, h)) - fromHour) / range) * (width - lw - 8)
  const hm = (h: number) => `${f.hour(Math.floor(h) % 24)}${String(Math.round((h % 1) * 60)).padStart(2, '0')}`

  return (
    <div className={styles.chart} ref={ref}>
      <svg width={width} height={height} role="img" aria-label={t('app.chart.timeline')}>
        {Array.from({ length: Math.max(0, toHour - fromHour) + 1 }, (_, k) => fromHour + k).map((h) => (
          <g key={h}>
            <line x1={x(h)} x2={x(h)} y1={top - 4} y2={height} stroke="var(--grid)" />
            <text x={x(h)} y={11} textAnchor="middle">{f.hour(h % 24)}</text>
          </g>
        ))}
        {rows.map((r, i) => (
          <g key={i}>
            <text className={styles.strong} x={lw - 10} y={top + i * rh + 16} textAnchor="end">{r.user}</text>
            {r.spans.map((s, k) => {
              const color = `var(--t-${s.type})`
              return (
                <rect
                  key={k} x={x(s.start)} y={top + i * rh + 5} width={Math.max(3, x(s.end) - x(s.start) - 2)} height={14} rx={4} fill={color}
                  aria-label={s.label}
                  {...tip.hit(s.label, [{ color, label: t(`app.type.${s.type}`), value: `${hm(s.start)} → ${hm(s.end)}` }])}
                />
              )
            })}
          </g>
        ))}
      </svg>
      {tip.node}
    </div>
  )
}
