import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { Tooltip } from './Tooltip'
import { useFormat } from '@/lib/format'
import styles from './BarList.module.css'

export type BarListRow = { key: string; label: string; value: number; to?: string; lead?: ReactNode; badge?: ReactNode; fill?: string; tooltip?: [string, string][] }

export function BarList({ rows, valueLabel, labelHeader, format, max }: {
  rows: BarListRow[]; valueLabel: string; labelHeader?: string; format?: (v: number) => string; max?: number
}) {
  const f = useFormat()
  const fmt = format ?? f.number
  const top = max ?? Math.max(1, ...rows.map((r) => r.value))
  return (
    <div className={styles.barlist}>
      {(labelHeader || valueLabel) && <div className={styles.head}><span>{labelHeader}</span><span>{valueLabel}</span></div>}
      {rows.map((r) => {
        const inner = (
          <>
            <span className={styles.fill} style={{ width: `${Math.min(100, (r.value / top) * 100)}%`, ...(r.fill ? { background: r.fill } : {}) }} />
            <span className={styles.label}>
              {r.lead}
              <span className={styles.t}>{r.label}</span>
              {r.badge}
            </span>
            <span className={styles.val}>{fmt(r.value)}</span>
          </>
        )
        const row = r.to
          ? <Link key={r.key} to={r.to} className={`${styles.row} ${styles.link}`}>{inner}</Link>
          : <div key={r.key} className={styles.row}>{inner}</div>
        return r.tooltip
          ? (
            <Tooltip key={r.key} content={<>{r.tooltip.map(([k, v]) => <div key={k}>{k} : {v}</div>)}</>}>{row}</Tooltip>
          )
          : row
      })}
    </div>
  )
}
