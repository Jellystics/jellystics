import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import MoviePage from './MoviePage'
import { calls, loc, mockApi, renderPage } from '../library/testing'
import { movieFixture, sittingsFixture } from '../library/fixtures'

afterEach(() => { vi.unstubAllGlobals() })
const open = (path: string) => renderPage('/movies/:id', path, <MoviePage />)

it('renders header, key figures, charts and viewers with period links', async () => {
  const spy = mockApi({ '/api/v2/movies/m1': movieFixture, '/api/v2/movies/m1/sittings': sittingsFixture })
  open('/movies/m1?range=90d')
  expect(await screen.findByRole('heading', { name: 'Les Rivages' })).toBeInTheDocument()
  expect(screen.getByText('4K · 18,2 Go')).toBeInTheDocument()
  expect(screen.getAllByText('48').length).toBeGreaterThan(0)
  expect(screen.getByText('60 %')).toBeInTheDocument()
  expect(screen.getByText('app.movie.funnel')).toBeInTheDocument()
  expect(screen.getByText('app.movie.stops')).toBeInTheDocument()
  expect(screen.getByRole('link', { name: 'Céline' })).toHaveAttribute('href', '/users/u1?range=90d')
  expect(await screen.findByRole('img', { name: 'app.movie.sittings.share' })).toBeInTheDocument()
  expect(calls(spy, '/api/v2/movies/m1')[0].searchParams.get('from')).toMatch(/^\d{4}-\d{2}-\d{2}$/)
})

it('history tab is bound to ?tab=', async () => {
  mockApi({ '/api/v2/movies/m1': movieFixture, '/api/v2/movies/m1/sittings': sittingsFixture })
  open('/movies/m1')
  await screen.findByRole('heading', { name: 'Les Rivages' })
  await userEvent.click(screen.getByRole('tab', { name: /app.entity.tab.history/ }))
  await waitFor(() => expect(loc.current).toBe('/movies/m1?tab=history'))
  expect(screen.getByText('Hugo')).toBeInTheDocument()
  expect(screen.getAllByText('app.method.transcode').length).toBeGreaterThan(0)
})

it('shows empty sittings and empty viewers', async () => {
  mockApi({
    '/api/v2/movies/m1': { ...movieFixture, viewers: [], history: [] },
    '/api/v2/movies/m1/sittings': sittingsFixture.map((b) => ({ ...b, count: 0 })),
  })
  open('/movies/m1')
  expect(await screen.findByText('app.entity.noViewers')).toBeInTheDocument()
  expect(await screen.findByText('app.movie.sittings.empty')).toBeInTheDocument()
})

it('links the library and shows a retryable error for a failed widget', async () => {
  mockApi({ '/api/v2/movies/m1': movieFixture, '/api/v2/movies/m1/sittings': { __status: 500 } })
  open('/movies/m1')
  expect(await screen.findByRole('link', { name: 'Films' })).toHaveAttribute('href', '/libraries/lib-m')
  expect(await screen.findByRole('alert')).toHaveTextContent('app.action.retry')
})

it('shows an error with retry, and not found on 404', async () => {
  mockApi({ '/api/v2/movies/m1': { __status: 500 } })
  open('/movies/m1')
  expect(await screen.findByRole('alert')).toHaveTextContent('app.entity.error')
  vi.unstubAllGlobals()
  mockApi({})
  open('/movies/zzz')
  expect(await screen.findByText('app.entity.notFound')).toBeInTheDocument()
})
