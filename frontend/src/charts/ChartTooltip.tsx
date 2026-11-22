import { useCallback, useState, type FocusEvent, type MouseEvent, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import styles from './charts.module.css'

export type TipRow = { color?: string | null; label: string; value: string }
type TipState = { x: number; y: number; title?: string; rows: TipRow[] }

/** Per-chart tooltip state. Spread `hit(...)` on a hit element; render `node` once. */
export function useChartTooltip() {
  const [tip, setTip] = useState<TipState | null>(null)
  const hide = useCallback(() => setTip(null), [])
  const hit = useCallback((title: string | undefined, rows: TipRow[], onActive?: () => void) => ({
    'data-hit': '' as const,
    tabIndex: 0,
    onMouseMove: (e: MouseEvent) => { setTip({ x: e.clientX, y: e.clientY, title, rows }); onActive?.() },
    onFocus: (e: FocusEvent) => { const r = e.currentTarget.getBoundingClientRect(); setTip({ x: r.right, y: r.top, title, rows }); onActive?.() },
    onMouseLeave: hide,
    onBlur: hide,
  }), [hide])
  const node: ReactNode = tip ? createPortal(<Tip {...tip} />, document.body) : null
  return { hit, hide, node, active: tip !== null }
}

function Tip({ x, y, title, rows }: TipState) {
  const w = 160, h = 24 + rows.length * 20
  let left = x + 14, top = y + 14
  if (left + w > window.innerWidth - 8) left = x - w - 14
  if (top + h > window.innerHeight - 8) top = y - h - 14
  return (
    <div role="tooltip" className={styles.tip} style={{ left, top }}>
      {title && <div className={styles.tipH}>{title}</div>}
      {rows.map((r, i) => (
        <div key={i} className={styles.tipR}>
          <span>{r.color && <i style={{ background: r.color }} />}{r.label}</span>
          <b>{r.value}</b>
        </div>
      ))}
    </div>
  )
}
