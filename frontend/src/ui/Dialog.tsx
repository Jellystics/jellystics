import type { ReactNode } from 'react'
import * as RDialog from '@radix-ui/react-dialog'
import styles from './Dialog.module.css'

export function Dialog({ open, onOpenChange, title, description, children, footer }: {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  description?: string
  children?: ReactNode
  footer?: ReactNode
}) {
  return (
    <RDialog.Root open={open} onOpenChange={onOpenChange}>
      <RDialog.Portal>
        <RDialog.Overlay className={styles.overlay} />
        <RDialog.Content className={styles.dialog} {...(description ? {} : { 'aria-describedby': undefined })}>
          <div className={styles.head}>
            <RDialog.Title className={styles.title}>{title}</RDialog.Title>
            {description && <RDialog.Description className={styles.desc}>{description}</RDialog.Description>}
          </div>
          {children && <div className={styles.body}>{children}</div>}
          {footer && <div className={styles.foot}>{footer}</div>}
        </RDialog.Content>
      </RDialog.Portal>
    </RDialog.Root>
  )
}
