import { useTranslation } from 'react-i18next'
import { useQuery } from '@tanstack/react-query'
import { Bug, ExternalLink, RotateCw } from 'lucide-react'
import { Badge, Button, buttonClass, Card, CardBody, CardHeader, ErrorState, Skeleton } from '@/ui'
import { apiFetch } from '@/lib/api'
import styles from '../SettingsPage.module.css'

const REPO_URL = 'https://github.com/Jellystics/Jellystics'

interface VersionInfo { current_version?: string; latest_version?: string; message?: string; update_available?: boolean }

export function AboutSection() {
  const { t } = useTranslation()
  const q = useQuery({ queryKey: ['settings', 'about'], queryFn: () => apiFetch<VersionInfo>('/api/CheckForUpdates'), retry: false })
  const info = q.data && !Array.isArray(q.data) ? q.data : undefined
  const current = info?.current_version ? `v${info.current_version}` : '—'
  const upToDate = !!info && !info.update_available && info.current_version !== '0.0.0'

  return (
    <div className={styles.stack}>
      <Card>
        <CardHeader title="Jellystics" subtitle={t('app.settings.about.description')} />
        <CardBody>
          {q.isError ? <ErrorState title={t('app.settings.about.checkError')} onRetry={() => q.refetch()} /> : (
            <div className={styles.form}>
              <div className={styles.row}>
                <span className={styles.grow}>{t('app.settings.about.installed')}</span>
                {q.isLoading ? <Skeleton width={64} height={20} /> : <span className={styles.mono}>{current}</span>}
              </div>
              <div className={styles.row}>
                <span className={styles.grow}>{t('app.settings.about.latest')}</span>
                {q.isLoading ? <Skeleton width={64} height={20} /> : <span className={styles.mono}>{info?.latest_version ?? '—'}</span>}
              </div>
              {info && (
                <div className={styles.row} role="status">
                  {info.update_available ? <Badge tone="warning">{t('app.settings.about.updateAvailable')}</Badge> : upToDate ? <Badge tone="good">{t('app.settings.about.upToDate')}</Badge> : null}
                  <span className={styles.status}>{info.message}</span>
                  {info.update_available && (
                    <a className={buttonClass({ variant: 'primary', size: 'sm' })} href={`${REPO_URL}/releases/latest`} target="_blank" rel="noopener noreferrer">{t('app.settings.about.download')}</a>
                  )}
                </div>
              )}
              <div><Button size="sm" icon={<RotateCw size={16} strokeWidth={1.6} />} onClick={() => q.refetch()} disabled={q.isFetching}>{t('app.settings.about.check')}</Button></div>
            </div>
          )}
        </CardBody>
      </Card>
      <Card>
        <CardHeader title={t('app.settings.about.links')} />
        <CardBody>
          <div className={styles.actions}>
            <a className={buttonClass()} href={REPO_URL} target="_blank" rel="noopener noreferrer"><ExternalLink size={16} strokeWidth={1.6} aria-hidden />{t('app.settings.about.github')}</a>
            <a className={buttonClass()} href={`${REPO_URL}/issues/new`} target="_blank" rel="noopener noreferrer"><Bug size={16} strokeWidth={1.6} aria-hidden />{t('app.settings.about.openIssue')}</a>
          </div>
        </CardBody>
      </Card>
    </div>
  )
}
