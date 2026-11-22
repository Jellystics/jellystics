import type { ReactNode } from 'react'
import * as RDialog from '@radix-ui/react-dialog'
import { X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from './Button'
import styles from './SidePanel.module.css'

/**
 * Right-hand drawer (Radix Dialog): focus trap, Escape and overlay click close
 * it. Meant to be bound to a URL param (e.g. `?track=`) by the caller.
 */
export function SidePanel({ open, onOpenChange, title, subtitle, lead, children }: {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  subtitle?: ReactNode
  /** Art or avatar shown left of the title. */
  lead?: ReactNode
  children?: ReactNode
}) {
  const { t } = useTranslation()
  return (
    <RDialog.Root open={open} onOpenChange={onOpenChange}>
      <RDialog.Portal>
        <RDialog.Overlay className={styles.overlay} />
        <RDialog.Content className={styles.panel} {...(subtitle ? {} : { 'aria-describedby': undefined })}>
          <div className={styles.head}>
            {lead}
            <div className={styles.titles}>
              <RDialog.Title className={styles.title}>{title}</RDialog.Title>
              {subtitle && <RDialog.Description asChild><div className={styles.sub}>{subtitle}</div></RDialog.Description>}
            </div>
            <RDialog.Close asChild>
              <Button variant="ghost" size="sm" iconOnly aria-label={t('app.sidePanel.close')} icon={<X size={16} strokeWidth={1.6} />} />
            </RDialog.Close>
          </div>
          <div className={styles.body}>{children}</div>
        </RDialog.Content>
      </RDialog.Portal>
    </RDialog.Root>
  )
}
