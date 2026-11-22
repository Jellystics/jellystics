import type { ReactNode } from 'react'
import * as RTabs from '@radix-ui/react-tabs'
import styles from './Tabs.module.css'

export function Tabs({ value, onValueChange, items, children }: {
  value: string
  onValueChange: (v: string) => void
  items: { value: string; label: string; count?: number }[]
  children?: ReactNode
}) {
  return (
    <RTabs.Root value={value} onValueChange={onValueChange}>
      <RTabs.List className={styles.tabs}>
        {items.map((it) => (
          <RTabs.Trigger key={it.value} value={it.value} className={styles.tab}>
            {it.label}
            {it.count !== undefined && <span className={styles.count}>{it.count}</span>}
          </RTabs.Trigger>
        ))}
      </RTabs.List>
      {children}
    </RTabs.Root>
  )
}
