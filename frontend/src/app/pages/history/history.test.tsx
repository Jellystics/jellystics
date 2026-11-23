import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import HistoryPage from './HistoryPage'
import { shiftMonth } from './links'
import { calendarFixture, historyFixture, usersFixture } from '../overview/fixtures'
import { renderPage, stubFetch } from '../overview/renderPage'

vi.mock('@/lib/socket', () => ({ default: { on: vi.fn(), off: vi.fn() }, useSocketEvent: vi.fn() }))
afterEach(() => { vi.unstubAllGlobals() })

const base = () => ({ '/api/v2/history': historyFixture(), '/api/v2/users': usersFixture(), '/api/v2/history/calendar': (u: URL) => calendarFixture(u.searchParams.get('month') ?? undefined) })

it('renders the table with entity links keeping the period, and the total', async () => {
  const api = stubFetch(base())
  renderPage(<HistoryPage />, { path: '/history?range=7d', route: '/history' })
  expect(await screen.findByRole('link', { name: 'Le phare' })).toHaveAttribute('href', '/series/s1/episodes/e1?range=7d')
  expect(screen.getByRole('link', { name: 'Léa' })).toHaveAttribute('href', '/users/u2?range=7d')
  expect(screen.getByRole('link', { name: 'Les Rivages' })).toHaveAttribute('href', '/movies/m1?range=7d')
  expect(screen.getByText('app.history.subtitle:2')).toBeInTheDocument()
  expect(screen.getByText('app.badge.click')).toBeInTheDocument()
  const p = api.calls('/api/v2/history')[0].searchParams
  expect(p.get('page')).toBe('1')
  expect(p.get('pageSize')).toBe('25')
  expect(p.get('sort')).toBe('startedAt')
})

it('binds filters to the URL and the request', async () => {
  const api = stubFetch(base())
  const { search } = renderPage(<HistoryPage />, { path: '/history?page=2', route: '/history' })
  await screen.findByRole('link', { name: 'Le phare' })
  await userEvent.click(screen.getByRole('button', { name: 'app.history.types.series' }))
  await waitFor(() => expect(search().get('type')).toBe('series'))
  expect(search().has('page')).toBe(false)
  await waitFor(() => expect(api.calls('/api/v2/history').some((u) => u.searchParams.get('type') === 'series')).toBe(true))
  // Clicking a client cell adds the client filter, shown as a removable chip.
  await userEvent.click(screen.getByRole('button', { name: 'Android TV' }))
  await waitFor(() => expect(search().get('client')).toBe('Android TV'))
  expect(screen.getByText('app.history.chip.client')).toBeInTheDocument()
  await userEvent.click(screen.getByRole('button', { name: 'app.history.viewsOnly' }))
  await waitFor(() => expect(api.calls('/api/v2/history').some((u) => u.searchParams.get('views') === '1' && u.searchParams.get('client') === 'Android TV')).toBe(true))
})

it('sorts on the server through ?sort / ?desc', async () => {
  const api = stubFetch(base())
  const { search } = renderPage(<HistoryPage />, { path: '/history', route: '/history' })
  await screen.findByRole('link', { name: 'Le phare' })
  await userEvent.click(screen.getByRole('button', { name: 'app.history.col.watched' }))
  await waitFor(() => expect(search().get('sort')).toBe('watchSeconds'))
  await waitFor(() => expect(api.calls('/api/v2/history').some((u) => u.searchParams.get('sort') === 'watchSeconds')).toBe(true))
})

it('shows the month calendar for ?view=calendar', async () => {
  const api = stubFetch(base())
  const { search } = renderPage(<HistoryPage />, { path: '/history?view=calendar&month=2025-09&range=7d', route: '/history' })
  expect(await screen.findByRole('link', { name: 'Station Nord' })).toHaveAttribute('href', '/series/s1?range=7d')
  expect(screen.getByText('app.history.calendar.views:5')).toBeInTheDocument()
  expect(screen.getByText('app.history.calendar.more:1')).toBeInTheDocument()
  expect(api.calls('/api/v2/history/calendar')[0].searchParams.get('month')).toBe('2025-09')
  await userEvent.click(screen.getByRole('button', { name: 'app.history.calendar.next' }))
  await waitFor(() => expect(search().get('month')).toBe('2025-10'))
})

it('shows empty and error states', async () => {
  stubFetch({ ...base(), '/api/v2/history': { rows: [], total: 0, page: 1, pageSize: 25 } })
  const first = renderPage(<HistoryPage />, { path: '/history', route: '/history' })
  expect(await screen.findByText('app.history.empty')).toBeInTheDocument()
  first.unmount()
  stubFetch({ ...base(), '/api/v2/history': { status: 500 } })
  renderPage(<HistoryPage />, { path: '/history', route: '/history' })
  expect(await screen.findByText('app.history.error')).toBeInTheDocument()
})

it('shifts months across years', () => {
  expect(shiftMonth('2026-01', -1)).toBe('2025-12')
})
