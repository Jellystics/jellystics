import type { ReactNode } from 'react'
import type { UseQueryResult } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { SearchX } from 'lucide-react'
import type { ColumnDef } from '@tanstack/react-table'
import { AppLink, Avatar, Card, DataTable, EmptyState, ErrorState, MediaTypeBadge, Meter, MethodBadge, Skeleton, StatStrip, ViewBadge } from '@/ui'
import { useFormat } from '@/lib/format'
import { entityPath, playMethodOf } from '@/lib/routes'
import type { HistoryRow, LibraryRef, RefUser } from '@/api'
import { formatWhen, isNotFound } from './entity'
import styles from './entity.module.css'

/** Loading / error / not-found wrapper of an entity query (keeps data on refetch). */
export function QueryView<T>({ query, skeleton, children }: { query: UseQueryResult<T>; skeleton: ReactNode; children: (data: T) => ReactNode }) {
  const { t } = useTranslation()
  if (query.data !== undefined) return <>{children(query.data)}</>
  if (query.isError) {
    if (isNotFound(query.error)) return <EmptyState headingLevel={1} icon={<SearchX size={20} strokeWidth={1.6} />} title={t('app.entity.notFound')} description={t('app.entity.notFoundHint')} />
    return <ErrorState title={t('app.entity.error')} onRetry={() => { void query.refetch() }} />
  }
  return <>{skeleton}</>
}

/** Kicker of an entity header: media type badge + link to its library. */
export function LibraryKicker({ type, library, children }: { type: 'movie' | 'series' | 'music'; library: LibraryRef | null | undefined; children?: ReactNode }) {
  return (
    <span className={styles.row}>
      <MediaTypeBadge type={type} />
      {library && <AppLink to={`/libraries/${encodeURIComponent(library.id)}`} className={styles.subLink}>{library.name}</AppLink>}
      {children}
    </span>
  )
}

/** Chart / list body of a secondary query: skeleton while loading, retry on error. */
export function SubQuery({ query, height, empty, children }: { query: Pick<UseQueryResult, 'isPending' | 'isError' | 'refetch'>; height: number; empty: ReactNode | null; children: ReactNode }) {
  const { t } = useTranslation()
  if (query.isError) return <ErrorState title={t('app.entity.error')} onRetry={() => { void query.refetch() }} />
  if (query.isPending) return <Skeleton height={height} radius="md" />
  return <>{empty ?? children}</>
}

/** Skeleton shaped like an entity page: header, tile strip, two cards, table. */
export function EntitySkeleton({ tiles = 4, header = true }: { tiles?: number; header?: boolean }) {
  return (
    <div className={styles.stack} aria-busy="true">
      {header && (
        <div className={styles.skHead}>
          <Skeleton width={120} height={180} radius="lg" />
          <div className={styles.skHeadText}><Skeleton width={90} /><Skeleton width="50%" height={28} /><Skeleton width="35%" /></div>
        </div>
      )}
      <Card>
        <StatStrip columns={tiles}>
          {Array.from({ length: tiles }, (_, i) => <div key={i} className={styles.skTile}><Skeleton width="50%" /><Skeleton width="40%" height={28} /></div>)}
        </StatStrip>
      </Card>
      <div className={styles.g2}>
        <Card><div className={styles.skCard}><Skeleton width="40%" /><Skeleton height={150} radius="md" /></div></Card>
        <Card><div className={styles.skCard}><Skeleton width="40%" /><Skeleton height={150} radius="md" /></div></Card>
      </div>
    </div>
  )
}

/** Thin meter + percentage (completion, retention, skip). */
export function RatioCell({ value, tone = 'accent', width = 90, label }: { value: number | null | undefined; tone?: 'accent' | 'serious' | 'muted' | 'series'; width?: number; label: string }) {
  const f = useFormat()
  if (value === null || value === undefined) return <span className="muted">—</span>
  return (
    <span className={styles.ratio}>
      <span style={{ width }}><Meter value={value} tone={tone} thin label={label} /></span>
      <span className={`num muted ${styles.ratioText}`}>{f.percent(value)}</span>
    </span>
  )
}

/** Skip rate cell: serious tone above 30 %. */
export function SkipCell({ rate }: { rate: number | null | undefined }) {
  const { t } = useTranslation()
  return <RatioCell value={rate} width={64} tone={rate != null && rate > 0.3 ? 'serious' : 'muted'} label={t('app.entity.skip')} />
}

/** Completion cell: serious tone below 70 %. */
export function CompletionCell({ value, width }: { value: number | null | undefined; width?: number }) {
  const { t } = useTranslation()
  return <RatioCell value={value} width={width} tone={value != null && value < 0.7 ? 'serious' : 'accent'} label={t('app.entity.completion')} />
}

export function UserCell({ user }: { user: RefUser }) {
  return (
    <span className={styles.cellMedia}>
      <Avatar id={user.id} name={user.name} size="sm" />
      <AppLink to={`/users/${encodeURIComponent(user.id)}`} className={styles.link}>{user.name}</AppLink>
    </span>
  )
}

/** Playback history table (entity pages): date, user, item, watched, completion, device, method, view. */
export function HistoryTable({ rows, showItem = true, pagination, loading }: {
  rows: HistoryRow[]
  showItem?: boolean
  pagination?: { page: number; pageSize: number; total: number; onPageChange(p: number): void }
  loading?: boolean
}) {
  const { t } = useTranslation()
  const f = useFormat()
  const columns: ColumnDef<HistoryRow>[] = [
    { id: 'when', header: t('app.entity.col.when'), cell: ({ row }) => <span className="num muted">{formatWhen(row.original.startedAt, f.locale)}</span> },
    { id: 'user', header: t('app.entity.col.user'), cell: ({ row }) => <UserCell user={row.original.user} /> },
    ...(showItem ? [{
      id: 'item', header: t('app.entity.col.item'),
      cell: ({ row }) => {
        const it = row.original.item
        const code = it.kind === 'episode' && it.season != null && it.episode != null ? t('app.entity.episodeCode', { season: it.season, episode: it.episode }) : null
        return (
          <span className={styles.itemCell}>
            <AppLink to={entityPath(it.kind, it.id, it.parentId)} className={styles.link}>{it.name}</AppLink>
            {code && <span className="muted"> · {code}</span>}
          </span>
        )
      },
    } satisfies ColumnDef<HistoryRow>] : []),
    { id: 'watched', header: t('app.entity.col.watched'), meta: { numeric: true }, cell: ({ row }) => f.duration(row.original.watchSeconds) },
    { id: 'completion', header: t('app.entity.col.completion'), cell: ({ row }) => <CompletionCell value={row.original.completion} width={72} /> },
    { id: 'device', header: t('app.entity.col.device'), cell: ({ row }) => <span className="ink2">{[row.original.client, row.original.device].filter(Boolean).join(' · ')}</span> },
    {
      id: 'method', header: t('app.entity.col.method'),
      cell: ({ row }) => { const m = playMethodOf(row.original.method); return m ? <MethodBadge method={m} /> : <span className="muted">—</span> },
    },
    { id: 'view', header: t('app.entity.col.type'), cell: ({ row }) => <ViewBadge view={row.original.isView} /> },
  ]
  return <DataTable columns={columns} data={rows} pagination={pagination} loading={loading} empty={<EmptyState title={t('app.entity.noHistory')} />} />
}
