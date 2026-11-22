import { Children, cloneElement, isValidElement, type ReactElement, type ReactNode } from 'react'
import styles from './Field.module.css'

export function Field({ label, htmlFor, hint, error, children }: { label: string; htmlFor: string; hint?: string; error?: string; children: ReactNode }) {
  const noteId = `${htmlFor}-note`
  const hasNote = !!(error || hint)
  // Wire the hint/error to the control so screen readers announce it with the field.
  const wired = Children.map(children, (child) => {
    if (!hasNote || !isValidElement(child) || (child.props as { id?: string }).id !== htmlFor) return child
    const props = child.props as { 'aria-describedby'?: string }
    const describedBy = [props['aria-describedby'], noteId].filter(Boolean).join(' ')
    return cloneElement(child as ReactElement<Record<string, unknown>>, { 'aria-describedby': describedBy, ...(error ? { 'aria-invalid': true } : {}) })
  })
  return (
    <div className={styles.field}>
      <label className={styles.label} htmlFor={htmlFor}>{label}</label>
      {wired}
      {error ? <span id={noteId} className={styles.error} role="alert">{error}</span> : hint ? <span id={noteId} className={styles.hint}>{hint}</span> : null}
    </div>
  )
}
