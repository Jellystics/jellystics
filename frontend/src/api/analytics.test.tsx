import { renderHook, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useInsightsCompletion, usePlaybackQuality, useRecap, useUserLanguages } from './analytics'

function wrap(initial: string) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['server-config'], { configured: true, jellyfinUrl: '', timezone: 'Europe/Paris' })
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={qc}><MemoryRouter initialEntries={[initial]}>{children}</MemoryRouter></QueryClientProvider>
  )
}
const ok = (body: unknown) => vi.spyOn(globalThis, 'fetch').mockImplementation(async () => new Response(JSON.stringify(body), { status: 200 }))
const url = (spy: ReturnType<typeof ok>, i = 0) => new URL(String(spy.mock.calls[i][0]), 'http://x')

beforeEach(() => { vi.restoreAllMocks() })

it('section hooks send the URL period and filters', async () => {
  const spy = ok({ units: 3 })
  const { result } = renderHook(() => useInsightsCompletion({ type: 'movie' }), { wrapper: wrap('/insights?from=2026-11-01&to=2026-11-08&compare=1') })
  await waitFor(() => expect(result.current.isSuccess).toBe(true))
  expect(url(spy).pathname).toBe('/api/v2/insights/completion')
  expect(Object.fromEntries(url(spy).searchParams)).toEqual({ compare: '1', from: '2026-11-01', to: '2026-11-08', type: 'movie' })
  const q = renderHook(() => usePlaybackQuality(), { wrapper: wrap('/playback?range=7d') })
  await waitFor(() => expect(q.result.current.isSuccess).toBe(true))
  expect(url(spy, 1).pathname).toBe('/api/v2/playback/quality')
})

it('useRecap ignores the URL period and stays idle without params', async () => {
  const spy = ok({ period: '2026' })
  const { result } = renderHook(() => useRecap('a b', '2026-11'), { wrapper: wrap('/?range=7d') })
  await waitFor(() => expect(result.current.isSuccess).toBe(true))
  expect(url(spy).pathname).toBe('/api/v2/recap/a%20b/2026-11')
  expect(url(spy).search).toBe('')
  const idle = renderHook(() => useRecap('u1', undefined), { wrapper: wrap('/') })
  expect(idle.result.current.fetchStatus).toBe('idle')
})

it('useUserLanguages targets the user and carries the period', async () => {
  const spy = ok({ captured: 0 })
  const { result } = renderHook(() => useUserLanguages('u1'), { wrapper: wrap('/users/u1?from=2026-10-01&to=2026-11-01') })
  await waitFor(() => expect(result.current.isSuccess).toBe(true))
  expect(url(spy).pathname).toBe('/api/v2/users/u1/languages')
  expect(url(spy).searchParams.get('from')).toBe('2026-10-01')
})
