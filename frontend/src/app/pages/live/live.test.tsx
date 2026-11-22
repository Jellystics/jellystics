import { act, screen, waitFor } from '@testing-library/react'
import LivePage from './LivePage'
import { slotLabels, timelineRows } from './timeline'
import { concurrencyFixture, sessionsFixture } from '../overview/fixtures'
import { renderPage, stubFetch } from '../overview/renderPage'

const socket = vi.hoisted(() => ({ handlers: {} as Record<string, (d: unknown) => void> }))
vi.mock('@/lib/socket', () => ({
  default: { on: vi.fn(), off: vi.fn() },
  useSocketEvent: (event: string, handler: (d: unknown) => void) => { socket.handlers[event] = handler },
}))

afterEach(() => { vi.unstubAllGlobals() })

it('shows the live tiles, session cards and today’s charts', async () => {
  const api = stubFetch({ '/sessions/current': sessionsFixture(), '/api/v2/concurrency': concurrencyFixture() })
  renderPage(<LivePage />, { path: '/live?range=7d', route: '/live' })
  expect((await screen.findAllByRole('link', { name: 'Station Nord' }))[0]).toHaveAttribute('href', '/series/s1/episodes/e1?range=7d')
  expect(screen.getAllByRole('link', { name: 'Les Rivages' })[0]).toHaveAttribute('href', '/movies/m1?range=7d')
  expect(screen.getByText('app.live.playing', { selector: 'span' }).nextSibling).toHaveTextContent('2')
  expect(screen.getByText('app.live.transcodes', { selector: 'span' }).nextSibling).toHaveTextContent('1')
  // 12 + 21 Mb/s outgoing.
  expect(screen.getByText('app.live.bandwidth', { selector: 'span' }).nextSibling).toHaveTextContent('33')
  expect(await screen.findByText('app.live.simultaneous')).toBeInTheDocument()
  expect(screen.getByText('app.live.timeline')).toBeInTheDocument()
  // The day endpoint is not tied to the period.
  expect(api.calls('/api/v2/concurrency')[0].searchParams.has('from')).toBe(false)
})

it('updates from the socket `sessions` event', async () => {
  stubFetch({ '/sessions/current': sessionsFixture(), '/api/v2/concurrency': concurrencyFixture() })
  renderPage(<LivePage />, { path: '/live', route: '/live' })
  await screen.findAllByRole('link', { name: 'Station Nord' })
  act(() => socket.handlers.sessions?.([sessionsFixture()[1]]))
  await waitFor(() => expect(screen.queryByRole('link', { name: 'Station Nord' })).toBeNull())
  expect(screen.getAllByRole('link', { name: 'Les Rivages' }).length).toBeGreaterThan(0)
})

it('shows an empty state when nothing plays', async () => {
  stubFetch({ '/sessions/current': [], '/api/v2/concurrency': { ...concurrencyFixture(), sessions: [], slots: Array(96).fill(0), peak: 0, peakAt: null } })
  renderPage(<LivePage />, { path: '/live', route: '/live' })
  expect(await screen.findByText('app.live.empty')).toBeInTheDocument()
  expect(await screen.findByText('app.live.timelineEmpty')).toBeInTheDocument()
})

it('shows error states with retry', async () => {
  stubFetch({ '/sessions/current': { status: 500 }, '/api/v2/concurrency': { status: 500 } })
  renderPage(<LivePage />, { path: '/live', route: '/live' })
  expect(await screen.findByText('app.live.error')).toBeInTheDocument()
  expect(await screen.findByText('app.live.concurrencyError')).toBeInTheDocument()
  expect(screen.getAllByRole('button', { name: 'app.action.retry' })).toHaveLength(2)
})

it('timelineRows groups spans per user and frames the hours', () => {
  const { rows, fromHour, toHour } = timelineRows(concurrencyFixture(), [], 22.5)
  expect(rows.map((r) => r.user)).toEqual(['Léa', 'Marc'])
  expect(rows[0].spans[0]).toMatchObject({ start: 20, type: 'series', label: 'Station Nord · Le phare' })
  expect(fromHour).toBe(18)
  expect(toHour).toBe(22)
})

it('slotLabels follows the local clock on DST days (92 slots)', () => {
  const labels = slotLabels('2026-03-29', 92, 15, 'Europe/Paris')
  expect(labels).toHaveLength(92)
  expect(labels[0]).toBe('00:00')
  expect(labels[7]).toBe('01:45')
  expect(labels[8]).toBe('03:00')
  expect(labels[91]).toBe('23:45')
  expect(slotLabels('2026-11-22', 96, 15, 'Europe/Paris')[95]).toBe('23:45')
})

it('timelineRows drops a live session already reported by /concurrency', () => {
  const day = concurrencyFixture()
  const base = sessionsFixture()[1]
  const live = [{ ...base, UserId: 'u3', UserName: 'Marc', NowPlayingItem: { ...base.NowPlayingItem!, Id: 'm1' } }]
  const { rows } = timelineRows(day, live, 21.5)
  expect(rows.find((r) => r.user === 'Marc')!.spans).toHaveLength(1)
})
