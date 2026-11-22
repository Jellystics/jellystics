import { renderHook, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useConcurrency, useUserRetention, useUserShare } from './usersExtra'
import { sessionBitrate } from './live'

function wrap(initial: string) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['server-config'], { configured: true, timezone: 'Europe/Paris' })
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={qc}><MemoryRouter initialEntries={[initial]}>{children}</MemoryRouter></QueryClientProvider>
  )
}
const spy = () => vi.spyOn(globalThis, 'fetch').mockImplementation(async () => new Response('{}', { status: 200 }))
const url = (s: ReturnType<typeof spy>, i = 0) => new URL(String(s.mock.calls[i][0]), 'http://x')

beforeEach(() => { vi.restoreAllMocks() })

it('retention and share send the URL period and filters', async () => {
  const s = spy()
  const r = renderHook(() => { useUserRetention({ type: 'series' }); return useUserShare() }, { wrapper: wrap('/users?from=2026-10-01&to=2026-11-01') })
  await waitFor(() => expect(r.result.current.isSuccess).toBe(true))
  const calls = s.mock.calls.map((_, i) => url(s, i))
  const ret = calls.find((u) => u.pathname === '/api/v2/users/retention')!
  expect(Object.fromEntries(ret.searchParams)).toEqual({ from: '2026-10-01', to: '2026-11-01', type: 'series' })
  expect(calls.find((u) => u.pathname === '/api/v2/users/share')!.searchParams.get('from')).toBe('2026-10-01')
})

it('concurrency ignores the period and sends the day', async () => {
  const s = spy()
  const r = renderHook(() => useConcurrency('2026-11-20'), { wrapper: wrap('/live?range=7d') })
  await waitFor(() => expect(r.result.current.isSuccess).toBe(true))
  expect(url(s).pathname).toBe('/api/v2/concurrency')
  expect(url(s).search).toBe('?date=2026-11-20')
})

it('sessionBitrate prefers the transcode output', () => {
  expect(sessionBitrate({ Id: 'a', UserId: 'u', UserName: 'u', Client: 'c', DeviceName: 'd', TranscodingInfo: { Bitrate: 5 }, NowPlayingItem: { Id: 'i', Name: 'n', Type: 'Movie', MediaSources: [{ Bitrate: 9 }] } })).toBe(5)
  expect(sessionBitrate({ Id: 'a', UserId: 'u', UserName: 'u', Client: 'c', DeviceName: 'd', NowPlayingItem: { Id: 'i', Name: 'n', Type: 'Movie', MediaSources: [{ Bitrate: 9 }] } })).toBe(9)
})
