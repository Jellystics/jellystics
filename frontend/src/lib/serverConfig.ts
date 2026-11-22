import { useQuery } from '@tanstack/react-query'
import { apiFetch } from './api'
export interface ServerConfig { configured: boolean; jellyfinUrl?: string; timezone: string }
export function useServerConfig() {
  return useQuery({
    queryKey: ['server-config'],
    staleTime: Infinity,
    queryFn: async (): Promise<ServerConfig> => {
      const r = await apiFetch<{ state: number; jellyfinUrl?: string; timezone?: string }>('/auth/isConfigured')
      return { configured: r.state === 1, jellyfinUrl: r.jellyfinUrl, timezone: r.timezone || 'UTC' }
    },
  })
}
