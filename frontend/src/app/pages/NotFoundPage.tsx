import { useTranslation } from 'react-i18next'
import { SearchX } from 'lucide-react'
import { ButtonLink, EmptyState } from '@/ui'

export function NotFoundPage() {
  const { t } = useTranslation()
  return (
    <EmptyState
      headingLevel={1}
      icon={<SearchX size={20} strokeWidth={1.6} />}
      title={t('app.notFound.title')}
      description={t('app.notFound.description')}
      action={<ButtonLink to="/" size="sm">{t('app.notFound.home')}</ButtonLink>}
    />
  )
}
