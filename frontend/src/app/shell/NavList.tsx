import { useTranslation } from 'react-i18next'
import { AppLink } from '@/ui'
import { useActiveNav } from './crumbs'
import { NAV_GROUPS } from './nav'
import styles from './Sidebar.module.css'

export function NavList({ liveCount, onNavigate }: { liveCount: number; onNavigate?: () => void }) {
  const { t } = useTranslation()
  const active = useActiveNav()
  return (
    <nav aria-label={t('app.nav.main')} className={styles.nav}>
      {NAV_GROUPS.map((g) => (
        <div key={g.key} className={styles.group}>
          <div className={styles.label}>{t(`app.navGroup.${g.key}`)}</div>
          {g.items.map(({ key, to, icon: Icon }) => (
            <AppLink key={key} to={to} onClick={onNavigate} className={[styles.item, active === key && styles.active].filter(Boolean).join(' ')} aria-current={active === key ? 'page' : undefined}>
              <Icon size={16} strokeWidth={1.6} aria-hidden />
              {t(`app.nav.${key}`)}
              {key === 'live' && liveCount > 0 && <span className={styles.liveDot} role="img" aria-label={t('app.nav.liveCount', { count: liveCount })} />}
            </AppLink>
          ))}
        </div>
      ))}
    </nav>
  )
}
