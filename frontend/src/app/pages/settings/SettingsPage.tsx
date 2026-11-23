import type { ReactNode } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { PageHeader, Tabs } from '@/ui'
import { ConfigSection } from './sections/ConfigSection'
import { TasksSection } from './sections/TasksSection'
import { BackupSection } from './sections/BackupSection'
import { LogsSection } from './sections/LogsSection'
import { WebhooksSection } from './sections/WebhooksSection'
import { ApiKeysSection } from './sections/ApiKeysSection'
import { AboutSection } from './sections/AboutSection'
import styles from './SettingsPage.module.css'

import { SETTINGS_SECTIONS, type SettingsSection } from './sections'

const VIEWS: Record<SettingsSection, () => ReactNode> = {
  config: () => <ConfigSection />,
  tasks: () => <TasksSection />,
  backup: () => <BackupSection />,
  logs: () => <LogsSection />,
  webhooks: () => <div className={styles.stack}><WebhooksSection /></div>,
  apikeys: () => <div className={styles.stack}><ApiKeysSection /></div>,
  about: () => <AboutSection />,
}

export function SettingsPage({ section }: { section: SettingsSection }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { search } = useLocation()
  return (
    <>
      <PageHeader title={t('app.nav.settings')} />
      <div className={styles.nav}>
        <Tabs
          idPrefix="settings-tab"
          value={section}
          onValueChange={(v) => navigate({ pathname: `/settings/${v}`, search }, { replace: true })}
          items={SETTINGS_SECTIONS.map((s) => ({ value: s, label: t(`app.settings.sections.${s}`) }))}
        />
      </div>
      <div role="tabpanel" id={`settings-panel-${section}`} aria-labelledby={`settings-tab-${section}`} tabIndex={0}>{VIEWS[section]()}</div>
    </>
  )
}

export default SettingsPage
