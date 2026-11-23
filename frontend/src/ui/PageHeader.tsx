import type { ReactNode } from 'react'
import styles from './PageHeader.module.css'

export function PageHeader({ title, subtitle, actions }: { title: string; subtitle?: ReactNode; actions?: ReactNode }) {
  return (
    <header className={styles.ph}>
      <div>
        <h1>{title}</h1>
        {subtitle && <div className={styles.sub}>{subtitle}</div>}
      </div>
      {actions && <div className={styles.actions}>{actions}</div>}
    </header>
  )
}
