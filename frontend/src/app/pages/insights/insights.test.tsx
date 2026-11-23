import { screen, within } from '@testing-library/react'
import InsightsPage from './InsightsPage'
import { mockApi, renderPage } from '../_shared/testUtils'
import { formatBitrate } from '../_shared/analyticsFormat'

const zeros = (n: number) => Array.from({ length: n }, () => 0)
const week = () => Array.from({ length: 7 }, () => zeros(24))

const FULL = {
  '/api/v2/insights/completion': { distribution: [2, 0, 0, 1, 0, 0, 0, 0, 0, 7], byType: { movie: 0.64, series: 0.81, music: null }, units: 10 },
  '/api/v2/insights/sessions': { buckets: [4, 4, 0, 0, 0, 0, 0, 0, 0], sessions: 8, medianSeconds: 70 },
  '/api/v2/insights/resumes': { fromStart: 71, resumed: 29, byType: { movie: 0.41, series: 0.22, music: null } },
  '/api/v2/insights/rewatches': [{ id: 'e1', kind: 'episode', name: 'Pilot', parentId: 's1', parentName: 'Station Nord', views: 3, user: { id: 'u1', name: 'Léa' } }],
  '/api/v2/insights/genres': { bucket: 'month', months: ['2026-10', '2026-11'], series: [{ name: 'Drame', values: [3600, 7200] }], other: [0, 1800] },
  '/api/v2/insights/moments': { movie: zeros(24).map((_, h) => (h === 21 ? 5 : 0)), series: zeros(24), music: zeros(24) },
  '/api/v2/insights/pauses': [{ id: 'm1', kind: 'movie', name: 'Orbital', ratio: 0.18, pausedSeconds: 600, pauseCount: 4, watchSeconds: 3333 }],
  '/api/v2/insights/binges': [{ id: 's1', name: 'Station Nord', binges: 14, users: 2 }],
  '/api/v2/insights/rhythm': { weekHour: week().map((r, i) => (i === 0 ? r.map((_, h) => (h === 20 ? 3 : 0)) : r)) },
}

beforeEach(() => { vi.restoreAllMocks() })

it('renders every insight with links that keep the period', async () => {
  mockApi(FULL)
  renderPage(<InsightsPage />, { path: '/insights', url: '/insights?range=90d' })

  expect(await screen.findByRole('heading', { name: 'app.insights.title' })).toBeInTheDocument()
  const binge = await screen.findByRole('link', { name: /Station Nord/ })
  expect(binge).toHaveAttribute('href', '/series/s1?range=90d')
  expect(screen.getByText('14')).toBeInTheDocument()
  expect(screen.getByRole('link', { name: /Orbital/ })).toHaveAttribute('href', '/movies/m1?range=90d')
  expect(screen.getByText(/^18\s%$/)).toBeInTheDocument()
  // Rewatch: episode link under its series, user link.
  expect(screen.getByRole('link', { name: 'Pilot' })).toHaveAttribute('href', '/series/s1/episodes/e1?range=90d')
  expect(screen.getByRole('link', { name: 'Léa' })).toHaveAttribute('href', '/users/u1?range=90d')
  // Completion by type skips families without data.
  expect(screen.getByText(/^64\s%$/)).toBeInTheDocument()
  expect(screen.getByText(/^81\s%$/)).toBeInTheDocument()
  expect(screen.getByText('app.insights.sessions.median')).toBeInTheDocument()
  expect(screen.getByRole('img', { name: /app.chart.views/ })).toBeInTheDocument()
})

it('sends the URL period to every insight endpoint', async () => {
  const spy = mockApi(FULL)
  renderPage(<InsightsPage />, { path: '/insights', url: '/insights?from=2026-09-01&to=2026-10-01' })
  await screen.findByRole('link', { name: /Station Nord/ })
  const urls = spy.mock.calls.map(([u]) => new URL(String(u), 'http://x'))
  const insights = urls.filter((u) => u.pathname.startsWith('/api/v2/insights/'))
  expect(insights).toHaveLength(9)
  insights.forEach((u) => {
    expect(u.searchParams.get('from')).toBe('2026-09-01')
    expect(u.searchParams.get('to')).toBe('2026-10-01')
  })
})

it('shows empty states without data and an error state with retry', async () => {
  mockApi({
    ...FULL,
    '/api/v2/insights/completion': { distribution: zeros(10), byType: { movie: null, series: null, music: null }, units: 0 },
    '/api/v2/insights/binges': [],
    '/api/v2/insights/pauses': Error,
  })
  renderPage(<InsightsPage />, { path: '/insights', url: '/insights' })
  expect(await screen.findByRole('alert')).toBeInTheDocument()
  expect(within(screen.getByRole('alert')).getByRole('button', { name: 'app.action.retry' })).toBeInTheDocument()
  expect((await screen.findAllByText('app.analytics.noData')).length).toBeGreaterThanOrEqual(3)
})

it('formats helpers', () => {
  expect(formatBitrate(8_500_000, 'en')).toBe('8.5 Mb/s')
})
