import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { TooltipProvider } from '@/ui/Tooltip'
import { MePage, SignedOut } from './MePage'
import type { Session } from './session'

/** Root of me.html: no router, no dashboard session; the Jellyfin session comes from jellyfin-web's storage. */
export function MeApp({ session, basePath, client }: { session: Session | null; basePath: string; client?: QueryClient }) {
  const { t } = useTranslation()
  const title = t('app.me.title')
  useEffect(() => { document.title = `${title} · Jellystics` }, [title])
  const [qc] = useState(() => client ?? new QueryClient({ defaultOptions: { queries: { refetchOnWindowFocus: false } } }))
  return (
    <QueryClientProvider client={qc}>
      <TooltipProvider>
        {session ? <MePage session={session} basePath={basePath} /> : <SignedOut basePath={basePath} />}
      </TooltipProvider>
    </QueryClientProvider>
  )
}
