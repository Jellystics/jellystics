import { useEffect, useState } from 'react'
import { useParams, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ChevronLeft, ChevronRight, LayoutGrid, List } from 'lucide-react'
import type { ColumnDef } from '@tanstack/react-table'
import { AppLink, Art, Button, Card, DataTable, EmptyState, PageHeader, SearchInput, SegmentedControl, StatStrip, StatTile, Tabs } from '@/ui'
import { Sparkline } from '@/charts'
import { useFormat } from '@/lib/format'
import { useSearchParam } from '@/lib/useUrlState'
import { useHistory, useLibrary, type LibraryDetail, type LibraryItem } from '@/api'
import { useSetCrumbLabel } from '@/app/shell/crumbs'
import { EntitySkeleton, HistoryTable, QueryView, RatioCell } from './EntityParts'
import { LibraryStats } from './LibraryStats'
import { entityPath, formatBytes, itemArt, libraryFamily, share } from './entity'
import styles from './entity.module.css'

const PAGE_SIZE = 25
const SORTS = ['name', 'views', 'viewers', 'watchSeconds']

function usePage(): [number, (p: number) => void] {
  const [page, setPage] = useSearchParam('page', '1')
  return [Math.max(1, Number(page) || 1), (p) => setPage(String(p))]
}

/** Pager of the grid layout (same look as the DataTable pager). */
function GridPager({ page, pageSize, total, onPageChange }: { page: number; pageSize: number; total: number; onPageChange: (p: number) => void }) {
  const { t } = useTranslation()
  const last = Math.max(1, Math.ceil(total / pageSize))
  if (total <= pageSize) return null
  return (
    <div className={styles.pager}>
      <span className="num">{t('app.table.range', { from: (page - 1) * pageSize + 1, to: Math.min(total, page * pageSize), total })}</span>
      <span className={styles.row}>
        <Button variant="ghost" size="sm" iconOnly aria-label={t('app.table.prev')} icon={<ChevronLeft size={16} />} disabled={page <= 1} onClick={() => onPageChange(page - 1)} />
        <Button variant="ghost" size="sm" iconOnly aria-label={t('app.table.next')} icon={<ChevronRight size={16} />} disabled={page >= last} onClick={() => onPageChange(page + 1)} />
      </span>
    </div>
  )
}

function ItemsTab({ data, loading }: { data: LibraryDetail; loading: boolean }) {
  const { t } = useTranslation()
  const f = useFormat()
  const [sp, setSp] = useSearchParams()
  const [layout, setLayout] = useSearchParam('layout', 'table')
  const [page, setPage] = usePage()
  const urlQ = sp.get('q') ?? ''
  const [term, setTerm] = useState(urlQ)
  useEffect(() => { setTerm(urlQ) }, [urlQ])
  useEffect(() => {
    if (term === urlQ) return
    const h = setTimeout(() => setSp((prev) => {
      const out = new URLSearchParams(prev)
      if (term) out.set('q', term); else out.delete('q')
      out.delete('page')
      return out
    }, { replace: true }), 250)
    return () => clearTimeout(h)
  }, [term, urlQ, setSp])
  const sort = SORTS.includes(sp.get('sort') ?? '') ? (sp.get('sort') as string) : 'views'
  const desc = sp.has('desc') ? sp.get('desc') === '1' : sort !== 'name'
  const family = libraryFamily(data.library.type)
  const items = data.items
  const path = (it: LibraryItem) => entityPath(it.kind, it.id)
  const color = `var(--t-${family})`

  const columns: ColumnDef<LibraryItem>[] = [
    {
      id: 'name', accessorKey: 'name', header: t(`app.library.col.${family}`),
      cell: ({ row }) => (
        <span className={styles.cellMedia}>
          <Art title={row.original.name} kind={family === 'music' ? 'square' : 'poster'} src={itemArt(row.original.id, 96)} seed={row.original.id} />
          <span className={styles.title}>{row.original.name}</span>
        </span>
      ),
    },
    { id: 'views', accessorKey: 'views', header: t('app.entity.col.views'), meta: { numeric: true }, cell: ({ row }) => f.number(row.original.views) },
    { id: 'viewers', accessorKey: 'viewers', header: t('app.entity.col.viewers'), meta: { numeric: true }, cell: ({ row }) => f.number(row.original.viewers) },
    { id: 'watchSeconds', accessorKey: 'watchSeconds', header: t('app.entity.watchTime'), meta: { numeric: true }, cell: ({ row }) => f.duration(row.original.watchSeconds) },
    ...(family === 'series' ? [{
      id: 'retention', header: t('app.library.col.retention'), enableSorting: false,
      cell: ({ row }) => <RatioCell value={row.original.retention} tone="series" label={t('app.library.col.retention')} />,
    } satisfies ColumnDef<LibraryItem>] : []),
    {
      id: 'trend', header: t('app.library.col.trend'), enableSorting: false,
      cell: ({ row }) => <Sparkline values={row.original.trend} width={72} height={22} color={color} />,
    },
  ]
  const pagination = { page, pageSize: PAGE_SIZE, total: items.total, onPageChange: setPage }
  return (
    <div className={styles.stack}>
      <div className={styles.toolbar}>
        <SearchInput value={term} onChange={setTerm} placeholder={t('app.library.search')} />
        <SegmentedControl
          aria-label={t('app.library.layout')} value={layout === 'grid' ? 'grid' : 'table'} onValueChange={setLayout}
          options={[
            { value: 'table', label: <List size={14} aria-hidden="true" />, ariaLabel: t('app.library.layoutTable') },
            { value: 'grid', label: <LayoutGrid size={14} aria-hidden="true" />, ariaLabel: t('app.library.layoutGrid') },
          ]}
        />
      </div>
      <Card>
        {layout === 'grid' ? (
          items.rows.length === 0 ? <EmptyState title={t('app.library.noItems')} /> : (
            <>
              <div className={styles.posters}>
                {items.rows.map((it) => (
                  <AppLink key={it.id} to={path(it)} className={styles.poster}>
                    <Art title={it.name} kind={family === 'music' ? 'square' : 'poster'} src={itemArt(it.id, 240)} seed={it.id} />
                    <span className={styles.tileName}>{it.name}</span>
                    <span className={styles.tileMeta}>{t('app.library.viewsCount', { count: it.views })}</span>
                  </AppLink>
                ))}
              </div>
              <GridPager {...pagination} />
            </>
          )
        ) : (
          <DataTable
            columns={columns} data={items.rows} rowHref={path} loading={loading && items.rows.length === 0}
            pagination={pagination}
            sort={{ id: sort, desc, onChange: (s) => setSp((prev) => { const out = new URLSearchParams(prev); out.set('sort', s.id); out.set('desc', s.desc ? '1' : '0'); out.delete('page'); return out }, { replace: true }) }}
            empty={<EmptyState title={t('app.library.noItems')} />}
          />
        )}
      </Card>
    </div>
  )
}

function ActivityTab({ id }: { id: string }) {
  const [page, setPage] = usePage()
  const q = useHistory({ libraryId: id, page, pageSize: PAGE_SIZE })
  return (
    <Card>
      <HistoryTable
        rows={q.data?.rows ?? []} loading={q.isPending}
        pagination={q.data ? { page, pageSize: PAGE_SIZE, total: q.data.total, onPageChange: setPage } : undefined}
      />
    </Card>
  )
}

function Library({ id, data, loading }: { id: string; data: LibraryDetail; loading: boolean }) {
  const { t } = useTranslation()
  const f = useFormat()
  const [sp, setSp] = useSearchParams()
  const tab = sp.get('tab') ?? 'items'
  const lib = data.library
  const family = libraryFamily(lib.type)
  const unwatchedShare = share(data.unwatched.count, lib.items)
  const setTab = (v: string) => setSp((prev) => {
    const out = new URLSearchParams(prev)
    if (v === 'items') out.delete('tab'); else out.set('tab', v)
    out.delete('page')
    return out
  }, { replace: true })
  return (
    <div className={styles.stack}>
      <PageHeader title={lib.name} subtitle={`${t(`app.libraries.items.${family}`, { count: lib.items })} · ${formatBytes(lib.sizeBytes, f.locale)}`} />
      <Card>
        <StatStrip columns={4}>
          <StatTile label={t('app.entity.watchTime')} value={f.duration(lib.totals.watchSeconds)} />
          <StatTile label={t('app.entity.views')} value={f.number(lib.totals.views)} foot={t('app.entity.viewersFoot', { count: lib.totals.activeUsers })} />
          <StatTile label={t('app.library.neverWatched')} value={f.number(data.unwatched.count)} foot={unwatchedShare != null ? t('app.library.ofCatalog', { pct: f.percent(unwatchedShare) }) : undefined} />
          <StatTile label={t('app.entity.completion')} value={lib.totals.completion == null ? '—' : f.percent(lib.totals.completion)} />
        </StatStrip>
      </Card>
      <Tabs value={tab} onValueChange={setTab} items={[
        { value: 'items', label: t('app.library.tab.items'), count: data.items.total },
        { value: 'stats', label: t('app.library.tab.stats') },
        { value: 'activity', label: t('app.library.tab.activity') },
      ]} />
      {tab === 'stats' ? <LibraryStats id={id} data={data} />
        : tab === 'activity' ? <ActivityTab id={id} />
          : <ItemsTab data={data} loading={loading} />}
    </div>
  )
}

export default function LibraryPage() {
  const { id = '' } = useParams()
  const [sp] = useSearchParams()
  const tab = sp.get('tab') ?? 'items'
  const sort = sp.get('sort') ?? undefined
  const desc = sp.has('desc') ? sp.get('desc') === '1' : undefined
  const page = tab === 'items' ? Math.max(1, Number(sp.get('page')) || 1) : 1
  const q = useLibrary(id, { q: sp.get('q') ?? undefined, page, pageSize: PAGE_SIZE, sort: sort && SORTS.includes(sort) ? sort : undefined, desc })
  useSetCrumbLabel(q.data?.library.name)
  return <QueryView query={q} skeleton={<EntitySkeleton header={false} />}>{(data) => <Library id={id} data={data} loading={q.isFetching} />}</QueryView>
}
