import type { ReactElement, ReactNode } from 'react'
import * as RTooltip from '@radix-ui/react-tooltip'
import styles from './Tooltip.module.css'

export function Tooltip({ content, children }: { content: ReactNode; children: ReactElement }) {
  return (
    <RTooltip.Provider delayDuration={300}>
      <RTooltip.Root>
        <RTooltip.Trigger asChild>{children}</RTooltip.Trigger>
        <RTooltip.Portal>
          <RTooltip.Content className={styles.tip} sideOffset={6}>{content}</RTooltip.Content>
        </RTooltip.Portal>
      </RTooltip.Root>
    </RTooltip.Provider>
  )
}
