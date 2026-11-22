import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import UserPage from './UserPage'
import { compareVersions, outdated } from './stats'
import { historyFixture, sessionsFixture, userFixture, usersFixture } from '../overview/fixtures'
import { renderPage, stubFetch } from '../overview/renderPage'

vi.mock('@/lib/socket', () => ({ default: { on: vi.fn(), off: vi.fn() }, useSocketEvent: vi.fn() }))
afterEach(() => { vi.unstubAllGlobals() })

const base = () => ({
  '/api/v2/users/u2': userFixture(),
  '/api/v2/users/u1': userFixture('u1', 'Céline'),
  '/api/v2/users': usersFixture(),
  '/api/v2/history': historyFixture(),
  '/sessions/current': sessionsFixture(),
})
const route = '/users/:id'

it('renders the header, tiles and overview widgets', async () => {
  stubFetch(base())
  renderPage(<UserPage />, { path: '/users/u2?range=7d', route })
  expect(await screen.findByRole('heading', { name: 'Léa' })).toBeInTheDocument()
  expect(screen.getByText('app.userPage.favoriteDevice')).toBeInTheDocument()
  // Léa is watching right now (socket/REST sessions).
  expect(await screen.findByText('app.userPage.watching')).toBeInTheDocument()
  expect(screen.getByText('190')).toBeInTheDocument()
  expect(screen.getByText('app.userPage.tiles.record:23')).toBeInTheDocument()
  // In-progress shelf links to the next episode, keeping the period.
  expect(screen.getByRole('link', { name: /Station Nord/ })).toHaveAttribute('href', '/series/s1/episodes/e6?range=7d')
  expect(screen.getByRole('link', { name: /app\.userPage\.recap/ }).getAttribute('href')).toMatch(/^\/users\/u2\/recap\/\d{4}\?range=7d$/)
  // Outdated client version flagged.
  expect(screen.getByText('app.userPage.devices.outdated')).toBeInTheDocument()
  expect(screen.getByText('app.userPage.compare.empty')).toBeInTheDocument()
})

it('compares with another user through ?vs=', async () => {
  const api = stubFetch(base())
  renderPage(<UserPage />, { path: '/users/u2?vs=u1&compare=1', route })
  expect(await screen.findByRole('link', { name: /Céline/ })).toHaveAttribute('href', '/users/u1?compare=1')
  expect(api.calls('/api/v2/users/u1')).toHaveLength(1)
  expect(screen.getByText('app.userPage.compare.seriesShare')).toBeInTheDocument()
})

it('switches tabs through ?tab= (taste, history filtered by user)', async () => {
  const api = stubFetch(base())
  const { search } = renderPage(<UserPage />, { path: '/users/u2', route })
  await screen.findByRole('heading', { name: 'Léa' })
  await userEvent.click(screen.getByRole('tab', { name: 'app.userPage.tabs.taste' }))
  await waitFor(() => expect(search().get('tab')).toBe('taste'))
  expect(screen.getByText('Science-fiction')).toBeInTheDocument()
  expect(screen.getByText('app.userPage.taste.empty')).toBeInTheDocument()
  await userEvent.click(screen.getByRole('tab', { name: 'app.userPage.tabs.history' }))
  expect(await screen.findByRole('link', { name: 'Le phare' })).toBeInTheDocument()
  expect(api.calls('/api/v2/history').every((u) => u.searchParams.get('userId') === 'u2')).toBe(true)
})

it('shows not found and error states', async () => {
  stubFetch({ ...base(), '/api/v2/users/u2': { status: 404 } })
  const first = renderPage(<UserPage />, { path: '/users/u2', route })
  expect(await screen.findByText('app.userPage.notFound')).toBeInTheDocument()
  first.unmount()
  stubFetch({ ...base(), '/api/v2/users/u2': { status: 500 } })
  renderPage(<UserPage />, { path: '/users/u2', route })
  expect(await screen.findByText('app.userPage.error')).toBeInTheDocument()
})

it('version helpers', () => {
  expect(compareVersions('10.10.0', '10.9.3')).toBeGreaterThan(0)
  expect([...outdated(userFixture().devices)]).toEqual([1])
})

it('wires the VO/VF languages card on the overview', async () => {
  const api = stubFetch({
    ...base(),
    '/api/v2/users/u2/languages': { collectedSince: '2026-11-01', captured: 0, split: { originalSubtitled: 0, original: 0, dubbed: 0, unknown: 0 }, audio: [], subtitles: [], subtitleOff: 0 },
  })
  renderPage(<UserPage />, { path: '/users/u2?range=7d', route })
  expect(await screen.findByText('app.languages.title')).toBeInTheDocument()
  expect(api.calls('/api/v2/users/u2/languages').length).toBeGreaterThan(0)
})
