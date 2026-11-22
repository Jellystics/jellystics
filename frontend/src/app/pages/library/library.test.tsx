import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import LibraryPage from './LibraryPage'
import { calls, loc, mockApi, renderPage } from './testing'
import { decadesFixture, growthFixture, libraryFixture, seriesHistoryFixture } from './fixtures'
import { formatBytes, entityPath, playMethod } from './entity'

afterEach(() => { vi.unstubAllGlobals() })

const api = () => mockApi({
  '/api/v2/libraries/lib-s': libraryFixture,
  '/api/v2/libraries/lib-s/decades': decadesFixture,
  '/api/v2/libraries/lib-s/growth': growthFixture,
  '/api/v2/history': seriesHistoryFixture,
})
const open = (path: string) => renderPage('/libraries/:id', path, <LibraryPage />)

it('renders header, tiles and the items table with period-keeping links', async () => {
  const spy = api()
  open('/libraries/lib-s?range=7d')
  expect(await screen.findByRole('heading', { name: 'Séries' })).toBeInTheDocument()
  expect(screen.getByText('app.libraries.items.series:4 · 412 Go')).toBeInTheDocument()
  expect(screen.getByText('app.library.ofCatalog')).toBeInTheDocument()
  const row = screen.getByRole('link', { name: /Station Nord/ })
  expect(row).toHaveAttribute('href', '/series/s1?range=7d')
  expect(screen.getAllByRole('progressbar', { name: 'app.library.col.retention' })[0]).toHaveAttribute('aria-valuenow', '33')
  const q = calls(spy, '/api/v2/libraries/lib-s')[0].searchParams
  expect(q.get('page')).toBe('1')
  expect(q.get('pageSize')).toBe('50')
})

it('sends the page size chosen in the URL', async () => {
  const spy = api()
  open('/libraries/lib-s?pageSize=100')
  expect(await screen.findByRole('heading', { name: 'Séries' })).toBeInTheDocument()
  expect(calls(spy, '/api/v2/libraries/lib-s')[0].searchParams.get('pageSize')).toBe('100')
  expect(screen.getByRole('combobox', { name: 'app.table.perPage' })).toHaveTextContent('100')
})

it('sorts through the URL and sends sort/desc', async () => {
  const spy = api()
  open('/libraries/lib-s')
  await screen.findByRole('link', { name: /Station Nord/ })
  await userEvent.click(screen.getByRole('button', { name: 'app.entity.col.viewers' }))
  await waitFor(() => expect(loc.current).toContain('sort=viewers'))
  await waitFor(() => expect(calls(spy, '/api/v2/libraries/lib-s').some((u) => u.searchParams.get('sort') === 'viewers')).toBe(true))
})

it('switches to the grid layout', async () => {
  api()
  open('/libraries/lib-s?layout=grid')
  const card = await screen.findByRole('link', { name: /Brigade 9/ })
  expect(card).toHaveTextContent('app.library.viewsCount:198')
  expect(screen.queryByRole('table')).toBeNull()
})

it('search updates ?q= after a debounce', async () => {
  const spy = api()
  open('/libraries/lib-s')
  await screen.findByRole('link', { name: /Station Nord/ })
  await userEvent.type(screen.getByRole('searchbox', { name: 'app.library.search' }), 'nord')
  await waitFor(() => expect(loc.current).toContain('q=nord'))
  await waitFor(() => expect(calls(spy, '/api/v2/libraries/lib-s').some((u) => u.searchParams.get('q') === 'nord')).toBe(true))
})

it('stats tab shows genres, methods, decades and growth', async () => {
  const spy = api()
  open('/libraries/lib-s?tab=stats')
  expect(await screen.findByText('Science-fiction')).toBeInTheDocument()
  expect(await screen.findByRole('img', { name: 'app.library.decades.title' })).toHaveTextContent('app.library.decades.label')
  expect(screen.getByRole('link', { name: 'Le Phare' })).toHaveAttribute('href', '/series/s4')
  expect(await screen.findByText('app.library.growth.title')).toBeInTheDocument()
  await waitFor(() => expect(calls(spy, '/api/v2/libraries/lib-s/growth')).toHaveLength(1))
})

it('activity tab lists the library history', async () => {
  const spy = api()
  open('/libraries/lib-s?tab=activity')
  const table = await screen.findByRole('table')
  expect(await within(table).findByRole('link', { name: 'La Fracture' })).toHaveAttribute('href', '/series/s1/episodes/e5')
  expect(calls(spy, '/api/v2/history')[0].searchParams.get('libraryId')).toBe('lib-s')
})

it('activity tab shows a retryable error instead of an empty table', async () => {
  mockApi({ '/api/v2/libraries/lib-s': libraryFixture, '/api/v2/history': { __status: 500 } })
  open('/libraries/lib-s?tab=activity')
  expect(await screen.findByRole('alert')).toHaveTextContent('app.entity.error')
  expect(screen.queryByRole('table')).toBeNull()
})

it('shows not found for an unknown library', async () => {
  mockApi({})
  open('/libraries/nope')
  expect(await screen.findByText('app.entity.notFound')).toBeInTheDocument()
})

it('entity helpers', () => {
  expect(formatBytes(18_200_000_000, 'fr')).toBe('18,2 Go')
  expect(formatBytes(950, 'en')).toBe('950 B')
  expect(entityPath('track', 't1', 'a1')).toBe('/albums/a1?track=t1')
  expect(entityPath('episode', 'e1', 's 1')).toBe('/series/s%201/episodes/e1')
  expect(playMethod('DirectStream')).toBe('stream')
  expect(playMethod('Unknown')).toBeNull()
})
