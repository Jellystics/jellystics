import { useTranslation } from 'react-i18next'
import { GitCompare, X } from 'lucide-react'
import { useUser, useUsers, type UserDetail, type UserRow } from '@/api'
import { useFormat } from '@/lib/format'
import { useSearchParam } from '@/lib/useUrlState'
import { avatarColor } from '@/ui/Avatar'
import { AppLink, Avatar, Button, Card, CardBody, CardHeader, EmptyState, ErrorState, Select, Skeleton } from '@/ui'
import { userHref } from '../history/links'
import s from './user.module.css'

const seriesShare = (r?: UserRow) => {
  if (!r) return null
  const total = r.byType.movie + r.byType.series + r.byType.music
  return total ? r.byType.series / total : null
}

/** Two-user comparison (?vs=:otherId — not ?compare=, which is the global period-comparison flag): useUser twice + the users list for the type split. */
export function ComparePanel({ user }: { user: UserDetail }) {
  const { t } = useTranslation()
  const f = useFormat()
  const [compare, setCompare] = useSearchParam('vs')
  const other = useUser(compare && compare !== user.id ? compare : undefined)
  const users = useUsers()
  const list = Array.isArray(users.data) ? users.data : []
  const options = list.filter((u) => u.id !== user.id).map((u) => ({ value: u.id, label: u.name }))
  const b = other.data && !Array.isArray(other.data) && other.data.totals ? other.data : undefined
  const rowA = list.find((u) => u.id === user.id), rowB = list.find((u) => u.id === compare)

  const metrics: { label: string; a: number | null; b: number | null; fmt: (v: number) => string }[] = b ? [
    { label: t('app.userPage.compare.time'), a: user.totals.watchSeconds, b: b.totals.watchSeconds, fmt: f.duration },
    { label: t('app.userPage.compare.views'), a: user.totals.views, b: b.totals.views, fmt: f.number },
    { label: t('app.userPage.compare.completion'), a: user.totals.completion, b: b.totals.completion, fmt: f.percent },
    { label: t('app.userPage.compare.regularity'), a: user.profile.regularity, b: b.profile.regularity, fmt: f.percent },
    { label: t('app.userPage.compare.seriesShare'), a: seriesShare(rowA), b: seriesShare(rowB), fmt: f.percent },
  ] : []

  return (
    <Card>
      <CardHeader
        title={t('app.userPage.compare.title')}
        subtitle={t('app.userPage.compare.subtitle')}
        action={
          <div className={s.compareActions}>
            {options.length > 0 && <Select aria-label={t('app.userPage.compare.pick')} value={compare ?? ''} onValueChange={(v) => setCompare(v)} options={options} />}
            {compare && <Button size="sm" variant="ghost" iconOnly aria-label={t('app.userPage.compare.clear')} icon={<X size={14} />} onClick={() => setCompare(undefined)} />}
          </div>
        }
      />
      <CardBody>
        {!compare ? (
          <EmptyState icon={<GitCompare size={20} strokeWidth={1.6} />} title={t('app.userPage.compare.empty')} description={t('app.userPage.compare.emptyHint')} />
        ) : other.isError && !b ? (
          <ErrorState title={t('app.userPage.compare.error')} onRetry={() => other.refetch()} />
        ) : !b ? (
          <Skeleton height={160} radius="md" />
        ) : (
          <>
            <div className={s.compareHead}>
              <span className={s.compareWho}><Avatar id={user.id} name={user.name} size="sm" />{user.name}</span>
              <span className="muted">{t('app.userPage.compare.vs')}</span>
              <AppLink to={userHref(b.id)} className={s.compareWho}>{b.name}<Avatar id={b.id} name={b.name} size="sm" /></AppLink>
            </div>
            <div className={s.tabBody}>
              {metrics.map((m) => {
                const total = (m.a ?? 0) + (m.b ?? 0)
                const ratio = total > 0 ? (m.a ?? 0) / total : 0.5
                return (
                  <div key={m.label} className={s.compareRow}>
                    <div className={s.compareLine}>
                      <span className="num">{m.a === null ? '—' : m.fmt(m.a)}</span>
                      <span className="muted">{m.label}</span>
                      <span className="num">{m.b === null ? '—' : m.fmt(m.b)}</span>
                    </div>
                    <div className={s.compareBar} aria-hidden="true">
                      <span style={{ flex: ratio, background: avatarColor(user.id) }} />
                      <span style={{ flex: 1 - ratio, background: avatarColor(b.id) }} />
                    </div>
                  </div>
                )
              })}
            </div>
          </>
        )}
      </CardBody>
    </Card>
  )
}
