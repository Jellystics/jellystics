import type { ReactNode } from 'react'
import styles from './EmptyState.module.css'

export function EmptyState({ icon, title, description, action, role }: { icon?: ReactNode; title: string; description?: string; action?: ReactNode; role?: 'alert' | 'status' }) {
  return (
    <div className={styles.empty} role={role}>
      {icon && <div className={styles.ico} aria-hidden="true">{icon}</div>}
      <b>{title}</b>
      {description && <span>{description}</span>}
      {action}
    </div>
  )
}
