import { Globe, LogOut, MoonStar, Settings, Sun, Check } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Avatar, Button, Menu, type MenuItem } from '@/ui'
import { getUserId, getUsername, logout } from '@/lib/auth'
import { useThemeMode } from '@/lib/themeMode'
import styles from './Sidebar.module.css'

const LANGS = ['fr', 'en', 'es'] as const

export function UserMenu() {
  const { t, i18n } = useTranslation()
  const { mode, toggle } = useThemeMode()
  const navigate = useNavigate()
  const name = getUsername() || t('app.user.guest')
  const current = i18n.resolvedLanguage || i18n.language
  const items: MenuItem[] = [
    { label: mode === 'dark' ? t('app.user.lightTheme') : t('app.user.darkTheme'), icon: mode === 'dark' ? <Sun size={16} aria-hidden /> : <MoonStar size={16} aria-hidden />, onSelect: toggle },
    'separator',
    { group: t('app.user.language') },
    ...LANGS.map((l): MenuItem => ({ label: t(`app.user.lang.${l}`), icon: current === l ? <Check size={16} aria-hidden /> : <Globe size={16} aria-hidden />, onSelect: () => { void i18n.changeLanguage(l) } })),
    'separator',
    { label: t('app.user.settings'), icon: <Settings size={16} aria-hidden />, onSelect: () => navigate('/settings/config') },
    { label: t('app.user.logout'), icon: <LogOut size={16} aria-hidden />, onSelect: () => { void logout() } },
  ]
  return (
    <div className={styles.foot}>
      <Avatar id={getUserId() || name} name={name} size="sm" />
      <span className={styles.user}>{name}</span>
      <Menu trigger={<Button variant="ghost" size="sm" iconOnly aria-label={t('app.user.menu')} icon={<Settings size={16} strokeWidth={1.6} aria-hidden />} />} items={items} />
    </div>
  )
}
