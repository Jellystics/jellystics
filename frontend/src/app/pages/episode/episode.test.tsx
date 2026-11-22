import { screen } from '@testing-library/react'
import EpisodePage from './EpisodePage'
import { calls, mockApi, renderPage } from '../library/testing'
import { episodeFixture } from '../library/fixtures'

afterEach(() => { vi.unstubAllGlobals() })
const open = (path: string) => renderPage('/series/:id/episodes/:episodeId', path, <EpisodePage />)

it('renders the trail, prev link, figures, charts and viewers', async () => {
  const spy = mockApi({ '/api/v2/episodes/e5': episodeFixture })
  open('/series/s1/episodes/e5?range=7d')
  expect(await screen.findByRole('heading', { name: 'La Fracture' })).toBeInTheDocument()
  const trail = screen.getByRole('navigation', { name: 'app.episode.trail' })
  expect(trail.querySelector('a[href="/series/s1?range=7d"]')).toHaveTextContent('Station Nord')
  expect(trail.querySelector('a[href="/series/s1?season=2&range=7d"]')).toHaveTextContent('app.series.season')
  expect(screen.getByRole('link', { name: 'app.episode.prev' })).toHaveAttribute('href', '/series/s1/episodes/e4?range=7d')
  expect(screen.queryByRole('link', { name: 'app.episode.next' })).toBeNull()
  expect(screen.getByText('86 %')).toBeInTheDocument()
  expect(screen.getByText('4 min')).toBeInTheDocument()
  expect(screen.getByRole('img', { name: 'app.chart.funnel' })).toBeInTheDocument()
  expect(screen.getByRole('link', { name: 'Malik' })).toHaveAttribute('href', '/users/u4?range=7d')
  expect(calls(spy, '/api/v2/episodes/e5')).toHaveLength(1)
})

it('error and not found states', async () => {
  mockApi({ '/api/v2/episodes/e5': { __status: 500 } })
  open('/series/s1/episodes/e5')
  expect(await screen.findByRole('alert')).toBeInTheDocument()
  vi.unstubAllGlobals()
  mockApi({})
  open('/series/s1/episodes/zz')
  expect(await screen.findByText('app.entity.notFound')).toBeInTheDocument()
})
