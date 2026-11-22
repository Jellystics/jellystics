import { render, screen, waitFor } from '@testing-library/react'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { routes } from './routes'
import { episodeFixture, seriesFixture } from './pages/library/fixtures'

vi.mock('@/app/legacy/LegacyPage', () => ({ LegacyPage: ({ children }: { children: React.ReactNode }) => <>{children}</> }))
vi.mock('@/lib/socket', () => ({ default: { connect: vi.fn(), disconnect: vi.fn() }, useSocketEvent: vi.fn() }))
vi.mock('./pages/overview/OverviewPage', () => ({ default: () => <p>page-overview</p> }))
vi.mock('./pages/live/LivePage', () => ({ default: () => <p>page-live</p> }))
vi.mock('./pages/history/HistoryPage', async () => {
  const { useSearchParams } = await import('react-router-dom')
  return { default: function History() { return <p>page-history:{useSearchParams()[0].get('view') ?? 'table'}</p> } }
})
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
vi.mock('./pages/users/UsersPage', () => ({ default: () => <p>page-users</p> }))
vi.mock('./pages/user/UserPage', () => ({ default: () => <p>page-user</p> }))
vi.mock('@/features/statistics/StatisticsPage', () => ({ default: () => <p>legacy-statistics</p> }))
vi.mock('./pages/settings/SettingsPage', async (orig) => ({ ...(await orig<object>()), SettingsPage: ({ section }: { section: string }) => <p>settings:{section}</p> }))

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

it('timeline redirect keeps query params and adds view=calendar', async () => {
  const router = renderApp('/activity/timeline?range=7d&userId=u1')
  await waitFor(() => expect(router.state.location.pathname).toBe('/history'))
  const sp = new URLSearchParams(router.state.location.search)
  expect(Object.fromEntries(sp)).toEqual({ range: '7d', userId: 'u1', view: 'calendar' })
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
it.each([['/', 'page-overview'], ['/live', 'page-live'], ['/users', 'page-users'], ['/users/lea', 'page-user']])('%s mounts the new page', async (p, text) => {
  renderApp(p); expect(await screen.findByText(text)).toBeInTheDocument()
})
it('history mounts the new page with ?view=calendar', async () => {
  renderApp('/history?view=calendar'); expect(await screen.findByText('page-history:calendar')).toBeInTheDocument()
})
it('episode page fetches the episode id (not the series id)', async () => {
  renderApp('/series/s1/episodes/e9')
  expect(await screen.findByText('app.entity.notFound')).toBeInTheDocument()
  expect(vi.mocked(fetch).mock.calls.some(([u]) => String(u).startsWith('/api/v2/episodes/e9'))).toBe(true)
})
it('album page fetches its album', async () => {
  renderApp('/albums/a7')
  expect(await screen.findByText('app.entity.notFound')).toBeInTheDocument()
  expect(vi.mocked(fetch).mock.calls.some(([u]) => String(u).startsWith('/api/v2/albums/a7'))).toBe(true)
})
it('settings section renders the new page', async () => {
  renderApp('/settings/backup'); expect(await screen.findByText('settings:backup')).toBeInTheDocument()
})
it('settings about renders the new page', async () => {
  renderApp('/settings/about'); expect(await screen.findByText('settings:about')).toBeInTheDocument()
})
function crumbTrail() {
  const nav = screen.getByRole('navigation', { name: 'app.nav.breadcrumbs' })
  return Array.from(nav.querySelectorAll('a, [aria-current]')).map((e) => [e.textContent, e.getAttribute('href')])
}
it('shows the breadcrumb trail for a user page', async () => {
  renderApp('/users/lea'); await screen.findByText('page-user')
  expect(crumbTrail()).toEqual([['app.nav.users', '/users'], ['app.crumb.user', null]])
})
it('episode breadcrumb names the library, series and season', async () => {
  mockFetch({ '/api/v2/episodes/e5': episodeFixture })
  renderApp('/series/s1/episodes/e5')
  await screen.findByRole('heading', { name: 'La Fracture' })
  await waitFor(() => expect(crumbTrail()).toEqual([
    ['app.nav.libraries', '/libraries'], ['Séries', '/libraries/lib-s'], ['Station Nord', '/series/s1'],
    ['app.series.season', '/series/s1?season=2'], ['app.episode.short · La Fracture', null],
  ]))
})
it('entity breadcrumbs show the library and the entity name', async () => {
  mockFetch({ '/api/v2/series/s1': seriesFixture })
  renderApp('/series/s1')
  await screen.findByRole('heading', { name: 'Station Nord' })
  await waitFor(() => expect(crumbTrail()).toEqual([['app.nav.libraries', '/libraries'], ['Séries', '/libraries/lib-s'], ['Station Nord', null]]))
})
it('entity breadcrumbs fall back to static labels before the data loads', async () => {
  renderApp('/series/zz/episodes/e0')
  await screen.findByText('app.entity.notFound')
  expect(crumbTrail()).toEqual([['app.nav.libraries', '/libraries'], ['app.crumb.series', '/series/zz'], ['app.crumb.episode', null]])
})
it('shows the breadcrumb trail for a settings section', async () => {
  renderApp('/settings/backup'); await screen.findByText('settings:backup')
  expect(crumbTrail()).toEqual([['app.nav.settings', '/settings'], ['app.crumb.settings.backup', null]])
})

it('unknown settings section renders the 404 page', async () => {
  renderApp('/settings/bogus'); expect(await screen.findByText('app.notFound.title')).toBeInTheDocument()
})
it('/setup redirects to /login even when unauthenticated, without next', async () => {
  const router = renderApp('/setup?x=1', { token: null })
  await waitFor(() => expect(router.state.location.pathname).toBe('/login'))
  expect(router.state.location.search).toBe('')
})
it('track redirect keeps the period params', async () => {
  mockFetch({ '/api/v2/items/x1/route': { type: 'track', id: 'x1', albumId: 'a1' } })
  const router = renderApp('/items/x1?range=7d')
  await waitFor(() => expect(router.state.location.pathname).toBe('/albums/a1'))
  expect(Object.fromEntries(new URLSearchParams(router.state.location.search))).toEqual({ track: 'x1', range: '7d' })
})
it.each([['/', 'page-overview'], ['/history', 'page-history:table'], ['/users/lea', 'page-user']])('page %s shows the global period picker', async (p, text) => {
  renderApp(p)
  await screen.findByText(text)
  expect(screen.getAllByRole('button', { name: /app\.period\./ }).length).toBeGreaterThan(0)
})
// --- phase 5 ---
it('/insights mounts the insights page with the global period picker', async () => {
  renderApp('/insights')
  expect(await screen.findByRole('heading', { name: 'app.insights.title' })).toBeInTheDocument()
  expect(screen.getAllByRole('button', { name: /app\.period\./ }).length).toBeGreaterThan(0)
})
