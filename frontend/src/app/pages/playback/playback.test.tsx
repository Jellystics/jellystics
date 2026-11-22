import { screen } from '@testing-library/react'
import PlaybackPage from './PlaybackPage'
import { mockApi, renderPage } from '../_shared/testUtils'

const QUALITY = {
  collectedSince: '2026-11-01', captured: 40, views: 100, transcodes: 20,
  resolution: { served: { '4K': 18, '1080p': 61, '720p': 15, SD: 6 }, sourceUhd: 25, downscaledUhd: 9 },
  hdr: { native: 11, toneMapped: 9, sdr: 80 },
  reasons: [{ label: 'VideoRangeTypeNotSupported', views: 8 }, { label: 'SomethingNew', views: 2 }],
  conversions: [{ videoFrom: 'HEVC', videoTo: 'H264', audioFrom: 'TRUEHD', audioTo: 'AAC', views: 12 }, { videoFrom: 'H264', videoTo: 'copy', audioFrom: 'DTS', audioTo: 'AAC', views: 5 }],
  codecPairs: [{ videoFrom: 'HEVC', videoTo: 'H264', views: 12 }, { videoFrom: 'H264', videoTo: 'copy', views: 5 }, { videoFrom: 'VP9', videoTo: 'H264', views: 1 }],
  streams: { audioOnly: 5, videoOnly: 3, both: 12 },
  hwAccel: [{ label: 'vaapi', views: 15 }],
  bitrate: { samples: 40, avg: 8_500_000, p95: 30_000_000, buckets: [2, 8, 20, 6, 3, 1] },
}
const FULL = {
  '/api/v2/playback/methods': {
    bucket: 'day', totals: { direct: 68, stream: 14, transcode: 18 }, previous: { direct: 63, stream: 15, transcode: 22 },
    daily: [{ date: '2026-11-20', direct: 30, stream: 7, transcode: 8 }, { date: '2026-11-21', direct: 38, stream: 7, transcode: 10 }],
  },
  '/api/v2/playback/clients': [
    { client: 'Jellyfin Web', clicks: 50, views: 40, transcodes: 25, transcodeRatio: 0.625, falseStarts: 10, falseStartRatio: 0.2 },
    { client: 'Android TV', clicks: 60, views: 60, transcodes: 0, transcodeRatio: 0, falseStarts: 0, falseStartRatio: 0 },
    { client: 'Rare TV', clicks: 3, views: 3, transcodes: 3, transcodeRatio: 1, falseStarts: 0, falseStartRatio: 0 },
  ],
  '/api/v2/playback/quality': QUALITY,
  '/api/v2/playback/network': { collectedSince: '2026-11-01', captured: 40, local: 26, remote: 14, unknown: 60, places: [{ country: 'FR', city: 'Lyon', views: 12 }] },
}

beforeEach(() => { vi.restoreAllMocks() })

it('renders tiles, clients linked to the history and phase-6 widgets', async () => {
  mockApi(FULL)
  renderPage(<PlaybackPage />, { path: '/playback', url: '/playback?range=30d&compare=1' })
  expect(await screen.findByText('app.playback.reasons.VideoRangeTypeNotSupported', { selector: 'span[class*=value]' })).toBeInTheDocument()
  expect(screen.getByText(/^68\s%$/)).toBeInTheDocument()
  expect(screen.getAllByText('app.playback.points')).toHaveLength(2) // direct + transcode deltas
  // Most transcoded client tile + history links keeping the period.
  const links = screen.getAllByRole('link', { name: /Jellyfin Web/ })
  expect(links[0]).toHaveAttribute('href', '/history?client=Jellyfin+Web&range=30d&compare=1')
  expect(screen.getByText('SomethingNew')).toBeInTheDocument() // unknown reason shown raw
  expect(screen.getByText('HEVC → H264')).toBeInTheDocument()
  expect(screen.getByText('H264 → app.playback.copy')).toBeInTheDocument()
  expect(screen.getByText('Lyon, FR')).toBeInTheDocument()
  expect(screen.getByText('app.playback.bitrate.subtitle')).toBeInTheDocument()
})

it('shows the "collected from" empty state when the capture has no data yet', async () => {
  mockApi({
    ...FULL,
    '/api/v2/playback/quality': { ...QUALITY, collectedSince: null, captured: 0, reasons: [], bitrate: { samples: 0, avg: null, p95: null, buckets: [0, 0, 0, 0, 0, 0] } },
    '/api/v2/playback/network': { collectedSince: null, captured: 0, local: 0, remote: 0, unknown: 100, places: [] },
  })
  renderPage(<PlaybackPage />, { path: '/playback', url: '/playback' })
  expect(await screen.findAllByText('app.analytics.notCollected')).toHaveLength(3)
  expect(screen.getAllByText('app.analytics.collectedNext')).toHaveLength(3)
})

it('shows the collection start date when the range has no captured rows', async () => {
  mockApi({ ...FULL, '/api/v2/playback/network': { collectedSince: '2026-11-01', captured: 0, local: 0, remote: 0, unknown: 5, places: [] } })
  renderPage(<PlaybackPage />, { path: '/playback', url: '/playback?from=2026-01-01&to=2026-02-01' })
  // Network empty state + partial-period notes under reasons and bitrate.
  expect(await screen.findAllByText('app.analytics.collectedSince')).toHaveLength(3)
  expect(screen.getByText('app.analytics.notCollected')).toBeInTheDocument()
})

it('ignores clients with too few views for the most transcoded tile and builds the matrix from all codec pairs', async () => {
  mockApi(FULL)
  renderPage(<PlaybackPage />, { path: '/playback', url: '/playback' })
  expect(await screen.findByText('app.playback.tiles.clientFoot')).toBeInTheDocument()
  // Tile value is Jellyfin Web (62 %), not Rare TV (3 views at 100 %).
  expect(screen.getByText('Jellyfin Web', { selector: 'span[class*=value]' })).toBeInTheDocument()
  expect(screen.queryByText('Rare TV', { selector: 'span[class*=value]' })).not.toBeInTheDocument()
  // VP9 → H264 is not in the top conversions but is in the codec matrix.
  expect(screen.queryByText('VP9 → H264')).not.toBeInTheDocument()
  expect(screen.getAllByText('VP9').length).toBeGreaterThan(0)
})

it('labels weekly method buckets on long ranges', async () => {
  mockApi({ ...FULL, '/api/v2/playback/methods': { ...FULL['/api/v2/playback/methods'], bucket: 'week' } })
  renderPage(<PlaybackPage />, { path: '/playback', url: '/playback' })
  expect((await screen.findAllByText('app.playback.weekOf')).length).toBeGreaterThan(0)
})

it('shows the usual empty state when the capture started before the period', async () => {
  mockApi({ ...FULL, '/api/v2/playback/network': { collectedSince: '2026-11-01', captured: 0, local: 0, remote: 0, unknown: 5, places: [] } })
  renderPage(<PlaybackPage />, { path: '/playback', url: '/playback?from=2026-11-05&to=2026-11-20' })
  expect(await screen.findByText('app.analytics.noData')).toBeInTheDocument()
  expect(screen.queryByText('app.analytics.notCollected')).not.toBeInTheDocument()
})

it('shows an error state with retry', async () => {
  mockApi({ ...FULL, '/api/v2/playback/clients': Error })
  renderPage(<PlaybackPage />, { path: '/playback', url: '/playback' })
  expect((await screen.findAllByRole('button', { name: 'app.action.retry' })).length).toBeGreaterThanOrEqual(2)
})
