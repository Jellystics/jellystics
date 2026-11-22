import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AppShell } from './AppShell'
import { AppLink } from '@/ui'

function renderAt(path: string) {
  const qc = new QueryClient(); qc.setQueryData(['server-config'], { configured: true, timezone: 'Europe/Paris' })
  const router = createMemoryRouter([{ element: <AppShell />, children: [
    { path: '/', handle: { nav: 'overview' }, element: <AppLink to="/users">Utilisateurs</AppLink> },
    { path: '/users', handle: { nav: 'users', crumb: () => ({ label: 'app.nav.users' }) }, element: <p>users</p> },
    { path: '/movies/:id', element: <p>movie</p> },
  ] }], { initialEntries: [path] })
  render(<QueryClientProvider client={qc}><RouterProvider router={router} /></QueryClientProvider>)
  return router
}

it('marks the active nav entry with aria-current', () => {
  renderAt('/users')
  expect(screen.getByRole('link', { name: 'app.nav.users' })).toHaveAttribute('aria-current', 'page')
})
it('AppLink keeps the period', () => {
  renderAt('/?range=7d')
  expect(screen.getByRole('link', { name: 'Utilisateurs' })).toHaveAttribute('href', '/users?range=7d')
})
it('sidebar nav links keep the period too', () => {
  renderAt('/?range=90d')
  expect(screen.getByRole('link', { name: 'app.nav.history' })).toHaveAttribute('href', '/history?range=90d')
})
it('opens the command palette with Ctrl+K', async () => {
  renderAt('/')
  await userEvent.keyboard('{Control>}k{/Control}')
  expect(screen.getByRole('dialog', { name: 'app.command.title' })).toBeInTheDocument()
})
it('renders breadcrumbs from route handles', () => {
  renderAt('/users')
  expect(screen.getByText('app.nav.users', { selector: '[aria-current="page"]:not(a)' })).toBeInTheDocument()
})
it('resets the palette query when reopened', async () => {
  renderAt('/')
  await userEvent.keyboard('{Control>}k{/Control}')
  await userEvent.type(screen.getByRole('searchbox'), 'xyz')
  await userEvent.keyboard('{Control>}k{/Control}')
  await userEvent.keyboard('{Control>}k{/Control}')
  expect(screen.getByRole('searchbox')).toHaveValue('')
})
it('hides the period picker on routes flagged period:false', () => {
  const qc = new QueryClient(); qc.setQueryData(['server-config'], { configured: true, timezone: 'Europe/Paris' })
  const router = createMemoryRouter([{ element: <AppShell />, children: [
    { path: '/', handle: { nav: 'overview', period: false }, element: <p>own</p> },
    { path: '/b', handle: { nav: 'users' }, element: <p>b</p> },
  ] }], { initialEntries: ['/'] })
  const { unmount } = render(<QueryClientProvider client={qc}><RouterProvider router={router} /></QueryClientProvider>)
  expect(screen.queryByRole('button', { name: /app\.period\./ })).toBeNull()
  unmount()
  const r2 = createMemoryRouter([{ element: <AppShell />, children: [{ path: '/b', handle: { nav: 'users' }, element: <p>b</p> }] }], { initialEntries: ['/b'] })
  render(<QueryClientProvider client={qc}><RouterProvider router={r2} /></QueryClientProvider>)
  expect(screen.getByRole('button', { name: /app\.period\./ })).toBeInTheDocument()
})

it('palette searches the catalog (debounced) and links results with the period', async () => {
  const spy = vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
    const url = String(input)
    if (url.startsWith('/api/v2/search')) {
      return new Response(JSON.stringify({
        items: [
          { id: 'm1', kind: 'movie', name: 'Inception', year: 2010 },
          { id: 't1', kind: 'track', name: 'Incipit', parentId: 'al1', parentName: 'Blue Album' },
        ],
        users: [{ id: 'u1', name: 'Incognito' }],
      }), { status: 200 })
    }
    return new Response('{}', { status: 200 })
  })
  const router = renderAt('/?range=7d')
  await userEvent.keyboard('{Control>}k{/Control}')
  await userEvent.type(screen.getByRole('searchbox'), 'inc')
  const movie = await screen.findByRole('link', { name: /Inception/ })
  expect(movie).toHaveAttribute('href', '/movies/m1?range=7d')
  expect(screen.getByRole('link', { name: /Incipit/ })).toHaveAttribute('href', '/albums/al1?track=t1&range=7d')
  expect(screen.getByRole('link', { name: /Incognito/ })).toHaveAttribute('href', '/users/u1?range=7d')
  const searchCalls = spy.mock.calls.filter(([u]) => String(u).startsWith('/api/v2/search'))
  expect(searchCalls).toHaveLength(1)
  expect(String(searchCalls[0][0])).toBe('/api/v2/search?q=inc')
  await userEvent.keyboard('{Enter}')
  expect(router.state.location.pathname).toBe('/movies/m1')
  expect(router.state.location.search).toBe('?range=7d')
  spy.mockRestore()
})
