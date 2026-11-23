import type { ReactNode } from 'react'
import styles from './EntityHeader.module.css'

export function EntityHeader({ art, kicker, title, meta, actions }: { art: ReactNode; kicker?: ReactNode; title: string; meta?: ReactNode[]; actions?: ReactNode }) {
  return (
    <header className={styles.eh}>
      {art}
      <div className={styles.main}>
        {kicker && <div className={styles.kicker}>{kicker}</div>}
        <h1>{title}</h1>
        {meta && meta.length > 0 && <div className={styles.meta}>{meta.map((m, i) => <span key={i}>{m}</span>)}</div>}
      </div>
      {actions && <div className={styles.actions}>{actions}</div>}
    </header>
  )
}
