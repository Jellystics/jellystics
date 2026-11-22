import type { MouseEventHandler, ReactNode } from 'react'
import { X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import styles from './Chip.module.css'

export function Chip({ pressed, onRemove, onClick, children, label }: { pressed?: boolean; onRemove?: () => void; onClick?: MouseEventHandler<HTMLButtonElement>; children: ReactNode; /** Plain-text name for the remove button when children are not text. */ label?: string }) {
  const { t } = useTranslation()
  const name = label ?? (typeof children === 'string' || typeof children === 'number' ? String(children) : undefined)
  return (
    <span className={styles.wrap}>
      <button type="button" className={styles.chip} aria-pressed={pressed} onClick={onClick}>
        {children}
      </button>
      {onRemove && (
        <button type="button" className={styles.x} aria-label={name ? t('app.action.removeNamed', { name }) : t('app.action.remove')} onClick={onRemove}>
          <X size={12} strokeWidth={1.6} />
        </button>
      )}
    </span>
  )
}
