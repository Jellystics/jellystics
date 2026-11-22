import type { ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router-dom'
import { getToken } from '@/lib/auth'
import { useServerConfig } from '@/lib/serverConfig'
import { Skeleton } from '@/ui'

export function RequireAuth({ children }: { children: ReactNode }) {
  const { data, isPending } = useServerConfig()
  const { pathname, search } = useLocation()
  if (isPending) return <div style={{ padding: 'var(--s-6)' }}><Skeleton height={24} width={240} /></div>
  if (data && !data.configured) return <Navigate to="/login" replace />
  if (!getToken()) return <Navigate to={`/login?next=${encodeURIComponent(pathname + search)}`} replace />
  return <>{children}</>
}
