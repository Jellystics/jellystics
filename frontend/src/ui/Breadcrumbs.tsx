import { AppLink as Link } from '@/app/AppLink'
import { useTranslation } from 'react-i18next'
import styles from './Breadcrumbs.module.css'

export function Breadcrumbs({ items }: { items: { label: string; to?: string }[] }) {
  const { t } = useTranslation()
  return (
    <nav aria-label={t('app.nav.breadcrumbs')} className={styles.crumbs}>
      {items.map((it, i) => {
        const last = i === items.length - 1
        return (
          <span key={i} className={styles.item}>
            {i > 0 && <span className={styles.sep} aria-hidden="true">/</span>}
            {last || !it.to
              ? <span className={styles.here} aria-current={last ? 'page' : undefined}>{it.label}</span>
              : <Link to={it.to}>{it.label}</Link>}
          </span>
        )
      })}
    </nav>
  )
}
