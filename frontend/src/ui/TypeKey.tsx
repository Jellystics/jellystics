import { useTranslation } from 'react-i18next'
import type { MediaType } from './Badge'
import styles from './TypeKey.module.css'

export function TypeKey({ type, label }: { type: MediaType; label?: string }) {
  const { t } = useTranslation()
  return (
    <span className={styles.tkey}>
      <i style={{ background: `var(--t-${type})` }} aria-hidden="true" />
      {label ?? t(`app.type.${type}`)}
    </span>
  )
}
