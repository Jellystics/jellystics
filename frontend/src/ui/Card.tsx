import type { ReactNode } from 'react'
import styles from './Card.module.css'

const cx = (...a: (string | false | undefined)[]) => a.filter(Boolean).join(' ')

export function Card({ children, className }: { children: ReactNode; className?: string }) {
  return <section className={cx(styles.card, className)}>{children}</section>
}
export function CardHeader({ title, subtitle, action }: { title: ReactNode; subtitle?: ReactNode; action?: ReactNode }) {
  return (
    <div className={styles.header}>
      <div>
        <h3>{title}</h3>
        {subtitle && <div className={styles.sub}>{subtitle}</div>}
      </div>
      {action}
    </div>
  )
}
export function CardBody({ flush, children }: { flush?: boolean; children?: ReactNode }) {
  return <div className={cx(styles.body, flush && styles.flush)}>{children}</div>
}
export function CardFooter({ children }: { children?: ReactNode }) {
  return <div className={styles.footer}>{children}</div>
}
