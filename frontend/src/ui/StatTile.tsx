import type { ReactNode } from 'react'
import { ArrowDown, ArrowUp, ChevronRight } from 'lucide-react'
import styles from './StatTile.module.css'

export type StatDelta = { text: string; direction: 'up' | 'down' | 'flat'; good: boolean }

const ICON = { up: ArrowUp, down: ArrowDown, flat: ChevronRight } as const

function Sparkline({ values }: { values: number[] }) {
  if (values.length < 2) return null
  const w = 96, h = 28, max = Math.max(...values), min = Math.min(...values)
  const x = (i: number) => (i * (w - 4)) / (values.length - 1) + 2
  const y = (v: number) => h - 3 - ((v - min) / (max - min || 1)) * (h - 6)
  const d = values.map((v, i) => `${i ? 'L' : 'M'}${x(i).toFixed(1)},${y(v).toFixed(1)}`).join('')
  return (
    <svg className={styles.spark} viewBox={`0 0 ${w} ${h}`} width={w} height={h} aria-hidden="true">
      <path d={d} fill="none" stroke="var(--accent)" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
      <circle cx={x(values.length - 1)} cy={y(values[values.length - 1])} r="2.5" fill="var(--accent)" />
    </svg>
  )
}

export function StatTile({ label, value, unit, delta, comparedTo, foot, trend, selected, onSelect }: {
  label: string; value: string; unit?: string; delta?: StatDelta; comparedTo?: string; foot?: ReactNode
  trend?: number[]; selected?: boolean; onSelect?: () => void
}) {
  const body = (
    <>
      <span className={styles.label}>{label}</span>
      <span className={styles.value}>{value}{unit && <small>{unit}</small>}</span>
      {(delta || comparedTo || foot || trend) && (
        <span className={styles.foot}>
          {delta && (() => {
            const Icon = ICON[delta.direction]
            const tone = delta.direction === 'flat' ? styles.flat : delta.good ? styles.good : styles.bad
            return (
              <span className={`${styles.delta} ${tone}`}>
                <Icon size={12} aria-hidden="true" data-icon={delta.direction} />
                {delta.text}
              </span>
            )
          })()}
          {comparedTo && <span>{comparedTo}</span>}
          {foot}
          {trend && <Sparkline values={trend} />}
        </span>
      )}
    </>
  )
  if (onSelect) {
    return (
      <button type="button" className={[styles.stat, styles.selectable, selected && styles.selected].filter(Boolean).join(' ')} aria-pressed={!!selected} onClick={onSelect}>
        {body}
      </button>
    )
  }
  return <div className={styles.stat}>{body}</div>
}
