// Test helper for the phase 3 page tests (not used by the app).
import { render } from '@testing-library/react'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactElement } from 'react'

type Body = unknown | ((url: URL) => unknown)
export type FetchMap = Record<string, Body>

/** Stubs fetch: exact pathname match in `map`; a `{ status }` object answers with that status; anything else 404. */
export function stubFetch(map: FetchMap) {
  const fn = vi.fn(async (input: RequestInfo | URL) => {
    const url = new URL(String(input), 'http://x')
    const hit = map[url.pathname]
    if (hit === undefined) return new Response('', { status: 404 })
    const body = typeof hit === 'function' ? (hit as (u: URL) => unknown)(url) : hit
    if (body && typeof body === 'object' && 'status' in body && typeof (body as { status: unknown }).status === 'number' && Object.keys(body).length === 1) {
      return new Response('{}', { status: (body as { status: number }).status })
    }
    return new Response(JSON.stringify(body), { status: 200 })
  })
  vi.stubGlobal('fetch', fn)
  return {
    fn,
    /** Every request URL made to `pathname`. */
    calls: (pathname: string) => fn.mock.calls.map((c) => new URL(String(c[0]), 'http://x')).filter((u) => u.pathname === pathname),
  }
}

/** Renders `element` at `path` (route pattern `route`) inside a router + fresh QueryClient. */
export function renderPage(element: ReactElement, { path = '/', route = '/' }: { path?: string; route?: string } = {}) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['server-config'], { configured: true, timezone: 'Europe/Paris' })
  const router = createMemoryRouter([{ path: route, element }, { path: '*', element: <p>elsewhere</p> }], { initialEntries: [path] })
  const view = render(<QueryClientProvider client={qc}><RouterProvider router={router} /></QueryClientProvider>)
  return { ...view, router, qc, search: () => new URLSearchParams(router.state.location.search) }
}
