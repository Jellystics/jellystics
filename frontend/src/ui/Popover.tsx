import type { ReactNode } from 'react'
import * as RPopover from '@radix-ui/react-popover'
import styles from './Menu.module.css'

export function Popover({ trigger, children, align = 'start' }: { trigger: ReactNode; children: ReactNode; align?: 'start' | 'end' }) {
  return (
    <RPopover.Root>
      <RPopover.Trigger asChild>{trigger}</RPopover.Trigger>
      <RPopover.Portal>
        <RPopover.Content className={styles.pop} align={align} sideOffset={4}>{children}</RPopover.Content>
      </RPopover.Portal>
    </RPopover.Root>
  )
}
