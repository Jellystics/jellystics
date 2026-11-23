import type { ReactNode } from 'react'
import * as DropdownMenu from '@radix-ui/react-dropdown-menu'
import styles from './Menu.module.css'

export type MenuItem = { label: string; icon?: ReactNode; hint?: string; onSelect: () => void; danger?: boolean } | 'separator' | { group: string }

export function Menu({ trigger, items }: { trigger: ReactNode; items: MenuItem[] }) {
  return (
    <DropdownMenu.Root>
      <DropdownMenu.Trigger asChild>{trigger}</DropdownMenu.Trigger>
      <DropdownMenu.Portal>
        <DropdownMenu.Content className={styles.pop} align="end" sideOffset={4}>
          {items.map((it, i) => {
            if (it === 'separator') return <DropdownMenu.Separator key={i} className={styles.sep} />
            if ('group' in it) return <DropdownMenu.Label key={i} className={styles.label}>{it.group}</DropdownMenu.Label>
            return (
              <DropdownMenu.Item key={i} className={[styles.item, it.danger && styles.danger].filter(Boolean).join(' ')} onSelect={it.onSelect}>
                {it.icon}
                {it.label}
                {it.hint && <span className={styles.hint}>{it.hint}</span>}
              </DropdownMenu.Item>
            )
          })}
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  )
}
