import { Navigate, useLocation, useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { ApiError, apiFetch } from '@/lib/api'
import { ErrorState, Skeleton, withPeriod } from '@/ui'
import { NotFoundPage } from './NotFoundPage'

export type ItemRoute = { type: 'movie' | 'series' | 'episode' | 'album' | 'track' | 'other'; id: string; seriesId?: string; albumId?: string }

export function itemPath(r: ItemRoute): string {
  switch (r.type) {
    case 'series': return `/series/${r.id}`
    case 'episode': return r.seriesId ? `/series/${r.seriesId}/episodes/${r.id}` : `/movies/${r.id}`
    case 'album': return `/albums/${r.id}`
    case 'track': return r.albumId ? `/albums/${r.albumId}?track=${encodeURIComponent(r.id)}` : `/movies/${r.id}`
    default: return `/movies/${r.id}`
  }
}

/** Legacy /items/:id URLs: resolve the item kind server-side, then redirect to its page. */
export function ItemRedirect() {
  const { id = '' } = useParams()
  const { t } = useTranslation()
  const { search } = useLocation()
  const q = useQuery({
    queryKey: ['item-route', id],
    queryFn: () => apiFetch<ItemRoute>(`/api/v2/items/${encodeURIComponent(id)}/route`),
    retry: (n, e) => !(e instanceof ApiError && e.status === 404) && n < 1,
  })
  if (q.isPending) return <div style={{ padding: 'var(--s-6)' }}><Skeleton height={24} width={240} /></div>
  if (q.isError) {
    if (q.error instanceof ApiError && q.error.status === 404) return <NotFoundPage />
    return <ErrorState title={t('app.error.generic')} onRetry={() => void q.refetch()} />
  }
  const target = itemPath(q.data)
  if (target.includes('?')) return <Navigate to={withPeriod(target, new URLSearchParams(search))} replace />
  return <Navigate to={target + search} replace />
}
