// User page: logins and "played to the end" from the Jellystics plugin.
import { screen, waitFor, within } from '@testing-library/react'
import UserPage from './UserPage'
import { historyFixture, sessionsFixture, userFixture, usersFixture } from '../overview/fixtures'
import { renderPage, stubFetch } from '../overview/renderPage'

vi.mock('@/lib/socket', () => ({ default: { on: vi.fn(), off: vi.fn() }, useSocketEvent: vi.fn() }))
afterEach(() => { vi.unstubAllGlobals() })

const LOGINS = {
  collectedSince: '2026-11-10', count: 14, lastLogin: '2026-11-22T20:41:00+01:00', failed: 2,
  devices: [{ device: 'Shield', client: 'Android TV', version: '0.18.3', count: 9, lastAt: '2026-11-22T20:41:00+01:00' }],
}
const base = (logins: unknown, user = userFixture()) => ({
  '/api/v2/users/u2': user,
  '/api/v2/users': usersFixture(),
  '/api/v2/history': historyFixture(),
  '/sessions/current': sessionsFixture(),
  '/api/v2/users/u2/logins': logins,
})
const open = (q = '') => renderPage(<UserPage />, { path: `/users/u2${q}`, route: '/users/:id' })

it('shows logins with failed attempts (admin view) and the measured finished-views foot', async () => {
  const api = stubFetch(base(LOGINS))
  open('?from=2026-11-01&to=2026-11-23')
  const logins = (await screen.findByRole('heading', { name: 'app.plugin.logins.title' })).closest('section')!
  expect(within(logins).getByText('14')).toBeInTheDocument()
  expect(screen.getByText('app.plugin.logins.failed')).toBeInTheDocument()
  expect(screen.getAllByText('Shield')).toHaveLength(2) // devices table + logins table
  expect(screen.getAllByText('app.analytics.collectedSince').length).toBeGreaterThanOrEqual(1)
  // One finished-views figure: the profile score, with its measured / estimated foot.
  expect(screen.queryByText('app.plugin.playedToEnd.label')).toBeNull()
  expect(screen.getByText('app.plugin.playedToEnd.footMixed')).toBeInTheDocument()
  expect(api.calls('/api/v2/users/u2/logins')[0].searchParams.get('from')).toBe('2026-11-01')
})

it('never shows failed attempts when the API withholds them (non-admin)', async () => {
  stubFetch(base({ ...LOGINS, failed: null }))
  open()
  expect(await screen.findByRole('heading', { name: 'app.plugin.logins.title' })).toBeInTheDocument()
  expect(screen.queryByText('app.plugin.logins.failed')).toBeNull()
})

it('hides logins and the plugin foot without plugin data', async () => {
  const user = { ...userFixture(), playedToEnd: { plays: 10, finished: 8, measured: 0, measuredFinished: 0, ratio: 0.8 }, pluginSince: null }
  const api = stubFetch(base({ ...LOGINS, collectedSince: null, count: 0, lastLogin: null, devices: [], failed: null }, user))
  open()
  expect(await screen.findByRole('heading', { name: 'Léa' })).toBeInTheDocument()
  await waitFor(() => expect(api.calls('/api/v2/users/u2/logins')).toHaveLength(1))
  expect(screen.queryByText('app.plugin.logins.title')).toBeNull()
  expect(screen.queryByText('app.plugin.playedToEnd.footMixed')).toBeNull()
})
