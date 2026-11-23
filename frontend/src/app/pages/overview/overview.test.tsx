import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import OverviewPage from './OverviewPage'
import { librariesDailyFixture, overviewFixture, playsFixture, sessionsFixture } from './fixtures'
import { renderPage, stubFetch } from './renderPage'

vi.mock('@/lib/socket', () => ({ default: { on: vi.fn(), off: vi.fn() }, useSocketEvent: vi.fn() }))

afterEach(() => { vi.unstubAllGlobals() })

it('renders the tiles, the chart, top content and users with period-keeping links', async () => {
  const api = stubFetch({ '/api/v2/overview': overviewFixture(), '/sessions/current': sessionsFixture(), '/api/v2/overview/plays': playsFixture(), '/api/v2/overview/libraries-daily': librariesDailyFixture() })
  renderPage(<OverviewPage />, { path: '/?range=7d&compare=1' })
  expect(await screen.findByText('3 412')).toBeInTheDocument()
  expect(screen.getByText('4 507')).toBeInTheDocument()
  expect(screen.getByText('71')).toBeInTheDocument()
  // Requests carry the URL period and compare.
  const p = api.calls('/api/v2/overview')[0].searchParams
  expect(p.get('compare')).toBe('1')
  expect(p.get('from')).toMatch(/^\d{4}-\d{2}-\d{2}$/)
  // Top content + users link to the typed routes and keep the period.
  expect(screen.getByRole('link', { name: /Station Nord/ })).toHaveAttribute('href', '/series/s1?range=7d&compare=1')
  expect(screen.getByRole('link', { name: /Halo/ })).toHaveAttribute('href', '/albums/al1?range=7d&compare=1')
  expect(screen.getByRole('link', { name: /Céline/ })).toHaveAttribute('href', '/users/u1?range=7d&compare=1')
  // Live card: two playing sessions (the idle one is dropped), link to /live.
  expect(await screen.findByRole('link', { name: 'Station Nord' })).toHaveAttribute('href', '/series/s1/episodes/e1?range=7d&compare=1')
  expect(screen.getByRole('link', { name: 'app.overview.seeAll' })).toHaveAttribute('href', '/live?range=7d&compare=1')
  // On this day, concurrency peak.
  expect(screen.getByRole('link', { name: 'Les Héritiers' })).toHaveAttribute('href', '/series/s2?range=7d&compare=1')
  expect(screen.getByText('app.overview.peak.title')).toBeInTheDocument()
  // Scatter cards right after the main chart.
  expect(await screen.findByText('app.overview.plays.title')).toBeInTheDocument()
  expect(await screen.findByText('app.overview.libraries.title')).toBeInTheDocument()
  expect(api.calls('/api/v2/overview/plays')[0].searchParams.has('compare')).toBe(false)
})

it('selecting a tile drives the main chart and the ?metric= param', async () => {
  stubFetch({ '/api/v2/overview': overviewFixture(), '/sessions/current': [] })
  const { search } = renderPage(<OverviewPage />)
  await screen.findByText('3 412')
  const views = screen.getByRole('button', { name: /app\.overview\.metric\.views/ })
  expect(views).toHaveAttribute('aria-pressed', 'false')
  await userEvent.click(views)
  await waitFor(() => expect(search().get('metric')).toBe('views'))
  expect(screen.getByRole('button', { name: /app\.overview\.metric\.views/ })).toHaveAttribute('aria-pressed', 'true')
  expect(screen.getByText('app.overview.perDay')).toBeInTheDocument()
})

it('filters top content by type', async () => {
  stubFetch({ '/api/v2/overview': overviewFixture(), '/sessions/current': [] })
  renderPage(<OverviewPage />)
  await screen.findByText('3 412')
  await userEvent.click(screen.getByRole('radio', { name: 'app.overview.top.movies' }))
  const card = screen.getByText('app.overview.top.title').closest('section')!
  expect(within(card).queryByText('Station Nord')).toBeNull()
  expect(within(card).getByText('Les Rivages')).toBeInTheDocument()
})

it('shows empty states when nothing was watched', async () => {
  stubFetch({
    '/api/v2/overview': { ...overviewFixture(), topItems: [], topUsers: [], onThisDay: [], byMethod: { direct: 0, stream: 0, transcode: 0 }, daily: [] },
    '/sessions/current': [],
  })
  renderPage(<OverviewPage />)
  expect(await screen.findByText('app.overview.top.empty')).toBeInTheDocument()
  expect(screen.getByText('app.overview.users.empty')).toBeInTheDocument()
  expect(screen.getByText('app.overview.onThisDay.empty')).toBeInTheDocument()
  expect(await screen.findByText('app.overview.live.empty')).toBeInTheDocument()
})

it('shows an error state with retry', async () => {
  const api = stubFetch({ '/api/v2/overview': { status: 500 }, '/sessions/current': [] })
  renderPage(<OverviewPage />)
  expect(await screen.findByText('app.overview.error')).toBeInTheDocument()
  await userEvent.click(screen.getByRole('button', { name: 'app.action.retry' }))
  await waitFor(() => expect(api.calls('/api/v2/overview').length).toBe(2))
})
