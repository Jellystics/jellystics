import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import ArtistPage from './ArtistPage'
import { loc, mockApi, renderPage } from '../library/testing'
import { artistFixture, trackFixture } from '../library/fixtures'

afterEach(() => { vi.unstubAllGlobals() })
const open = (path: string) => renderPage('/artists/:id', path, <ArtistPage />)

it('renders figures, top tracks with skip, albums and skip by hour', async () => {
  mockApi({ '/api/v2/artists/ar1': artistFixture })
  open('/artists/ar1?range=ytd')
  expect(await screen.findByRole('heading', { name: 'Lune Pâle' })).toBeInTheDocument()
  expect(screen.getByText('612')).toBeInTheDocument()
  expect(screen.getByText('19 %')).toBeInTheDocument()
  const skips = screen.getAllByRole('progressbar', { name: 'app.entity.skip' })
  expect(skips[2]).toHaveAttribute('aria-valuenow', '36')
  expect(screen.getAllByRole('link', { name: 'Halo' })[0]).toHaveAttribute('href', '/albums/al1?range=ytd')
  expect(screen.getByRole('link', { name: /Rivière noire.*app.artist.albumMeta/ })).toHaveAttribute('href', '/albums/al2?range=ytd')
  expect(screen.getByRole('img', { name: 'app.entity.skipRate' })).toBeInTheDocument()
})

it('a top track opens the track panel', async () => {
  mockApi({ '/api/v2/artists/ar1': artistFixture, '/api/v2/tracks/t3': trackFixture })
  open('/artists/ar1')
  await screen.findByRole('heading', { name: 'Lune Pâle' })
  await userEvent.click(screen.getByRole('link', { name: /3\s*Basse lumière/ }))
  await waitFor(() => expect(loc.current).toBe('/artists/ar1?track=t3'))
  expect(await screen.findByRole('dialog', { name: 'Basse lumière' })).toHaveTextContent('Halo')
})

it('empty and error states', async () => {
  mockApi({ '/api/v2/artists/ar1': { ...artistFixture, topTracks: [], albums: [], skipByHour: [] } })
  open('/artists/ar1')
  expect(await screen.findByText('app.artist.noTracks')).toBeInTheDocument()
  expect(screen.getByText('app.artist.noAlbums')).toBeInTheDocument()
  expect(screen.getByText('app.artist.noSkips')).toBeInTheDocument()
  vi.unstubAllGlobals()
  mockApi({ '/api/v2/artists/ar1': { __status: 500 } })
  open('/artists/ar1')
  expect(await screen.findByRole('alert')).toBeInTheDocument()
})
