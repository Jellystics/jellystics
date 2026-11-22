import { ChevronDown, Globe, LogOut, MoonStar, Settings, Sun, Check } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useLocation, useNavigate } from 'react-router-dom'
import { Avatar, Menu, withPeriod, type MenuItem } from '@/ui'
import { getUserId, getUsername, logout } from '@/lib/auth'
import { useThemeMode } from '@/lib/themeMode'
import styles from './UserMenu.module.css'

const LANGS = ['fr', 'en', 'es'] as const

/** Signed-in user (Jellyfin avatar + name) and their menu, top right of the top bar. */
export function UserMenu() {
  const { t, i18n } = useTranslation()
  const { mode, toggle } = useThemeMode()
  const navigate = useNavigate()
  const { search } = useLocation()
  const name = getUsername() || t('app.user.guest')
  const current = i18n.resolvedLanguage || i18n.language
  const items: MenuItem[] = [
    { label: mode === 'dark' ? t('app.user.lightTheme') : t('app.user.darkTheme'), icon: mode === 'dark' ? <Sun size={16} aria-hidden /> : <MoonStar size={16} aria-hidden />, onSelect: toggle },
    'separator',
    { group: t('app.user.language') },
    ...LANGS.map((l): MenuItem => ({ label: t(`app.user.lang.${l}`), icon: current === l ? <Check size={16} aria-hidden /> : <Globe size={16} aria-hidden />, onSelect: () => { void i18n.changeLanguage(l) } })),
    'separator',
    { label: t('app.user.settings'), icon: <Settings size={16} aria-hidden />, onSelect: () => navigate(withPeriod('/settings/config', new URLSearchParams(search))) },
    { label: t('app.user.logout'), icon: <LogOut size={16} aria-hidden />, onSelect: () => { void logout() } },
  ]
  return (
    <Menu
      trigger={
        <button type="button" className={styles.trigger} aria-label={t('app.user.menu')}>
          <Avatar id={getUserId() || name} name={name} />
          <span className={styles.name}>{name}</span>
          <ChevronDown size={14} strokeWidth={1.6} aria-hidden className={styles.chev} />
        </button>
      }
      items={items}
    />
  )
}
