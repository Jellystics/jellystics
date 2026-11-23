import type { ReactNode } from 'react'
import styles from './EmptyState.module.css'

export function EmptyState({ icon, title, description, action, role, headingLevel }: {
  icon?: ReactNode; title: string; description?: string; action?: ReactNode; role?: 'alert' | 'status'
  /** Renders the title as <h1>–<h3> when the empty state stands for the whole page (404, entity not found). */
  headingLevel?: 1 | 2 | 3
}) {
  const Title = headingLevel ? (`h${headingLevel}` as const) : 'p'
  return (
    <div className={styles.empty} role={role}>
      {icon && <div className={styles.ico} aria-hidden="true">{icon}</div>}
      <Title className={styles.title}>{title}</Title>
      {description && <span>{description}</span>}
      {action}
    </div>
  )
}
