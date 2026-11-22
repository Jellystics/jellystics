import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import type { ColumnDef } from '@tanstack/react-table'
import { Users as UsersIcon } from 'lucide-react'
import { useUsers, type UserRow } from '@/api'
import { useUserRetention, useUserShare } from '@/api/usersExtra'
import { ChartFrame, Matrix, Sparkline } from '@/charts'
import { useFormat } from '@/lib/format'
import { useSearchParams } from 'react-router-dom'
import { AppLink, Avatar, Card, DataTable, EmptyState, ErrorState, PageHeader, SegBar, Skeleton } from '@/ui'
import { dateOf, userHref } from '../history/links'
import l from '../overview/layout.module.css'
import s from './users.module.css'

type SortId = 'watchSeconds' | 'views' | 'lastActivity' | 'name'
const SORTS: SortId[] = ['watchSeconds', 'views', 'lastActivity', 'name']

function sortRows(rows: UserRow[], id: SortId, desc: boolean): UserRow[] {
  const val = (r: UserRow): number | string => id === 'name' ? r.name.toLowerCase() : id === 'lastActivity' ? (r.lastActivity ? Date.parse(r.lastActivity) : 0) : r[id]
  return [...rows].sort((a, b) => { const x = val(a), y = val(b); const c = x < y ? -1 : x > y ? 1 : 0; return desc ? -c : c })
}

export default function UsersPage() {
  const { t } = useTranslation()
  const f = useFormat()
  const q = useUsers()
  const [sp, setSp] = useSearchParams()
  const sortParam = sp.get('sort'), descParam = sp.get('desc')
  const onSort = (x: { id: string; desc: boolean }) => setSp((prev) => {
    const out = new URLSearchParams(prev)
    if (x.id === 'watchSeconds') out.delete('sort'); else out.set('sort', x.id)
    if (x.desc) out.delete('desc'); else out.set('desc', '0')
    return out
  }, { replace: true })
  const sort: SortId = SORTS.includes(sortParam as SortId) ? (sortParam as SortId) : 'watchSeconds'
  const desc = descParam !== '0'
  const data = Array.isArray(q.data) ? q.data : undefined
  const rows = useMemo(() => (data ? sortRows(data, sort, desc) : []), [data, sort, desc])
  const active = data?.filter((u) => u.watchSeconds > 0).length ?? 0

  const columns: ColumnDef<UserRow>[] = [
    {
      id: 'name', accessorFn: (r) => r.name, header: t('app.users.col.user'),
      cell: ({ row: { original: r } }) => <div className={l.cellMedia}><Avatar id={r.id} name={r.name} /><span className={s.name}>{r.name}</span></div>,
    },
    { id: 'watchSeconds', accessorFn: (r) => r.watchSeconds, header: t('app.users.col.time'), meta: { numeric: true }, cell: ({ row: { original: r } }) => f.duration(r.watchSeconds) },
    { id: 'views', accessorFn: (r) => r.views, header: t('app.users.col.views'), meta: { numeric: true }, cell: ({ row: { original: r } }) => f.number(r.views) },
    {
      id: 'split', header: t('app.users.col.split'), enableSorting: false,
      cell: ({ row: { original: r } }) => {
        const total = r.byType.movie + r.byType.series + r.byType.music
        return total === 0 ? <span className="muted">—</span> : (
          <div className={s.split}>
            <SegBar thin parts={(['series', 'movie', 'music'] as const).map((k) => ({ label: t(`app.type.${k}`), value: r.byType[k], color: `var(--t-${k})`, display: f.duration(r.byType[k]) }))} />
          </div>
        )
      },
    },
    {
      id: 'lastActivity', accessorFn: (r) => r.lastActivity ?? '', header: t('app.users.col.last'),
      cell: ({ row: { original: r } }) => <span className="muted">{r.lastActivity ? f.relative(new Date(r.lastActivity)) : t('app.users.never')}</span>,
    },
    {
      id: 'trend', header: t('app.users.col.trend'), enableSorting: false,
      cell: ({ row: { original: r } }) => r.trend?.some((v) => v > 0) ? <Sparkline values={r.trend} width={72} height={22} /> : <span className="muted">—</span>,
    },
  ]

  return (
    <div className={l.page}>
      <PageHeader title={t('app.users.title')} subtitle={data ? t('app.users.subtitle', { count: data.length, active }) : undefined} />
      {q.isError && !data ? (
        <Card><ErrorState title={t('app.users.error')} onRetry={() => q.refetch()} /></Card>
      ) : (
        <Card>
          <DataTable
            columns={columns}
            data={rows}
            loading={!data}
            rowHref={(r) => userHref(r.id)}
            sort={{ id: sort, desc, onChange: onSort }}
            empty={<EmptyState icon={<UsersIcon size={20} strokeWidth={1.6} />} title={t('app.users.empty')} />}
          />
        </Card>
      )}
      <div className={l.g2}>
        <RetentionCard />
        <ShareCard />
      </div>
    </div>
  )
}

function RetentionCard() {
  const { t } = useTranslation()
  const f = useFormat()
  const q = useUserRetention()
  const data = q.data && Array.isArray(q.data.rows) ? q.data : undefined
  if (q.isError && !data) return <Card><ErrorState title={t('app.users.retention.error')} onRetry={() => q.refetch()} /></Card>
  if (!data) return <Skeleton height={240} radius="lg" />
  const cols = Array.from({ length: data.offsets }, (_, k) => (k === 0 ? t('app.users.retention.offset0') : t('app.users.retention.offset', { k })))
  const rowLabels = data.rows.map((r) => t('app.users.retention.week', { date: f.day(dateOf(r.week)) }))
  return (
    <ChartFrame
      title={t('app.users.retention.title')}
      subtitle={t('app.users.retention.subtitle')}
      csvName="users-retention"
      table={{ columns: [t('app.users.retention.cohort'), t('app.users.retention.size'), ...cols], rows: data.rows.map((r, i) => [rowLabels[i], r.cohort, ...r.retained.map((v) => v ?? '')]) }}
    >
      {data.rows.length === 0
        ? <EmptyState title={t('app.users.retention.empty')} />
        : <Matrix rows={rowLabels} cols={cols} values={data.rows.map((r) => r.retained)} unit={t('app.users.retention.unit')} labelWidth={96} />}
    </ChartFrame>
  )
}

function ShareCard() {
  const { t } = useTranslation()
  const f = useFormat()
  const q = useUserShare()
  const data = q.data && Array.isArray(q.data.users) ? q.data : undefined
  if (q.isError && !data) return <Card><ErrorState title={t('app.users.share.error')} onRetry={() => q.refetch()} /></Card>
  if (!data) return <Skeleton height={240} radius="lg" />
  return (
    <ChartFrame
      title={t('app.users.share.title')}
      subtitle={t(`app.users.share.subtitle.${data.bucket}`)}
      csvName="users-share"
      table={{
        columns: [t('app.users.col.user'), t('app.users.share.total'), ...data.periods],
        rows: data.users.map((u) => [u.name, f.percent(u.share), ...u.values.map((v) => f.percent(v))]),
      }}
    >
      {data.users.length === 0 ? <EmptyState title={t('app.users.share.empty')} /> : (
        <ul className={`${l.stack} ${l.plainList}`}>
          {data.users.map((u) => (
            <li key={u.id} className={s.shareRow}>
              <Avatar id={u.id} name={u.name} size="sm" />
              <AppLink to={userHref(u.id)} className={s.shareName}>{u.name}</AppLink>
              <span className={s.shareSpark}><Sparkline values={u.values.length > 1 ? u.values : [u.share, u.share]} width={200} height={24} /></span>
              <span className={`num ${s.sharePct}`}>{f.percent(u.share)}</span>
            </li>
          ))}
        </ul>
      )}
    </ChartFrame>
  )
}
