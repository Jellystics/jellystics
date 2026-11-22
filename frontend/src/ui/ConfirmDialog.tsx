import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from './Button'
import { Dialog } from './Dialog'

export function ConfirmDialog({ open, onOpenChange, title, description, confirmLabel, danger, onConfirm }: {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  description: string
  confirmLabel: string
  danger?: boolean
  onConfirm: () => void | Promise<void>
}) {
  const { t } = useTranslation()
  const [busy, setBusy] = useState(false)
  const confirm = async () => {
    setBusy(true)
    try { await onConfirm(); onOpenChange(false) } finally { setBusy(false) }
  }
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={title}
      description={description}
      footer={
        <>
          <Button variant="secondary" onClick={() => onOpenChange(false)}>{t('app.action.cancel')}</Button>
          <Button variant={danger ? 'danger' : 'primary'} disabled={busy} onClick={confirm}>{confirmLabel}</Button>
        </>
      }
    />
  )
}
