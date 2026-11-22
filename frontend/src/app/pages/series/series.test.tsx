import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import SeriesPage from './SeriesPage'
import { fixedBuckets, seasonViews, CADENCE } from './series'
import { calls, loc, mockApi, renderPage } from '../library/testing'
import { seriesFixture, seriesHistoryFixture } from '../library/fixtures'

afterEach(() => { vi.unstubAllGlobals() })
const api = () => mockApi({ '/api/v2/series/s1': seriesFixture, '/api/v2/series/s1/history': seriesHistoryFixture })
const open = (path: string) => renderPage('/series/:id', path, <SeriesPage />)

it('renders header, tiles, retention and the episodes of the first season', async () => {
  api()
  open('/series/s1?range=30d')
  expect(await screen.findByRole('heading', { name: 'Station Nord' })).toBeInTheDocument()
  expect(screen.getByText('2022 – 2025')).toBeInTheDocument()
  expect(screen.getByText('33 %')).toBeInTheDocument()
  expect(screen.getByText('14')).toBeInTheDocument()
  expect(screen.getByRole('img', { name: 'app.chart.retention' })).toBeInTheDocument()
  expect(screen.getByRole('link', { name: 'Dérive' })).toHaveAttribute('href', '/series/s1/episodes/e2?range=30d')
  expect(screen.queryByRole('link', { name: 'Hivernage' })).toBeNull()
  expect(screen.getByRole('img', { name: 'app.series.cadence.count' })).toBeInTheDocument()
})

it('season picker is bound to ?season= and abandons are flagged', async () => {
  api()
  open('/series/s1')
  await screen.findByRole('heading', { name: 'Station Nord' })
  const radios = within(screen.getByLabelText('app.series.seasonPicker')).getAllByRole('radio')
  expect(radios).toHaveLength(2)
  await userEvent.click(radios[1])
  await waitFor(() => expect(loc.current).toBe('/series/s1?season=2'))
  const link = screen.getByRole('link', { name: 'Hivernage' })
  expect(within(link.closest('tr') as HTMLElement).getByText('2')).toBeInTheDocument()
})

it('clicking a retention point opens the episode page', async () => {
  api()
  open('/series/s1?range=7d')
  await screen.findByRole('heading', { name: 'Station Nord' })
  const rects = screen.getByRole('img', { name: 'app.chart.retention' }).querySelectorAll('rect[aria-label]')
  expect(rects).toHaveLength(5)
  fireEvent.click(rects[3])
  await waitFor(() => expect(loc.current).toBe('/series/s1/episodes/e4?range=7d'))
})

it('viewers and history tabs', async () => {
  const spy = api()
  open('/series/s1?tab=viewers')
  expect(await screen.findByRole('link', { name: 'Inès' })).toHaveAttribute('href', '/users/u5')
  await userEvent.click(screen.getByRole('tab', { name: /app.entity.tab.history/ }))
  expect(await screen.findByRole('link', { name: 'La Fracture' })).toHaveAttribute('href', '/series/s1/episodes/e5')
  expect(screen.getAllByText(/app.entity.episodeCode/).length).toBeGreaterThan(0)
  expect(calls(spy, '/api/v2/series/s1/history')[0].searchParams.get('page')).toBe('1')
})

it('helpers fill fixed buckets and flag specials', () => {
  expect(fixedBuckets(CADENCE, [{ bucket: 'nextDay', count: 3 }])).toEqual([0, 0, 0, 0, 3, 0, 0])
  const v = seasonViews({ seasons: [{ number: 0, episodes: [] }, { number: 1, episodes: [] }], retention: [[1], { values: [4, 3], special: false }] as never })
  expect(v.map((s) => [s.number, s.special, s.retention])).toEqual([[0, true, [1]], [1, false, [4, 3]]])
})

it('not found', async () => {
  mockApi({})
  open('/series/zzz')
  expect(await screen.findByText('app.entity.notFound')).toBeInTheDocument()
})
