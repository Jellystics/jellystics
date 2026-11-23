import { renderHook, act } from '@testing-library/react'
import { MemoryRouter, useLocation } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { usePeriod, useSearchParam } from './useUrlState'

function wrap(initial: string) {
  const qc = new QueryClient()
  qc.setQueryData(['server-config'], { configured: true, jellyfinUrl: '', timezone: 'Europe/Paris' })
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={qc}><MemoryRouter initialEntries={[initial]}>{children}</MemoryRouter></QueryClientProvider>
  )
}

it('reads and writes the period, keeping other params', () => {
  const { result } = renderHook(() => ({ p: usePeriod(), loc: useLocation() }), { wrapper: wrap('/history?user=lea&range=7d') })
  expect(result.current.p[0].preset).toBe('7d')
  act(() => result.current.p[1]({ preset: '90d' }))
  expect(result.current.loc.search).toBe('?user=lea&range=90d')
  act(() => result.current.p[1]({ compare: true }))
  expect(new URLSearchParams(result.current.loc.search).get('compare')).toBe('1')
})

it('single param read/write and removal', () => {
  const { result } = renderHook(() => ({ s: useSearchParam('tab', 'overview'), loc: useLocation() }), { wrapper: wrap('/users/lea') })
  expect(result.current.s[0]).toBe('overview')
  act(() => result.current.s[1]('history'))
  expect(result.current.loc.search).toBe('?tab=history')
  act(() => result.current.s[1](undefined))
  expect(result.current.loc.search).toBe('')
})

it('pushes a history entry for period changes and supports opt-in push for params', () => {
  const { result } = renderHook(() => ({ s: useSearchParam('page', '1', { push: true }), loc: useLocation() }), { wrapper: wrap('/x') })
  act(() => result.current.s[1]('2'))
  expect(result.current.loc.search).toBe('?page=2')
})

it('custom ranges round-trip and can return to a preset', () => {
  const { result } = renderHook(() => ({ p: usePeriod(), loc: useLocation() }), { wrapper: wrap('/x') })
  act(() => result.current.p[1]({ preset: 'custom', from: '2026-10-01', to: '2026-11-01' }))
  expect(result.current.p[0]).toMatchObject({ preset: 'custom', from: '2026-10-01', to: '2026-11-01' })
  expect(result.current.loc.search).toBe('?from=2026-10-01&to=2026-11-01')
  act(() => result.current.p[1]({ preset: '7d' }))
  expect(result.current.p[0].preset).toBe('7d')
})

it('rolls over past midnight', () => {
  vi.useFakeTimers()
  vi.setSystemTime(new Date('2026-11-22T22:59:30Z')) // 23:59:30 in Paris
  try {
    const { result } = renderHook(() => usePeriod(), { wrapper: wrap('/x?range=today') })
    expect(result.current[0].from).toBe('2026-11-22')
    act(() => { vi.advanceTimersByTime(120_000) })
    expect(result.current[0].from).toBe('2026-11-23')
  } finally { vi.useRealTimers() }
})
