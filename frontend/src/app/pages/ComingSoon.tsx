import { useTranslation } from 'react-i18next'
import { Hourglass } from 'lucide-react'
import { EmptyState } from '@/ui'
import type { NavKey } from '@/app/shell/crumbs'

export function ComingSoon({ nav }: { nav?: NavKey }) {
  const { t } = useTranslation()
  return (
    <section data-nav={nav}>
      <EmptyState icon={<Hourglass size={20} strokeWidth={1.6} />} title={t('app.comingSoon.title')} description={t('app.comingSoon.description')} />
    </section>
  )
}
