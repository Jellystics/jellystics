import { useEffect, useState } from 'react'
import * as RDialog from '@radix-ui/react-dialog'
import { Menu as MenuIcon, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/ui'
import { NavList } from './NavList'
import { UserMenu } from './UserMenu'
import { Brand } from './Brand'
import styles from './MobileNav.module.css'

export function MobileNav({ liveCount }: { liveCount: number }) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  // The drawer is mobile-only: close it when the viewport grows to the desktop layout.
  useEffect(() => {
    const mq = window.matchMedia('(min-width: 900px)')
    const onChange = (e: MediaQueryListEvent) => { if (e.matches) setOpen(false) }
    mq.addEventListener('change', onChange)
    return () => mq.removeEventListener('change', onChange)
  }, [])
  return (
    <RDialog.Root open={open} onOpenChange={setOpen}>
      <RDialog.Trigger asChild>
        <Button variant="ghost" size="sm" iconOnly className={styles.trigger} aria-label={t('app.nav.openMenu')} icon={<MenuIcon size={16} strokeWidth={1.6} aria-hidden />} />
      </RDialog.Trigger>
      <RDialog.Portal>
        <RDialog.Overlay className={styles.overlay} />
        <RDialog.Content className={styles.drawer} aria-describedby={undefined}>
          <div className={styles.head}>
            <RDialog.Title className={styles.title}><Brand />Jellystics</RDialog.Title>
            <RDialog.Close asChild><Button variant="ghost" size="sm" iconOnly aria-label={t('app.nav.closeMenu')} icon={<X size={16} strokeWidth={1.6} aria-hidden />} /></RDialog.Close>
          </div>
          <NavList liveCount={liveCount} onNavigate={() => setOpen(false)} />
          <UserMenu />
        </RDialog.Content>
      </RDialog.Portal>
    </RDialog.Root>
  )
}
