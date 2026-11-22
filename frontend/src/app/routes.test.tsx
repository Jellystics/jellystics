import { render, screen, waitFor } from '@testing-library/react'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { routes } from './routes'

vi.mock('@/app/legacy/LegacyPage', () => ({ LegacyPage: ({ children }: { children: React.ReactNode }) => <>{children}</> }))
vi.mock('@/lib/socket', () => ({ default: { connect: vi.fn(), disconnect: vi.fn() }, useSocketEvent: vi.fn() }))
vi.mock('@/features/dashboard/DashboardPage', () => ({ default: () => <p>legacy-dashboard</p> }))
vi.mock('@/features/activity/ActivityPage', () => ({ default: () => <p>legacy-activity</p> }))
vi.mock('@/features/activity/TimelinePage', () => ({ default: () => <p>legacy-timeline</p> }))
vi.mock('@/features/libraries/LibrariesPage', () => ({ default: () => <p>legacy-libraries</p> }))
vi.mock('@/features/libraries/LibraryDetailPage', () => ({ default: () => <p>legacy-library</p> }))
vi.mock('@/features/libraries/ItemDetailPage', async () => {
  const { useParams } = await import('react-router-dom')
  return { default: function Item() { return <p>legacy-item:{useParams().itemId}</p> } }
})
vi.mock('@/features/libraries/AlbumDetailPage', async () => {
  const { useParams } = await import('react-router-dom')
  return { default: function Album() { return <p>legacy-album:{useParams().albumId}</p> } }
})
vi.mock('@/features/users/UsersPage', () => ({ default: () => <p>legacy-users</p> }))
vi.mock('@/features/users/UserDetailPage', () => ({ default: () => <p>legacy-user</p> }))
vi.mock('@/features/statistics/StatisticsPage', () => ({ default: () => <p>legacy-statistics</p> }))
vi.mock('@/features/settings/SettingsPage', async () => {
  const { useSearchParams } = await import('react-router-dom')
  return { default: function Settings() { return <p>legacy-settings:{useSearchParams()[0].get('view')}</p> } }
})
vi.mock('@/features/about/AboutPage', () => ({ default: () => <p>legacy-about</p> }))

function mockFetch(map: Record<string, unknown>) {
  vi.stubGlobal('fetch', vi.fn(async (url: string) => {
    const hit = map[String(url).split('?')[0]]
    return new Response(hit === undefined ? '' : JSON.stringify(hit), { status: hit === undefined ? 404 : 200 })
  }))
}

function renderApp(path: string, { token = 'tok' as string | null } = {}) {
  if (token) localStorage.setItem('jellystics-token', token)
  else localStorage.removeItem('jellystics-token')
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['server-config'], { configured: true, timezone: 'Europe/Paris' })
  const router = createMemoryRouter(routes, { initialEntries: [path] })
  render(<QueryClientProvider client={qc}><RouterProvider router={router} /></QueryClientProvider>)
  return router
}

beforeEach(() => mockFetch({}))
afterEach(() => { vi.unstubAllGlobals(); localStorage.clear() })

it.each([
  ['/statistics', '/insights'], ['/activity', '/history'], ['/activity/timeline', '/history?view=calendar'],
  ['/about', '/settings/about'], ['/setup', '/login'], ['/settings', '/settings/config'], ['/settings?view=backup', '/settings/backup'],
  ['/libraries/l1/albums/a1', '/albums/a1'],
])('redirects %s -> %s', async (from, to) => {
  const router = renderApp(from)
  await waitFor(() => expect(router.state.location.pathname + router.state.location.search).toBe(to))
})

it.each([
  [{ type: 'episode', id: 'x1', seriesId: 's1' }, '/series/s1/episodes/x1'],
  [{ type: 'movie', id: 'x1' }, '/movies/x1'],
  [{ type: 'series', id: 'x1' }, '/series/x1'],
  [{ type: 'album', id: 'x1' }, '/albums/x1'],
  [{ type: 'track', id: 'x1', albumId: 'a1' }, '/albums/a1?track=x1'],
  [{ type: 'other', id: 'x1' }, '/movies/x1'],
])('item redirect resolves %j', async (body, to) => {
  mockFetch({ '/api/v2/items/x1/route': body })
  const router = renderApp('/items/x1')
  await waitFor(() => expect(router.state.location.pathname + router.state.location.search).toBe(to))
})
it('item redirect resolves the episode case from the brief', async () => {
  mockFetch({ '/api/v2/items/e1/route': { type: 'episode', id: 'e1', seriesId: 's1' } })
  const router = renderApp('/libraries/l1/items/e1')
  await waitFor(() => expect(router.state.location.pathname).toBe('/series/s1/episodes/e1'))
})
it('item redirect shows 404 when unknown', async () => {
  renderApp('/items/zzz')
  expect(await screen.findByText('app.notFound.title')).toBeInTheDocument()
})
it('unknown path renders the 404 page', async () => {
  renderApp('/nope'); expect(await screen.findByText('app.notFound.title')).toBeInTheDocument()
})
it('unauthenticated access goes to login with next', async () => {
  const router = renderApp('/users?range=7d', { token: null })
  await waitFor(() => expect(router.state.location.pathname).toBe('/login'))
  expect(router.state.location.search).toBe('?next=%2Fusers%3Frange%3D7d')
})
it('legacy dashboard is mounted at /', async () => { renderApp('/'); expect(await screen.findByText('legacy-dashboard')).toBeInTheDocument() })
it('history shows the timeline for ?view=calendar', async () => {
  renderApp('/history?view=calendar'); expect(await screen.findByText('legacy-timeline')).toBeInTheDocument()
})
it('legacy item page sees itemId (episode id wins)', async () => {
  renderApp('/series/s1/episodes/e9'); expect(await screen.findByText('legacy-item:e9')).toBeInTheDocument()
})
it('legacy album page sees albumId', async () => {
  renderApp('/albums/a7'); expect(await screen.findByText('legacy-album:a7')).toBeInTheDocument()
})
it('settings section maps to ?view=', async () => {
  renderApp('/settings/backup'); expect(await screen.findByText('legacy-settings:backup')).toBeInTheDocument()
})
it('coming soon pages render', async () => {
  renderApp('/live'); expect(await screen.findByText('app.comingSoon.title')).toBeInTheDocument()
})
