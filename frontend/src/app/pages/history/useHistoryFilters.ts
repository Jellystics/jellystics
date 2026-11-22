import { useCallback, useMemo } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useHistory, type HistoryFilters, type MediaFamily } from '@/api'

/** URL keys of the history filters (spec §6: user, type, client, device, method, views=1, q). */
export type FilterKey = 'user' | 'type' | 'client' | 'device' | 'method' | 'views' | 'q'
export const FILTER_KEYS: FilterKey[] = ['user', 'type', 'client', 'device', 'method', 'views', 'q']
const TYPES: MediaFamily[] = ['movie', 'series', 'music']
export type HistorySort = 'startedAt' | 'watchSeconds' | 'completion'
const SORTS: HistorySort[] = ['startedAt', 'watchSeconds', 'completion']

export interface HistoryUrlState {
  values: Partial<Record<FilterKey, string>>
  /** Filters for the API (no paging). */
  filters: HistoryFilters
  page: number
  sort: HistorySort
  desc: boolean
  set: (key: FilterKey, value?: string) => void
  setPage: (page: number) => void
  setSort: (s: { id: string; desc: boolean }) => void
}

/** History filters, paging and sort bound to the URL; changing a filter resets the page. */
export function useHistoryFilters(fixed: HistoryFilters = {}): HistoryUrlState {
  const [sp, setSp] = useSearchParams()
  const values = useMemo(() => {
    const v: Partial<Record<FilterKey, string>> = {}
    FILTER_KEYS.forEach((k) => { const x = sp.get(k); if (x) v[k] = x })
    return v
  }, [sp])
  const type = TYPES.includes(values.type as MediaFamily) ? (values.type as MediaFamily) : undefined
  const filters: HistoryFilters = {
    userId: values.user, type, client: values.client, device: values.device, method: values.method,
    views: values.views === '1', q: values.q, ...fixed,
  }
  const page = Math.max(1, Number(sp.get('page')) || 1)
  const sortParam = sp.get('sort') as HistorySort | null
  const sort: HistorySort = sortParam && SORTS.includes(sortParam) ? sortParam : 'startedAt'
  const desc = sp.get('desc') !== '0'

  const update = useCallback((fn: (p: URLSearchParams) => void, replace = false) => {
    setSp((prev) => { const out = new URLSearchParams(prev); fn(out); return out }, { replace })
  }, [setSp])
  const set = useCallback((key: FilterKey, value?: string) => update((p) => {
    if (value) p.set(key, value); else p.delete(key)
    p.delete('page')
  }, key === 'q'), [update])
  const setPage = useCallback((n: number) => update((p) => { if (n > 1) p.set('page', String(n)); else p.delete('page') }), [update])
  const setSort = useCallback((s: { id: string; desc: boolean }) => update((p) => {
    if (s.id === 'startedAt') p.delete('sort'); else p.set('sort', s.id)
    if (s.desc) p.delete('desc'); else p.set('desc', '0')
    p.delete('page')
  }), [update])

  return { values, filters, page, sort, desc, set, setPage, setSort }
}

export const HISTORY_PAGE_SIZE = 25

/** The current history page for the URL state (one cache entry shared by the table and the page header). */
export function useHistoryRows(state: Pick<HistoryUrlState, 'filters' | 'page' | 'sort' | 'desc'>) {
  // The API sorts descending unless desc=0 is sent explicitly; cleanParams drops `false`, so send 0.
  const desc = (state.desc ? true : 0) as unknown as boolean
  return useHistory({ ...state.filters, page: state.page, pageSize: HISTORY_PAGE_SIZE, sort: state.sort, desc })
}
