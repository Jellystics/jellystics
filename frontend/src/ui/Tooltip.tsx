import { createContext, useContext, type ReactElement, type ReactNode } from 'react'
import * as RTooltip from '@radix-ui/react-tooltip'
import styles from './Tooltip.module.css'

const HasProvider = createContext(false)

/** Mount once near the app root: all Tooltips then share one provider (delay and skip-delay behaviour). */
export function TooltipProvider({ children }: { children: ReactNode }) {
  return (
    <HasProvider.Provider value>
      <RTooltip.Provider delayDuration={300}>{children}</RTooltip.Provider>
    </HasProvider.Provider>
  )
}

export function Tooltip({ content, children }: { content: ReactNode; children: ReactElement }) {
  const hasProvider = useContext(HasProvider)
  const tip = (
    <RTooltip.Root>
      <RTooltip.Trigger asChild>{children}</RTooltip.Trigger>
      <RTooltip.Portal>
        <RTooltip.Content className={styles.tip} sideOffset={6}>{content}</RTooltip.Content>
      </RTooltip.Portal>
    </RTooltip.Root>
  )
  // Fallback keeps a lone Tooltip working (tests, isolated renders) when no app-level provider is mounted.
  return hasProvider ? tip : <RTooltip.Provider delayDuration={300}>{tip}</RTooltip.Provider>
}
