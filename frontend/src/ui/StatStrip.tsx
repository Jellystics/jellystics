import type { CSSProperties, ReactNode } from 'react'
import styles from './StatStrip.module.css'

export function StatStrip({ children, columns = 4 }: { children: ReactNode; columns?: number }) {
  return <div className={styles.strip} style={{ '--n': columns } as CSSProperties}>{children}</div>
}
