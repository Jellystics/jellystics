import type { MouseEventHandler, ReactNode } from 'react'
import { X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import styles from './Chip.module.css'

export function Chip({ pressed, onRemove, onClick, children }: { pressed?: boolean; onRemove?: () => void; onClick?: MouseEventHandler<HTMLButtonElement>; children: ReactNode }) {
  const { t } = useTranslation()
  return (
    <span className={styles.wrap}>
      <button type="button" className={styles.chip} aria-pressed={pressed} onClick={onClick}>
        {children}
      </button>
      {onRemove && (
        <button type="button" className={styles.x} aria-label={t('app.action.remove')} onClick={onRemove}>
          <X size={12} strokeWidth={1.6} />
        </button>
      )}
    </span>
  )
}
