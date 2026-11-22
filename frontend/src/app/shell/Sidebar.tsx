import { Search } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { AppLink } from '@/ui'
import { NavList } from './NavList'
import { UserMenu } from './UserMenu'
import { Brand } from './Brand'
import styles from './Sidebar.module.css'

export function Sidebar({ liveCount, onOpenCommand }: { liveCount: number; onOpenCommand: () => void }) {
  const { t } = useTranslation()
  return (
    <aside className={styles.side}>
      <AppLink to="/" className={styles.brand}><Brand />Jellystics</AppLink>
      <button type="button" className={styles.search} onClick={onOpenCommand}>
        <Search size={14} strokeWidth={1.6} aria-hidden />
        <span>{t('app.command.search')}</span>
        <kbd className={styles.kbd}>⌘K</kbd>
      </button>
      <NavList liveCount={liveCount} />
      <UserMenu />
    </aside>
  )
}
