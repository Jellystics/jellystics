import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import AlbumPage from './AlbumPage'
import { calls, loc, mockApi, renderPage } from '../library/testing'
import { albumFixture, fullListensFixture, trackFixture } from '../library/fixtures'

afterEach(() => { vi.unstubAllGlobals() })
const api = () => mockApi({ '/api/v2/albums/al1': albumFixture, '/api/v2/albums/al1/full-listens': fullListensFixture, '/api/v2/tracks/t3': trackFixture })
const open = (path: string) => renderPage('/albums/:id', path, <AlbumPage />)

it('renders header, figures, tracks with skip and the full-listen card', async () => {
  api()
  open('/albums/al1?range=30d')
  expect(await screen.findByRole('heading', { name: 'Halo' })).toBeInTheDocument()
  expect(screen.getByRole('link', { name: 'Lune Pâle' })).toHaveAttribute('href', '/artists/ar1?range=30d')
  expect(screen.getByText('app.album.wholeAlbum:9')).toBeInTheDocument()
  expect(screen.getByText('62 %')).toBeInTheDocument()
  const skip = screen.getAllByRole('progressbar', { name: 'app.entity.skip' })
  expect(skip).toHaveLength(3)
  expect(skip[2]).toHaveAttribute('aria-valuenow', '36')
  expect(await screen.findByRole('link', { name: /Rivière noire/ })).toHaveAttribute('href', '/albums/al2?range=30d')
  expect(screen.queryByRole('dialog')).toBeNull()
})

it('clicking a track row opens the side panel bound to ?track=', async () => {
  const spy = api()
  open('/albums/al1?range=30d')
  await screen.findByRole('heading', { name: 'Halo' })
  await userEvent.click(screen.getByRole('link', { name: /Basse lumière/ }))
  await waitFor(() => expect(loc.current).toBe('/albums/al1?track=t3&range=30d'))
  const dialog = await screen.findByRole('dialog', { name: 'Basse lumière' })
  expect(within(dialog).getByRole('link', { name: 'Lune Pâle' })).toHaveAttribute('href', '/artists/ar1?range=30d')
  expect(await within(dialog).findByText('96')).toBeInTheDocument()
  expect(within(dialog).getByRole('link', { name: 'Hugo' })).toHaveAttribute('href', '/users/u2?range=30d')
  expect(calls(spy, '/api/v2/tracks/t3')[0].searchParams.get('from')).toMatch(/^\d{4}-/)
  await userEvent.keyboard('{Escape}')
  await waitFor(() => expect(loc.current).toBe('/albums/al1?range=30d'))
  expect(screen.queryByRole('dialog')).toBeNull()
})

it('opens directly from the URL', async () => {
  api()
  open('/albums/al1?track=t3')
  expect(await screen.findByRole('dialog', { name: 'Basse lumière' })).toBeInTheDocument()
})

it('empty full listens and not found', async () => {
  mockApi({ '/api/v2/albums/al1': { ...albumFixture, tracks: [] }, '/api/v2/albums/al1/full-listens': [] })
  open('/albums/al1')
  expect(await screen.findByText('app.album.noTracks')).toBeInTheDocument()
  expect(await screen.findByText('app.album.full.empty')).toBeInTheDocument()
  vi.unstubAllGlobals()
  mockApi({})
  open('/albums/zz')
  expect(await screen.findByText('app.entity.notFound')).toBeInTheDocument()
})
