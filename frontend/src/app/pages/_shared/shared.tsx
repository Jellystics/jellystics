import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Database, Inbox } from 'lucide-react'
import { Card, CardBody, CardHeader, EmptyState, ErrorState, Skeleton } from '@/ui'
import styles from './analytics.module.css'

/** Minimal shape of a React Query result used by the analysis cards. */
export interface QueryLike<T> { data?: T; isError: boolean; refetch: () => unknown }

/**
 * Loading / error / empty states of one analysis card. Data is kept on
 * refetch (placeholderData), so the skeleton only shows on the first load.
 */
export function QueryCard<T>({ title, subtitle, query, isEmpty, empty, skeleton = 180, wide, children }: {
  title: string
  subtitle?: string
  query: QueryLike<T>
  isEmpty?: (d: T) => boolean
  /** Custom empty state (default: generic "no data for the period"). */
  empty?: ReactNode
  skeleton?: number
  wide?: boolean
  children: (data: T) => ReactNode
}) {
  const { t } = useTranslation()
  const wrap = (node: ReactNode) => <div className={wide ? styles.wide : styles.cell}>{node}</div>
  const shell = (body: ReactNode) => wrap(<Card><CardHeader title={title} subtitle={subtitle} /><CardBody>{body}</CardBody></Card>)
  if (query.data === undefined) {
    if (query.isError) return shell(<ErrorState title={t('app.error.generic')} onRetry={() => { query.refetch() }} />)
    return shell(<div aria-busy="true"><Skeleton height={skeleton} radius="md" /></div>)
  }
  if (isEmpty?.(query.data)) return shell(empty ?? <NoData />)
  return wrap(children(query.data))
}

export function NoData() {
  const { t } = useTranslation()
  return <EmptyState icon={<Inbox size={20} strokeWidth={1.6} />} title={t('app.analytics.noData')} description={t('app.analytics.noDataHint')} />
}

/**
 * Empty state of a capture-fed widget over a period starting at `from`
 * (YYYY-MM-DD): the "collected from" message only when the capture had not
 * started yet at the beginning of the period, else the usual "no data".
 */
export function CaptureEmpty({ since, from }: { since: string | null; from: string }) {
  return since !== null && since <= from ? <NoData /> : <CollectedEmpty since={since} />
}

/** Empty state of the widgets fed by the playback capture (migration 011). */
export function CollectedEmpty({ since }: { since: string | null }) {
  const { t, i18n } = useTranslation()
  const date = since ? new Intl.DateTimeFormat(i18n.resolvedLanguage ?? 'fr', { day: 'numeric', month: 'long', year: 'numeric' }).format(new Date(`${since}T12:00:00`)) : null
  return (
    <EmptyState
      icon={<Database size={20} strokeWidth={1.6} />}
      title={t('app.analytics.notCollected')}
      description={date ? t('app.analytics.collectedSince', { date }) : t('app.analytics.collectedNext')}
    />
  )
}

