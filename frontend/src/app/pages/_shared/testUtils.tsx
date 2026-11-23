import type { ReactElement } from 'react'
import { render } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

type Responder = unknown | ((url: URL) => unknown)

/**
 * Mocks fetch by pathname (exact match). A value of `Error` answers 500.
 * Unknown paths answer 404 so missing mocks are visible.
 */
export function mockApi(routes: Record<string, Responder>) {
  return vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
    const url = new URL(String(input), 'http://x')
    if (!(url.pathname in routes)) return new Response(JSON.stringify({ error: 'not mocked' }), { status: 404 })
    const r = routes[url.pathname]
    if (r === Error) return new Response(JSON.stringify({ error: 'boom' }), { status: 500 })
    const body = typeof r === 'function' ? (r as (u: URL) => unknown)(url) : r
    return new Response(JSON.stringify(body), { status: 200 })
  })
}

/** Renders `element` mounted at `path` with the router at `url`. */
export function renderPage(element: ReactElement, { path, url }: { path: string; url: string }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['server-config'], { configured: true, jellyfinUrl: '', timezone: 'Europe/Paris' })
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[url]}>
        <Routes><Route path={path} element={element} /></Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}
