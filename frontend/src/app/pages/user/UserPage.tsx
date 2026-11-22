import { useTranslation } from 'react-i18next'
import { useParams } from 'react-router-dom'
import { Radio, Sparkles, UserX } from 'lucide-react'
import { ApiError } from '@/lib/api'
import { useUser, type UserDetail } from '@/api'
import { useLiveSessions } from '@/api/live'
import { useFormat } from '@/lib/format'
import { todayIn } from '@/lib/period'
import { useServerConfig } from '@/lib/serverConfig'
import { useSearchParam } from '@/lib/useUrlState'
import { useSetCrumbLabel } from '@/app/shell/crumbs'
import { Avatar, Badge, BarList, ButtonLink, Card, CardBody, CardHeader, EmptyState, EntityHeader, ErrorState, Skeleton, Tabs } from '@/ui'
import { FilterBar } from '../history/FilterBar'
import { HistoryTable } from '../history/HistoryTable'
import { useHistoryFilters, useHistoryRows } from '../history/useHistoryFilters'
import { dateOf } from '../history/links'
import { UserOverview } from './UserOverview'
import l from '../overview/layout.module.css'
import s from './user.module.css'

const TABS = ['overview', 'history', 'taste'] as const
type Tab = (typeof TABS)[number]
const uniq = (xs: string[]) => [...new Set(xs.filter(Boolean))].sort((a, b) => a.localeCompare(b))

function UserHistory({ userId }: { userId: string }) {
  const state = useHistoryFilters({ userId })
  const page = useHistoryRows(state)
  const rows = page.data && Array.isArray(page.data.rows) ? page.data.rows : []
  return (
    <div className={s.tabBody}>
      <FilterBar state={state} hideUser clients={uniq(rows.map((r) => r.client))} devices={uniq(rows.map((r) => r.device))} />
      <HistoryTable state={state} hideUser />
    </div>
  )
}

function UserTaste({ user }: { user: UserDetail }) {
  const { t } = useTranslation()
  const f = useFormat()
  return (
    <div className={l.g3}>
      {(['series', 'movie', 'music'] as const).map((k) => {
        const genres = user.genres?.[k] ?? []
        return (
          <Card key={k}>
            <CardHeader title={t(`app.userPage.taste.${k}`)} subtitle={t('app.userPage.taste.subtitle')} />
            <CardBody>
              {genres.length === 0 ? <EmptyState title={t('app.userPage.taste.empty')} /> : (
                <BarList
                  labelHeader={t('app.userPage.taste.genre')}
                  valueLabel={t('app.userPage.tiles.time')}
                  format={f.duration}
                  rows={genres.slice(0, 10).map((g) => ({ key: g.name, label: g.name, value: g.watchSeconds, fill: `color-mix(in srgb, var(--t-${k}) 18%, transparent)` }))}
                />
              )}
            </CardBody>
          </Card>
        )
      })}
    </div>
  )
}

/** /users/:id — tabs overview | history | taste (?tab=), compare (?compare=). */
export default function UserPage() {
  const { t } = useTranslation()
  const f = useFormat()
  const { id } = useParams()
  const q = useUser(id)
  const user = q.data && !Array.isArray(q.data) && q.data.totals ? q.data : undefined
  useSetCrumbLabel(user?.name)
  const [tabParam, setTab] = useSearchParam('tab', 'overview', { push: true })
  const tab: Tab = TABS.includes(tabParam as Tab) ? (tabParam as Tab) : 'overview'
  const live = useLiveSessions()
  const watching = (live.data ?? []).find((s) => s.UserId === id)
  const tz = useServerConfig().data?.timezone ?? Intl.DateTimeFormat().resolvedOptions().timeZone
  const year = todayIn(tz).slice(0, 4)

  if (!user && q.isError) {
    const notFound = q.error instanceof ApiError && q.error.status === 404
    return (
      <div className={l.page}>
        <Card>
          {notFound
            ? <EmptyState icon={<UserX size={20} strokeWidth={1.6} />} title={t('app.userPage.notFound')} role="alert" />
            : <ErrorState title={t('app.userPage.error')} onRetry={() => q.refetch()} />}
        </Card>
      </div>
    )
  }
  if (!user) {
    return (
      <div className={l.page}>
        <div className={l.row}><Skeleton width={56} height={56} radius="lg" /><div className={l.grow}><Skeleton width="30%" height={24} /><div className={l.spacer} /><Skeleton width="50%" /></div></div>
        <Skeleton height={36} />
        <Skeleton height={96} radius="lg" />
        <Skeleton height={180} radius="lg" />
      </div>
    )
  }

  const meta = [
    user.memberSince ? t('app.userPage.memberSince', { date: f.day(dateOf(user.memberSince.slice(0, 10))) }) : null,
    user.favoriteDevice ? t('app.userPage.favoriteDevice', { device: user.favoriteDevice }) : null,
    watching ? <Badge tone="critical" icon={<Radio size={12} aria-hidden="true" />}>{t('app.userPage.watching', { title: watching.NowPlayingItem?.SeriesName || watching.NowPlayingItem?.Name })}</Badge> : null,
  ].filter(Boolean) as React.ReactNode[]

  return (
    <div className={l.page}>
      <EntityHeader
        art={<Avatar id={user.id} name={user.name} size="lg" />}
        title={user.name}
        meta={meta}
        actions={<ButtonLink size="sm" to={`/users/${encodeURIComponent(user.id)}/recap/${year}`}><Sparkles size={14} aria-hidden="true" />{t('app.userPage.recap', { year })}</ButtonLink>}
      />
      <Tabs
        value={tab}
        onValueChange={setTab}
        items={[
          { value: 'overview', label: t('app.userPage.tabs.overview') },
          { value: 'history', label: t('app.userPage.tabs.history') },
          { value: 'taste', label: t('app.userPage.tabs.taste') },
        ]}
      />
      {tab === 'overview' && <UserOverview user={user} />}
      {tab === 'history' && <UserHistory userId={user.id} />}
      {tab === 'taste' && <UserTaste user={user} />}
    </div>
  )
}
