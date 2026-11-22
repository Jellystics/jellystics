import { useTranslation } from 'react-i18next'
import { SearchX } from 'lucide-react'
import { AppLink } from '@/app/AppLink'
import { Button, EmptyState } from '@/ui'

export function NotFoundPage() {
  const { t } = useTranslation()
  return (
    <EmptyState
      icon={<SearchX size={20} strokeWidth={1.6} />}
      title={t('app.notFound.title')}
      description={t('app.notFound.description')}
      action={<AppLink to="/"><Button size="sm">{t('app.notFound.home')}</Button></AppLink>}
    />
  )
}
