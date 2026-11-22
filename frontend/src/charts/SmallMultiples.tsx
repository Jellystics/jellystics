import { useTranslation } from 'react-i18next'
import { useFormat } from '@/lib/format'
import { useChartTooltip } from './ChartTooltip'
import { useChartWidth } from './useChartWidth'
import { barPath } from './scale'
import styles from './charts.module.css'

type Item = { name: string; color: string; values: number[] }

export function SmallMultiples({ items }: { items: Item[] }) {
  const { t } = useTranslation()
  const f = useFormat()
  const max = Math.max(1, ...items.flatMap((it) => it.values))
  return (
    <div className={styles.multiples}>
      {items.map((it) => {
        const peak = it.values.indexOf(Math.max(...it.values))
        return (
          <div key={it.name}>
            <div className={styles.multiplesH}>
              <span><i style={{ background: it.color }} aria-hidden="true" />{it.name}</span>
              <span>{`${t('app.chart.peak')} ${f.hour(Math.max(0, peak))}`}</span>
            </div>
            <Mini item={it} max={max} />
          </div>
        )
      })}
    </div>
  )
}

function Mini({ item, max, height = 110 }: { item: Item; max: number; height?: number }) {
  const { t } = useTranslation()
  const f = useFormat()
  const [ref, width] = useChartWidth()
  const tip = useChartTooltip()
  const m = { t: 4, b: 18, l: 2, r: 2 }
  const ih = height - m.t - m.b, band = (width - m.l - m.r) / Math.max(1, item.values.length), bw = Math.max(2, Math.min(10, band - 2))
  return (
    <div className={styles.chart} ref={ref}>
      <svg width={width} height={height} role="img" aria-label={item.name}>
        {item.values.map((v, i) => {
          const h = (v / max) * ih
          return (
            <g key={i}>
              <path d={barPath(m.l + band * i + (band - bw) / 2, m.t + ih - h, bw, h, 2)} fill={item.color} />
              <rect
                x={m.l + band * i} y={m.t} width={band} height={ih} fill="transparent"
                {...tip.hit(`${item.name} · ${f.hour(i)}`, [{ color: item.color, label: t('app.chart.views'), value: f.number(v) }])}
              />
            </g>
          )
        })}
        <line x1={0} x2={width} y1={m.t + ih} y2={m.t + ih} stroke="var(--axis)" />
        {[0, 6, 12, 18].map((h) => <text key={h} x={m.l + band * h + band / 2} y={height - 3} textAnchor="middle">{f.hour(h)}</text>)}
      </svg>
      {tip.node}
    </div>
  )
}
