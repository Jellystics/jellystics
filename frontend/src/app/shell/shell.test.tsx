import { render, screen, act } from '@testing-library/react'
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
  await userEvent.type(screen.getByRole('combobox'), 'xyz')
  await userEvent.keyboard('{Control>}k{/Control}')
  await userEvent.keyboard('{Control>}k{/Control}')
  expect(screen.getByRole('combobox')).toHaveValue('')
})
it('palette input is a combobox that tracks the active option', async () => {
  renderAt('/')
  await userEvent.keyboard('{Control>}k{/Control}')
  const input = screen.getByRole('combobox')
  const list = screen.getByRole('listbox')
  expect(input).toHaveAttribute('aria-controls', list.id)
  const first = screen.getAllByRole('option')[0]
  expect(input).toHaveAttribute('aria-activedescendant', first.id)
  await userEvent.keyboard('{ArrowDown}')
  expect(input).toHaveAttribute('aria-activedescendant', screen.getAllByRole('option')[1].id)
})
it('offers a skip link that focuses the main landmark', async () => {
  renderAt('/')
  await userEvent.click(screen.getByRole('link', { name: 'app.nav.skipToContent' }))
  expect(screen.getByRole('main')).toHaveFocus()
})
it('closes the mobile drawer when the viewport reaches the desktop layout', async () => {
  let listener: ((e: { matches: boolean }) => void) | undefined
  const orig = window.matchMedia
  window.matchMedia = ((q: string) => ({
    matches: false, media: q, addEventListener: (_: string, l: typeof listener) => { listener = l }, removeEventListener: () => {}, addListener() {}, removeListener() {}, dispatchEvent: () => false,
  })) as unknown as typeof window.matchMedia
  try {
    renderAt('/')
    await userEvent.click(screen.getByRole('button', { name: 'app.nav.openMenu' }))
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    act(() => listener?.({ matches: true }))
    expect(screen.queryByRole('dialog')).toBeNull()
  } finally { window.matchMedia = orig }
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
  await userEvent.type(screen.getByRole('combobox'), 'inc')
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
