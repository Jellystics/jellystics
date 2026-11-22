import { useCallback, useEffect, useRef, useSyncExternalStore } from 'react'
import { Check, CircleAlert, Info } from 'lucide-react'
import styles from './Toast.module.css'

type Kind = 'success' | 'error' | 'info'
interface ToastItem { id: number; kind: Kind; message: string; action?: { label: string; onClick: () => void } }

const MAX = 5
const DURATION = 5000
let items: ToastItem[] = []
let nextId = 1
const listeners = new Set<() => void>()
const emit = () => listeners.forEach((l) => l())
const subscribe = (l: () => void) => { listeners.add(l); return () => { listeners.delete(l) } }
const snapshot = () => items

function push(kind: Kind, message: string, action?: ToastItem['action']) {
  items = [...items, { id: nextId++, kind, message, action }].slice(-MAX)
  emit()
}
function dismiss(id: number) { items = items.filter((i) => i.id !== id); emit() }

export const toast = {
  success: (msg: string) => push('success', msg),
  error: (msg: string, action?: { label: string; onClick: () => void }) => push('error', msg, action),
  info: (msg: string) => push('info', msg),
}

function ToastView({ item }: { item: ToastItem }) {
  const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)
  const start = useCallback(() => { clearTimeout(timer.current); timer.current = setTimeout(() => dismiss(item.id), DURATION) }, [item.id])
  const stop = useCallback(() => clearTimeout(timer.current), [])
  useEffect(() => { start(); return stop }, [start, stop])
  const Icon = item.kind === 'success' ? Check : item.kind === 'error' ? CircleAlert : Info
  return (
    <div className={styles.toast} role={item.kind === 'error' ? 'alert' : undefined} onMouseEnter={stop} onMouseLeave={start} onFocus={stop} onBlur={start}>
      <span className={item.kind === 'success' ? styles.ok : item.kind === 'error' ? styles.err : undefined}><Icon size={16} strokeWidth={1.6} aria-hidden /></span>
      <span>{item.message}</span>
      {item.action && <button type="button" className={styles.action} onClick={() => { item.action!.onClick(); dismiss(item.id) }}>{item.action.label}</button>}
    </div>
  )
}

export function Toaster() {
  const list = useSyncExternalStore(subscribe, snapshot, snapshot)
  return (
    <div className={styles.stack} role="status" aria-live="polite">
      {list.map((i) => <ToastView key={i.id} item={i} />)}
    </div>
  )
}
