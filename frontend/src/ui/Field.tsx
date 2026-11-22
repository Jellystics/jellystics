import type { ReactNode } from 'react'
import styles from './Field.module.css'

export function Field({ label, htmlFor, hint, error, children }: { label: string; htmlFor: string; hint?: string; error?: string; children: ReactNode }) {
  return (
    <div className={styles.field}>
      <label className={styles.label} htmlFor={htmlFor}>{label}</label>
      {children}
      {error ? <span className={styles.error} role="alert">{error}</span> : hint ? <span className={styles.hint}>{hint}</span> : null}
    </div>
  )
}
