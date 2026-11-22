import { screen } from '@testing-library/react'
import DiscoverPage from './DiscoverPage'
import { mockApi, renderPage } from '../insights/testUtils'

const FULL = {
  '/api/v2/discover/unwatched': {
    count: 86, catalog: 344, sizeBytes: 1_200_000_000_000,
    items: [{ id: 'm9', kind: 'movie', name: 'Atlas', dateAdded: '2025-10-05', sizeBytes: 48_000_000_000 }],
  },
  '/api/v2/discover/abandoned': {
    count: 11,
    items: [
      { id: 's2', kind: 'series', name: 'Cartographes', user: { id: 'u3', name: 'Léa' }, progress: 0.1, lastPlayed: '2026-08-01T20:00:00+02:00', episodesViewed: 1, episodesTotal: 10 },
      { id: 'm3', kind: 'movie', name: 'Orbital', user: { id: 'u4', name: 'Marc' }, progress: 0.42, lastPlayed: '2026-07-01T20:00:00+02:00', positionSeconds: 3240 },
    ],
  },
  '/api/v2/discover/added': { count: 23, seen: 9, items: [] },
  '/api/v2/discover/storage': [
    { id: 'm9', kind: 'movie', name: 'Atlas', sizeBytes: 48_000_000_000, views: 0 },
    { id: 'm1', kind: 'movie', name: 'Les Rivages', sizeBytes: 18_000_000_000, views: 48 },
  ],
}

beforeEach(() => { vi.restoreAllMocks() })

it('renders tiles, lists and the storage scatter with period-keeping links', async () => {
  mockApi(FULL)
  renderPage(<DiscoverPage />, { path: '/discover', url: '/discover?range=90d' })
  expect(await screen.findByText('86')).toBeInTheDocument()
  expect(screen.getByText('11')).toBeInTheDocument()
  expect(screen.getByText('23')).toBeInTheDocument()
  expect(screen.getByText('app.discover.added.foot:9')).toBeInTheDocument()
  expect(screen.getByRole('link', { name: 'Atlas' })).toHaveAttribute('href', '/movies/m9?range=90d')
  expect(screen.getByRole('link', { name: 'Cartographes' })).toHaveAttribute('href', '/series/s2?range=90d')
  expect(screen.getByRole('link', { name: 'Marc' })).toHaveAttribute('href', '/users/u4?range=90d')
  expect(screen.getAllByRole('progressbar')).toHaveLength(2)
  expect(screen.getByRole('img', { name: /app.chart.views/ })).toBeInTheDocument()
})

it('shows empty lists and an error state for the tiles', async () => {
  mockApi({
    ...FULL,
    '/api/v2/discover/unwatched': Error,
    '/api/v2/discover/abandoned': { count: 0, items: [] },
    '/api/v2/discover/storage': [],
  })
  renderPage(<DiscoverPage />, { path: '/discover', url: '/discover' })
  expect((await screen.findAllByRole('alert')).length).toBeGreaterThanOrEqual(1)
  expect(await screen.findAllByText('app.analytics.noData')).toHaveLength(2)
})
