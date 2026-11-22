import { Search } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Breadcrumbs, Button, PeriodPicker } from '@/ui'
import { useCrumbs, useShowPeriod } from './crumbs'
import { MobileNav } from './MobileNav'
import styles from './TopBar.module.css'

export function TopBar({ liveCount, onOpenCommand }: { liveCount: number; onOpenCommand: () => void }) {
  const { t } = useTranslation()
  const crumbs = useCrumbs()
  const showPeriod = useShowPeriod()
  return (
    <header className={styles.top}>
      <MobileNav liveCount={liveCount} />
      <div className={styles.crumbs}>{crumbs.length > 0 && <Breadcrumbs items={crumbs} />}</div>
      <Button variant="ghost" size="sm" iconOnly className={styles.searchBtn} aria-label={t('app.command.open')} onClick={onOpenCommand} icon={<Search size={16} strokeWidth={1.6} aria-hidden />} />
      {showPeriod && <PeriodPicker />}
    </header>
  )
}
