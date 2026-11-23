import { AlertTriangle, RotateCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from './Button'
import { EmptyState } from './EmptyState'

export function ErrorState({ title, description, onRetry }: { title: string; description?: string; onRetry?: () => void }) {
  const { t } = useTranslation()
  return (
    <EmptyState
      role="alert"
      icon={<AlertTriangle size={20} strokeWidth={1.6} />}
      title={title}
      description={description}
      action={onRetry && <Button size="sm" icon={<RotateCw size={16} strokeWidth={1.6} />} onClick={onRetry}>{t('app.action.retry')}</Button>}
    />
  )
}
