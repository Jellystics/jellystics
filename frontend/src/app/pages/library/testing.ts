// Test helpers for the phase 4 entity pages (imported by *.test.tsx only).
import { createElement, Fragment, type ReactElement } from 'react'
import { render } from '@testing-library/react'
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

type Reply = unknown | ((url: URL) => unknown)

/** Stubs fetch: path → JSON body (404 when unknown, 500 for `{ __status: 500 }`). Returns the spy. */
export function mockApi(map: Record<string, Reply>) {
  const spy = vi.fn(async (input: RequestInfo | URL) => {
    const url = new URL(String(input), 'http://x')
    const hit = map[url.pathname]
    const body = typeof hit === 'function' ? (hit as (u: URL) => unknown)(url) : hit
    if (body === undefined) return new Response('{"error":"not found"}', { status: 404 })
    const status = (body as { __status?: number })?.__status
    if (status) return new Response('{"error":"x"}', { status })
    return new Response(JSON.stringify(body), { status: 200 })
  })
  vi.stubGlobal('fetch', spy)
  return spy
}

/** URLs fetched so far for a path. */
export const calls = (spy: ReturnType<typeof mockApi>, path: string) =>
  spy.mock.calls.map((c) => new URL(String(c[0]), 'http://x')).filter((u) => u.pathname === path)

/** Current location (pathname + search) of the last rendered page. */
export const loc = { current: '' }
function Loc() { const l = useLocation(); loc.current = l.pathname + l.search; return null }

/** Renders `element` at `path` (route `pattern`) with a fresh query client. */
export function renderPage(pattern: string, path: string, element: ReactElement) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['server-config'], { configured: true, timezone: 'Europe/Paris' })
  const routed = createElement(Fragment, null, element, createElement(Loc))
  return render(
    createElement(QueryClientProvider, { client: qc },
      createElement(MemoryRouter, { initialEntries: [path] },
        createElement(Routes, null,
          createElement(Route, { path: pattern, element: routed }),
          createElement(Route, { path: '*', element: createElement(Loc) })))),
  )
}

/** A realistic history row. */
export const historyRow = (over: Record<string, unknown> = {}) => ({
  id: 'h1', startedAt: '2026-11-20T21:10:00+01:00', endedAt: '2026-11-20T22:00:00+01:00',
  user: { id: 'u1', name: 'Céline' }, item: { id: 'e5', kind: 'episode', name: 'La Fracture', parentId: 's1', parentName: 'Station Nord', season: 2, episode: 5 },
  mediaType: 'Episode', watchSeconds: 2810, completion: 0.98, client: 'Jellyfin Web', device: 'Chrome', method: 'DirectPlay', isView: true, pausedSeconds: 0,
  ...over,
})
