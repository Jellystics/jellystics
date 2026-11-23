import { useTranslation } from 'react-i18next'
import { useFormat } from '@/lib/format'
import { useChartTooltip } from './ChartTooltip'
import { useChartWidth } from './useChartWidth'
import { SEQ, seqIndex, weekdayLong } from './labels'
import styles from './charts.module.css'

export function WeekHeatmap({ matrix, unit }: { matrix: number[][]; unit: string }) {
  const { t } = useTranslation()
  const f = useFormat()
  const [ref, width] = useChartWidth()
  const tip = useChartTooltip()
  const m = { t: 4, r: 4, b: 22, l: 40 }, gap = 2, ch = 18
  const cw = (width - m.l - m.r) / 24
  const height = m.t + ch * 7 + m.b
  const max = matrix.reduce((a, row) => Math.max(a, ...row), 0)
  const days = f.weekdays(), full = weekdayLong(f.locale)

  return (
    <div>
      <div className={styles.chart} ref={ref}>
        <svg width={width} height={height} role="img" aria-label={t('app.chart.weekHeatmap')}>
          {matrix.map((row, d) => (
            <g key={d}>
              <text x={m.l - 8} y={m.t + d * ch + ch / 2 + 3.5} textAnchor="end">{days[d]}</text>
              {row.map((v, h) => (
                <rect
                  key={h} x={m.l + h * cw} y={m.t + d * ch} width={cw - gap} height={ch - gap} rx={2}
                  fill={SEQ[seqIndex(v, max, 7)]}
                  {...tip.hit(`${full[d]} · ${f.hour(h)}`, [{ label: unit, value: f.number(v) }])}
                />
              ))}
            </g>
          ))}
          {[0, 6, 12, 18, 23].map((h) => (
            <text key={h} x={m.l + h * cw + cw / 2} y={height - 6} textAnchor="middle">{f.hour(h)}</text>
          ))}
        </svg>
        {tip.node}
      </div>
      <div className={styles.scale}>
        {t('app.chart.less')}
        {SEQ.map((c, i) => <span key={i} style={{ background: c }} />)}
        {t('app.chart.more')}
      </div>
    </div>
  )
}
