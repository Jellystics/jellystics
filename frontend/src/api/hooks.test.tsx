import { renderHook, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useHistory, useMovie, useOverview, useSearch } from './hooks'

function wrap(initial: string) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['server-config'], { configured: true, jellyfinUrl: '', timezone: 'Europe/Paris' })
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={qc}><MemoryRouter initialEntries={[initial]}>{children}</MemoryRouter></QueryClientProvider>
  )
}

function mockFetch(body: unknown) {
  return vi.spyOn(globalThis, 'fetch').mockImplementation(async () => new Response(JSON.stringify(body), { status: 200 }))
}
const calledUrl = (spy: ReturnType<typeof mockFetch>, i = 0) => new URL(String(spy.mock.calls[i][0]), 'http://x')

beforeEach(() => { vi.restoreAllMocks(); localStorage.clear() })

it('useOverview sends the URL period, compare and filters', async () => {
  const spy = mockFetch({ totals: { views: 3 } })
  const { result } = renderHook(() => useOverview({ type: 'movie' }), { wrapper: wrap('/?from=2026-11-01&to=2026-11-08&compare=1') })
  await waitFor(() => expect(result.current.isSuccess).toBe(true))
  const url = calledUrl(spy)
  expect(url.pathname).toBe('/api/v2/overview')
  expect(Object.fromEntries(url.searchParams)).toEqual({ compare: '1', from: '2026-11-01', to: '2026-11-08', type: 'movie' })
  expect(result.current.data?.totals.views).toBe(3)
})

it('useHistory maps history filters (views=1, paging) and omits compare when off', async () => {
  const spy = mockFetch({ rows: [], total: 0, page: 2, pageSize: 50 })
  const { result } = renderHook(() => useHistory({ views: true, page: 2, pageSize: 50, client: '' }), { wrapper: wrap('/history?range=7d') })
  await waitFor(() => expect(result.current.isSuccess).toBe(true))
  const params = calledUrl(spy).searchParams
  expect(params.get('views')).toBe('1')
  expect(params.get('page')).toBe('2')
  expect(params.has('compare')).toBe(false)
  expect(params.has('client')).toBe(false)
  expect(params.get('from')).toMatch(/^\d{4}-\d{2}-\d{2}$/)
})

it('entity hooks encode the id and stay idle without one', async () => {
  const spy = mockFetch({ item: { id: 'a b' } })
  const { result } = renderHook(() => useMovie('a b'), { wrapper: wrap('/movies/a%20b') })
  await waitFor(() => expect(result.current.isSuccess).toBe(true))
  expect(calledUrl(spy).pathname).toBe('/api/v2/movies/a%20b')
  const idle = renderHook(() => useMovie(undefined), { wrapper: wrap('/') })
  expect(idle.result.current.fetchStatus).toBe('idle')
  expect(spy).toHaveBeenCalledTimes(1)
})

it('useSearch does not fetch for a blank query', async () => {
  const spy = mockFetch({ items: [], users: [] })
  const blank = renderHook(() => useSearch('  '), { wrapper: wrap('/') })
  expect(blank.result.current.fetchStatus).toBe('idle')
  const { result } = renderHook(() => useSearch(' inc '), { wrapper: wrap('/') })
  await waitFor(() => expect(result.current.isSuccess).toBe(true))
  expect(calledUrl(spy).search).toBe('?q=inc')
})
