import type { ReactNode } from 'react'
import styles from './EmptyState.module.css'

export function EmptyState({ icon, title, description, action }: { icon?: ReactNode; title: string; description?: string; action?: ReactNode }) {
  return (
    <div className={styles.empty}>
      {icon && <div className={styles.ico} aria-hidden="true">{icon}</div>}
      <b>{title}</b>
      {description && <span>{description}</span>}
      {action}
    </div>
  )
}
