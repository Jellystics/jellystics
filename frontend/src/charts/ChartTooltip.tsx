import { useCallback, useLayoutEffect, useRef, useState, type FocusEvent, type KeyboardEvent, type MouseEvent, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import styles from './charts.module.css'

export type TipRow = { color?: string | null; label: string; value: string }
type TipState = { x: number; y: number; title?: string; rows: TipRow[] }

/**
 * Per-chart tooltip state. Spread `hit(...)` on a hit element; render `node` once.
 * Pass `activate` to make the element a keyboard-operable button (click, Enter, Space).
 */
export function useChartTooltip() {
  const [tip, setTip] = useState<TipState | null>(null)
  const hide = useCallback(() => setTip(null), [])
  const hit = useCallback((title: string | undefined, rows: TipRow[], onActive?: () => void, activate?: () => void) => ({
    'data-hit': '' as const,
    tabIndex: 0,
    role: activate ? ('button' as const) : ('graphics-symbol' as const),
    'aria-label': title ?? rows.map((r) => `${r.label} ${r.value}`).join(', '),
    ...(activate ? {
      style: { cursor: 'pointer' },
      onClick: activate,
      onKeyDown: (e: KeyboardEvent) => {
        if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); activate() }
      },
    } : {}),
    onMouseMove: (e: MouseEvent) => { setTip({ x: e.clientX, y: e.clientY, title, rows }); onActive?.() },
    onFocus: (e: FocusEvent) => { const r = e.currentTarget.getBoundingClientRect(); setTip({ x: r.right, y: r.top, title, rows }); onActive?.() },
    onMouseLeave: hide,
    onBlur: hide,
  }), [hide])
  const node: ReactNode = tip ? createPortal(<Tip {...tip} />, document.body) : null
  return { hit, hide, node, active: tip !== null }
}

function Tip({ x, y, title, rows }: TipState) {
  const ref = useRef<HTMLDivElement>(null)
  // Flip against the viewport using the real rendered size, before paint.
  useLayoutEffect(() => {
    const el = ref.current
    if (!el) return
    const { width: w, height: h } = el.getBoundingClientRect()
    let left = x + 14, top = y + 14
    if (left + w > window.innerWidth - 8) left = Math.max(8, x - w - 14)
    if (top + h > window.innerHeight - 8) top = Math.max(8, y - h - 14)
    el.style.left = `${left}px`
    el.style.top = `${top}px`
  })
  return (
    <div ref={ref} role="tooltip" className={styles.tip} style={{ left: x + 14, top: y + 14 }}>
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
