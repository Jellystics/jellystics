import { screen } from '@testing-library/react'
import EpisodePage from './EpisodePage'
import { calls, mockApi, renderPage } from '../library/testing'
import { episodeFixture } from '../library/fixtures'

afterEach(() => { vi.unstubAllGlobals() })
const open = (path: string) => renderPage('/series/:id/episodes/:episodeId', path, <EpisodePage />)

it('renders the kicker, prev link, figures, charts and viewers', async () => {
  const spy = mockApi({ '/api/v2/episodes/e5': episodeFixture })
  open('/series/s1/episodes/e5?range=7d')
  expect(await screen.findByRole('heading', { name: 'La Fracture' })).toBeInTheDocument()
  expect(screen.getByRole('link', { name: 'Séries' })).toHaveAttribute('href', '/libraries/lib-s?range=7d')
  expect(screen.getByRole('link', { name: 'Station Nord' })).toHaveAttribute('href', '/series/s1?range=7d')
  // Season / episode live in the breadcrumb only.
  expect(screen.queryByRole('link', { name: 'app.series.season' })).toBeNull()
  const prev = screen.getByRole('link', { name: 'app.episode.prev' })
  expect(prev).toHaveAttribute('href', '/series/s1/episodes/e4?range=7d')
  expect(prev).toHaveTextContent('app.episode.short')
  expect(screen.queryByRole('link', { name: 'app.episode.next' })).toBeNull()
  expect(screen.getByText('86 %')).toBeInTheDocument()
  expect(screen.getByText('4 min')).toBeInTheDocument()
  expect(screen.getByRole('img', { name: 'app.chart.funnel' })).toBeInTheDocument()
  expect(screen.getByRole('link', { name: 'Malik' })).toHaveAttribute('href', '/users/u4?range=7d')
  expect(calls(spy, '/api/v2/episodes/e5')).toHaveLength(1)
})

it('labels neighbours in another season with their season', async () => {
  const item = { ...episodeFixture.item, season: 2, episode: 1, prev: { id: 'e3', season: 1, episode: 10 }, next: { id: 'e5', season: 2, episode: 2 } }
  mockApi({ '/api/v2/episodes/e4': { ...episodeFixture, item: { ...item, id: 'e4' } } })
  open('/series/s1/episodes/e4')
  const prev = await screen.findByRole('link', { name: 'app.episode.prev' })
  expect(prev).toHaveAttribute('href', '/series/s1/episodes/e3')
  expect(prev).toHaveTextContent('app.entity.episodeCode')
  expect(screen.getByRole('link', { name: 'app.episode.next' })).toHaveTextContent('app.episode.short')
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
