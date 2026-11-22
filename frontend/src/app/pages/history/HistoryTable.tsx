import { useTranslation } from 'react-i18next'
import type { ColumnDef } from '@tanstack/react-table'
import { History as HistoryIcon } from 'lucide-react'
import type { HistoryRow } from '@/api'
import { itemImageUrl } from '@/api/live'
import { useFormat } from '@/lib/format'
import { AppLink, Art, Avatar, Badge, Card, DataTable, EmptyState, ErrorState, Meter, MethodBadge, TypeKey, ViewBadge } from '@/ui'
import { dateOf, familyOf, itemHref, methodOf, userHref, wallClock, wallDate } from './links'
import { useHistoryRows, type FilterKey, type HistoryUrlState } from './useHistoryFilters'
import l from '../overview/layout.module.css'
import s from './history.module.css'

function subtitle(r: HistoryRow, t: (k: string, o?: Record<string, unknown>) => string): string {
  const it = r.item
  if (it.kind === 'episode') {
    const ep = it.season != null && it.episode != null ? t('app.history.episodeShort', { season: it.season, episode: it.episode }) : ''
    return [it.parentName, ep].filter(Boolean).join(' · ')
  }
  if (it.kind === 'track') return it.parentName ?? ''
  return t('app.type.movie')
}

/** Paginated, server-sorted history table (used by /history and the user page). */
export function HistoryTable({ state, hideUser }: { state: HistoryUrlState; hideUser?: boolean }) {
  const { t } = useTranslation()
  const f = useFormat()
  const q = useHistoryRows(state)
  const data = q.data && Array.isArray(q.data.rows) ? q.data : undefined
  const filterButton = (key: FilterKey, value: string, label: string) => (
    value ? <button type="button" className={s.cellFilter} onClick={() => state.set(key, value)} title={t('app.history.filterBy', { value })}>{label}</button> : <span className="muted">—</span>
  )

  const columns: ColumnDef<HistoryRow>[] = [
    {
      id: 'startedAt', accessorFn: (r) => r.startedAt, header: t('app.history.col.date'),
      cell: ({ row: { original: r } }) => <span className={`num ${s.nowrap} muted`}>{f.day(dateOf(wallDate(r.startedAt)))} · {wallClock(r.startedAt)}</span>,
    },
    ...(hideUser ? [] : [{
      id: 'user', header: t('app.history.col.user'), enableSorting: false,
      cell: ({ row: { original: r } }) => (
        <div className={l.cellMedia}><Avatar id={r.user.id} name={r.user.name} size="sm" /><AppLink to={userHref(r.user.id)}>{r.user.name}</AppLink></div>
      ),
    } satisfies ColumnDef<HistoryRow>]),
    {
      id: 'content', header: t('app.history.col.content'), enableSorting: false,
      cell: ({ row: { original: r } }) => {
        const fam = familyOf(r.mediaType)
        const title = r.item.name
        return (
          <div className={l.cellMedia}>
            <Art title={r.item.parentName || title} kind={fam === 'music' ? 'square' : 'poster'} src={itemImageUrl(r.item.kind === 'movie' ? r.item.id : r.item.parentId || r.item.id, 64)} />
            <div className={l.ellipsis}>
              <AppLink to={itemHref(r.item)} className={s.title}>{title}</AppLink>
              <div className={l.cellSub}><TypeKey type={fam} label={subtitle(r, t) || t(`app.type.${fam}`)} /></div>
            </div>
          </div>
        )
      },
    },
    {
      id: 'watchSeconds', accessorFn: (r) => r.watchSeconds, header: t('app.history.col.watched'), meta: { numeric: true },
      cell: ({ row: { original: r } }) => f.duration(r.watchSeconds),
    },
    {
      id: 'completion', accessorFn: (r) => r.completion ?? -1, header: t('app.history.col.completion'),
      cell: ({ row: { original: r } }) => r.completion === null ? <span className="muted">—</span> : (
        <div className={s.completion}>
          <Meter value={r.completion} thin label={t('app.history.col.completion')} />
          <span className="num muted">{f.percent(r.completion)}</span>
        </div>
      ),
    },
    {
      id: 'device', header: t('app.history.col.device'), enableSorting: false,
      cell: ({ row: { original: r } }) => (
        <div className={s.device}>
          <div>{filterButton('client', r.client, r.client)}</div>
          <div className={l.cellSub}>{filterButton('device', r.device, r.device)}</div>
        </div>
      ),
    },
    {
      id: 'method', header: t('app.history.col.method'), enableSorting: false,
      cell: ({ row: { original: r } }) => { const m = methodOf(r.method); return m ? <MethodBadge method={m} /> : <Badge>{r.method || '—'}</Badge> },
    },
    {
      id: 'view', header: () => <span className="sr-only">{t('app.history.col.view')}</span>, enableSorting: false,
      cell: ({ row: { original: r } }) => <ViewBadge view={r.isView} />,
    },
  ]

  if (q.isError && !data) return <Card><ErrorState title={t('app.history.error')} onRetry={() => q.refetch()} /></Card>
  return (
    <Card>
      <DataTable
        columns={columns}
        data={data?.rows ?? []}
        loading={!data}
        sort={{ id: state.sort, desc: state.desc, onChange: state.setSort }}
        pagination={data ? { page: data.page, pageSize: data.pageSize, total: data.total, onPageChange: state.setPage, onPageSizeChange: state.setPageSize } : undefined}
        empty={<EmptyState icon={<HistoryIcon size={20} strokeWidth={1.6} />} title={t('app.history.empty')} description={t('app.history.emptyHint')} />}
      />
    </Card>
  )
}
