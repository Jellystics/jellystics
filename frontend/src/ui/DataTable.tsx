import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ArrowDown, ArrowUp, ChevronLeft, ChevronRight } from 'lucide-react'
import { flexRender, getCoreRowModel, useReactTable, type ColumnDef, type SortingState } from '@tanstack/react-table'
import { Button } from './Button'
import { EmptyState } from './EmptyState'
import { Skeleton } from './Skeleton'
import styles from './DataTable.module.css'

declare module '@tanstack/react-table' {
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  interface ColumnMeta<TData, TValue> { numeric?: boolean; hideFromColumnsMenu?: boolean }
}

export type DataTableProps<T> = {
  columns: ColumnDef<T, any>[] // eslint-disable-line @typescript-eslint/no-explicit-any
  data: T[]
  rowHref?: (row: T) => string
  pagination?: { page: number; pageSize: number; total: number; onPageChange(p: number): void }
  sort?: { id: string; desc: boolean; onChange(s: { id: string; desc: boolean }): void }
  dense?: boolean
  empty?: ReactNode
  loading?: boolean
}

export function DataTable<T>({ columns, data, rowHref, pagination, sort, dense, empty, loading }: DataTableProps<T>) {
  const { t } = useTranslation()
  const sorting: SortingState = sort ? [{ id: sort.id, desc: sort.desc }] : []
  const table = useReactTable({
    columns, data, getCoreRowModel: getCoreRowModel(),
    manualSorting: true, enableSortingRemoval: false, state: { sorting },
    onSortingChange: (u) => {
      if (!sort) return
      const next = (typeof u === 'function' ? u(sorting) : u)[0]
      if (next) sort.onChange({ id: next.id, desc: next.desc })
    },
  })
  const colCount = table.getAllLeafColumns().length
  const rows = table.getRowModel().rows
  const from = pagination ? (pagination.page - 1) * pagination.pageSize + 1 : 0
  const to = pagination ? Math.min(pagination.total, pagination.page * pagination.pageSize) : 0
  const lastPage = pagination ? Math.max(1, Math.ceil(pagination.total / pagination.pageSize)) : 1

  return (
    <div>
      <div className={styles.wrap}>
        <table className={[styles.table, dense && styles.dense].filter(Boolean).join(' ')}>
          <thead>
            {table.getHeaderGroups().map((hg) => (
              <tr key={hg.id}>
                {hg.headers.map((h) => {
                  const numeric = h.column.columnDef.meta?.numeric
                  const sortable = !!sort && h.column.getCanSort()
                  const dir = h.column.getIsSorted()
                  return (
                    <th key={h.id} className={numeric ? styles.num : undefined} aria-sort={dir ? (dir === 'desc' ? 'descending' : 'ascending') : undefined}>
                      {h.isPlaceholder ? null : sortable ? (
                        <button type="button" className={styles.sortBtn} onClick={h.column.getToggleSortingHandler()}>
                          {flexRender(h.column.columnDef.header, h.getContext())}
                          {dir === 'desc' && <ArrowDown size={12} className={styles.arrow} aria-hidden="true" />}
                          {dir === 'asc' && <ArrowUp size={12} className={styles.arrow} aria-hidden="true" />}
                        </button>
                      ) : flexRender(h.column.columnDef.header, h.getContext())}
                    </th>
                  )
                })}
              </tr>
            ))}
          </thead>
          <tbody>
            {loading
              ? Array.from({ length: 5 }, (_, i) => (
                <tr key={i}>{Array.from({ length: colCount }, (_, j) => <td key={j}><Skeleton width="70%" /></td>)}</tr>
              ))
              : rows.map((row) => {
                const href = rowHref?.(row.original)
                return (
                  <tr key={row.id} className={href ? styles.clickable : undefined}>
                    {row.getVisibleCells().map((cell, i) => {
                      const content = flexRender(cell.column.columnDef.cell, cell.getContext())
                      return (
                        <td key={cell.id} className={cell.column.columnDef.meta?.numeric ? styles.num : undefined}>
                          {href && i === 0 ? <Link to={href} className={styles.rowLink}>{content}</Link> : content}
                        </td>
                      )
                    })}
                  </tr>
                )
              })}
          </tbody>
        </table>
      </div>
      {!loading && rows.length === 0 && (empty ?? <EmptyState title={t('app.table.empty')} />)}
      {pagination && pagination.total > 0 && (
        <div className={styles.pager}>
          <span>{t('app.table.range', { from, to, total: pagination.total })}</span>
          <div className={styles.pagerBtns}>
            <Button variant="ghost" size="sm" iconOnly aria-label={t('app.table.prev')} icon={<ChevronLeft size={16} />} disabled={pagination.page <= 1} onClick={() => pagination.onPageChange(pagination.page - 1)} />
            <Button variant="ghost" size="sm" iconOnly aria-label={t('app.table.next')} icon={<ChevronRight size={16} />} disabled={pagination.page >= lastPage} onClick={() => pagination.onPageChange(pagination.page + 1)} />
          </div>
        </div>
      )}
    </div>
  )
}
