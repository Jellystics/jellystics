import { useTranslation } from 'react-i18next'
import { useFormat } from '@/lib/format'
import { useChartTooltip } from './ChartTooltip'
import styles from './charts.module.css'

const STEPS = [1, 2, 3, 4, 5, 6].map((i) => `var(--seq-${i})`)

export function Matrix({ rows, cols, values, unit, labelWidth = 120, width = 440, cell = 28 }: {
  rows: string[]
  cols: string[]
  values: (number | null)[][]
  unit: string
  labelWidth?: number
  width?: number
  cell?: number
}) {
  const { t } = useTranslation()
  const f = useFormat()
  const tip = useChartTooltip()
  const nums = values.flat().filter((v): v is number => v !== null)
  const max = Math.max(0, ...nums) || 1
  const cw = Math.min(cell * 1.6, (width - labelWidth) / Math.max(1, cols.length)), ch = cell, top = 22
  const natural = labelWidth + cols.length * cw
  const height = top + rows.length * ch + 4

  return (
    // Drawn at natural size and scrolled when the container is narrower, so text never shrinks below its 11px design size.
    <div className={styles.chart} style={{ overflowX: 'auto' }}>
      <svg viewBox={`0 0 ${natural} ${height}`} width={natural} height={height} style={{ display: 'block' }} role="img" aria-label={t('app.chart.matrix')}>
        {cols.map((c, j) => <text key={j} x={labelWidth + j * cw + cw / 2} y={14} textAnchor="middle">{c}</text>)}
        {rows.map((r, i) => (
          <g key={i}>
            <text className={styles.strong} x={labelWidth - 8} y={top + i * ch + ch / 2 + 4} textAnchor="end">{r}</text>
            {cols.map((c, j) => {
              const v = values[i]?.[j] ?? null
              const x = labelWidth + j * cw, y = top + i * ch
              if (v === null) return <rect key={j} x={x} y={y} width={cw - 2} height={ch - 2} rx={3} fill="transparent" stroke="var(--grid)" />
              const idx = Math.min(5, Math.floor((v / max) * 5.999))
              return (
                <g key={j}>
                  <rect
                    x={x} y={y} width={cw - 2} height={ch - 2} rx={3} fill={v === 0 ? 'var(--seq-0)' : STEPS[idx]}
                    {...tip.hit(`${r} · ${c}`, [{ label: unit, value: f.number(v) }])}
                  />
                  <text x={x + (cw - 2) / 2} y={y + ch / 2 + 3} textAnchor="middle" className={`mx-t ${idx >= 3 ? 'hi' : 'lo'}`} style={{ pointerEvents: 'none' }}>{f.number(v)}</text>
                </g>
              )
            })}
          </g>
        ))}
      </svg>
      {tip.node}
    </div>
  )
}
