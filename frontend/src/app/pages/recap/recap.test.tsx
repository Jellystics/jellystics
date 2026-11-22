import { screen } from '@testing-library/react'
import RecapPage from './RecapPage'
import { recapOptions } from './recapOptions'
import { CrumbLabelContext } from '@/app/shell/crumbs'
import { mockApi, renderPage } from '../_shared/testUtils'

const months = Array.from({ length: 12 }, (_, i) => `2025-${String(i + 1).padStart(2, '0')}`)
const RECAP = {
  user: { id: 'u3', name: 'Léa' }, period: '2025', granularity: 'month', from: '2025-01-01', to: '2026-01-01',
  totals: { watchSeconds: 6_523_200, views: 5210, activeDays: 312 }, previous: { watchSeconds: 0, views: 0, activeDays: 0 },
  topSeries: [{ id: 's1', name: 'Station Nord', watchSeconds: 1_036_800, views: 300 }],
  topMovies: [{ id: 'm1', name: 'Les Rivages', watchSeconds: 20_000, views: 3 }],
  topArtists: [{ id: 'ar1', name: 'Lune Pâle', watchSeconds: 133_200, views: 900 }],
  buckets: months.map((k, i) => ({ key: k, watchSeconds: i === 7 ? 723_600 : 360_000, views: 10, newArtists: i === 4 ? 6 : 1 })),
  records: { bestBucket: { key: '2025-08', watchSeconds: 723_600, views: 10, newArtists: 1 }, binge: { seriesId: 's1', seriesName: 'Station Nord', episodes: 9, date: '2025-08-14' }, longestStreak: 23 },
  ranking: [
    { id: 'al1', kind: 'album', name: 'Rivière noire', parentName: 'Lune Pâle', views: 182, rank: 1, previousRank: 3 },
    { id: 'al2', kind: 'album', name: 'Sel', parentName: 'Mira Okafor', views: 160, rank: 2, previousRank: 1 },
    { id: 's1', kind: 'series', name: 'Station Nord', parentName: null, views: 141, rank: 3, previousRank: null },
  ],
}

beforeEach(() => { vi.restoreAllMocks() })

it('renders the hero, tops, records and the ranking with deltas', async () => {
  const spy = mockApi({ '/api/v2/recap/u3/2025': RECAP })
  renderPage(<RecapPage />, { path: '/users/:id/recap/:period', url: '/users/u3/recap/2025?range=30d' })
  expect(await screen.findByText('app.recap.hoursValue')).toBeInTheDocument()
  expect(screen.getByRole('link', { name: 'Léa' })).toHaveAttribute('href', '/users/u3?range=30d')
  expect(screen.getAllByRole('link', { name: /Station Nord/ })[0]).toHaveAttribute('href', '/series/s1?range=30d')
  expect(screen.getByRole('link', { name: /Lune Pâle/ })).toHaveAttribute('href', '/artists/ar1?range=30d')
  expect(screen.getByRole('link', { name: 'Rivière noire' })).toHaveAttribute('href', '/albums/al1?range=30d')
  expect(screen.getByText('app.recap.episodes:9')).toBeInTheDocument()
  expect(screen.getByText('app.recap.days:23')).toBeInTheDocument()
  expect(screen.getByText('app.recap.ranking.up')).toBeInTheDocument()
  expect(screen.getByText('app.recap.ranking.down')).toBeInTheDocument()
  expect(screen.getByText('app.recap.ranking.new')).toBeInTheDocument()
  // Period switch: year + months, current one marked.
  expect(screen.getByRole('link', { name: '2025' })).toHaveAttribute('aria-current', 'page')
  expect(spy.mock.calls.every(([u]) => !String(u).includes('from='))).toBe(true)
})

it('handles unknown users, invalid periods and errors', async () => {
  mockApi({ '/api/v2/recap/u3/2025-02': Error })
  const { unmount } = renderPage(<RecapPage />, { path: '/users/:id/recap/:period', url: '/users/nobody/recap/2025' })
  expect(await screen.findByText('app.recap.notFound')).toBeInTheDocument()
  unmount()
  const bad = renderPage(<RecapPage />, { path: '/users/:id/recap/:period', url: '/users/u3/recap/2025-13' })
  expect(screen.getByText('app.recap.badPeriod')).toBeInTheDocument()
  bad.unmount()
  renderPage(<RecapPage />, { path: '/users/:id/recap/:period', url: '/users/u3/recap/2025-02' })
  expect(await screen.findByRole('button', { name: 'app.action.retry' })).toBeInTheDocument()
})

it('shows an empty state when the period has no activity', async () => {
  mockApi({ '/api/v2/recap/u3/2025': { ...RECAP, totals: { watchSeconds: 0, views: 0, activeDays: 0 } } })
  renderPage(<RecapPage />, { path: '/users/:id/recap/:period', url: '/users/u3/recap/2025' })
  expect(await screen.findByText('app.analytics.noData')).toBeInTheDocument()
})

it('lists the year and its latest months, never in the future', () => {
  const now = new Date(2026, 10, 22)
  expect(recapOptions('2026', now)).toEqual(['2026', '2026-11', '2026-10', '2026-09'])
  expect(recapOptions('2025-03', now)).toEqual(['2025', '2025-12', '2025-11', '2025-10', '2025-03'])
  expect(recapOptions('2027', now)).toEqual(['2027'])
})

it('uses the server time zone for the current month', () => {
  const now = new Date(Date.UTC(2026, 10, 30, 23, 30)) // 30 Nov 23:30 UTC = 1 Dec in Kiritimati (UTC+14)
  expect(recapOptions('2026', now, 'UTC')).toEqual(['2026', '2026-11', '2026-10', '2026-09'])
  expect(recapOptions('2026', now, 'Pacific/Kiritimati')).toEqual(['2026', '2026-12', '2026-11', '2026-10'])
})

it('shows the user name and the period in the breadcrumb', async () => {
  mockApi({ '/api/v2/recap/u3/2025': RECAP })
  const set = vi.fn()
  renderPage(
    <CrumbLabelContext.Provider value={{ label: null, set }}><RecapPage /></CrumbLabelContext.Provider>,
    { path: '/users/:id/recap/:period', url: '/users/u3/recap/2025' },
  )
  await screen.findByText('app.recap.hoursValue')
  expect(set).toHaveBeenLastCalledWith(['Léa', 'app.recap.crumb'])
})

it('labels the ranking columns for screen readers', async () => {
  mockApi({ '/api/v2/recap/u3/2025': RECAP })
  renderPage(<RecapPage />, { path: '/users/:id/recap/:period', url: '/users/u3/recap/2025' })
  expect(await screen.findByRole('columnheader', { name: 'app.recap.ranking.rank' })).toBeInTheDocument()
})
