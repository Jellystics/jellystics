// Precise data of the Jellystics plugin on the entity pages: shown with its
// precision label when present, hidden (never zeros) without plugin data.
import { screen, waitFor, within } from '@testing-library/react'
import EpisodePage from '../episode/EpisodePage'
import MoviePage from '../movie/MoviePage'
import SeriesPage from '../series/SeriesPage'
import LibraryPage from './LibraryPage'
import { calls, mockApi, renderPage } from './testing'
import {
  decadesFixture, episodeFixture, episodeMarksFixture, firstViewDelayFixture, growthFixture, libraryFixture,
  movieFixture, movieMarksFixture, seriesFixture, seriesHistoryFixture, sittingsFixture,
} from './fixtures'

afterEach(() => { vi.unstubAllGlobals() })

const NO_PLUGIN = {
  playedToEnd: { plays: 5, finished: 4, measured: 0, measuredFinished: 0, ratio: 0.8 },
  skips: { plays: 0, skipped: 0, introPlays: 0, intro: 0, skipRate: null, introRate: null, avgSkippedSeconds: null },
  pluginSince: null,
}
const NO_MARKS = { collectedSince: null, manual: 0, automatic: 0, other: 0, unplayed: 0, watched: 5, manualWithoutView: 0 }

describe('episode page', () => {
  const open = (path: string) => renderPage('/series/:id/episodes/:episodeId', path, <EpisodePage />)

  it('shows finished views, skips and played marks with their precision labels', async () => {
    const spy = mockApi({ '/api/v2/episodes/e5': episodeFixture, '/api/v2/items/e5/marks': episodeMarksFixture })
    open('/series/s1/episodes/e5?from=2026-11-01&to=2026-11-23')
    expect(await screen.findByText('app.metric.finishedViews')).toBeInTheDocument()
    expect(screen.getByText('80 %')).toBeInTheDocument()
    expect(screen.queryByText('app.plugin.playedToEnd.label')).toBeNull()
    // 3 views measured by the plugin, 2 estimated from the stop position.
    expect(screen.getByText('app.plugin.playedToEnd.footMixed')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'app.plugin.skips.title' })).toBeInTheDocument()
    expect(screen.getByText('app.plugin.skips.plays:3')).toBeInTheDocument()
    expect(screen.getAllByText('67 %')).toHaveLength(2) // intro rate and skip rate
    expect(screen.getByText('app.plugin.skips.introFoot:3')).toBeInTheDocument()
    expect(screen.getByText('1 min 14 s')).toBeInTheDocument()
    expect(screen.getByText('app.plugin.skips.estimate')).toBeInTheDocument()
    // The plugin started on 2026-11-10, inside the period.
    expect(screen.getAllByText('app.analytics.collectedSince').length).toBeGreaterThanOrEqual(1)
    expect(await screen.findByText('app.plugin.marks.manual:2')).toBeInTheDocument()
    expect(screen.getByText(/app\.plugin\.marks\.manualWithoutView:1/)).toBeInTheDocument()
    expect(screen.getByText('app.plugin.marks.automatic:3')).toBeInTheDocument()
    // Played marks are a current state: no period sent.
    const marks = calls(spy, '/api/v2/items/e5/marks')[0].searchParams
    expect(marks.get('from')).toBeNull()
  })

  it('the finished-views tile and the funnel last step show the same finished views', async () => {
    const playedToEnd = { plays: 4, finished: 3, measured: 2, measuredFinished: 1, ratio: 0.75 }
    mockApi({
      '/api/v2/episodes/e5': { ...episodeFixture, playedToEnd, funnel: { clicks: 7, views: 5, half: 4, completed: 3 } },
      '/api/v2/items/e5/marks': NO_MARKS,
    })
    open('/series/s1/episodes/e5')
    expect(await screen.findByText('75 %')).toBeInTheDocument()
    // Funnel data table: the last step is the tile's finished count, read against views.
    const funnel = screen.getByRole('table', { name: 'app.episode.funnel' })
    const last = within(funnel).getByRole('row', { name: /app\.entity\.funnel\.completed/ })
    expect(last).toHaveTextContent(String(playedToEnd.finished))
    expect(screen.getByText('app.plugin.funnel.ofViews')).toBeInTheDocument()
    // Not the average completion (86 %) under the finished label.
    expect(screen.getByText('app.metric.finishedViews').parentElement).toHaveTextContent('75 %')
  })

  it('hides every plugin widget without plugin data', async () => {
    mockApi({ '/api/v2/episodes/e5': { ...episodeFixture, ...NO_PLUGIN }, '/api/v2/items/e5/marks': NO_MARKS })
    open('/series/s1/episodes/e5')
    expect(await screen.findByRole('heading', { name: 'La Fracture' })).toBeInTheDocument()
    await waitFor(() => expect(screen.queryByText('app.plugin.marks.title')).toBeNull())
    // Finished views stay (estimated), without the plugin foot.
    expect(screen.getByText('app.metric.finishedViews')).toBeInTheDocument()
    expect(screen.queryByText('app.plugin.playedToEnd.footMixed')).toBeNull()
    expect(screen.queryByText('app.plugin.skips.title')).toBeNull()
    expect(screen.queryByText('app.analytics.collectedSince')).toBeNull()
  })

  it('says "measured" when every view comes from the plugin, no note when the plugin predates the period', async () => {
    mockApi({
      '/api/v2/episodes/e5': { ...episodeFixture, playedToEnd: { plays: 3, finished: 2, measured: 3, measuredFinished: 2, ratio: 2 / 3 } },
      '/api/v2/items/e5/marks': NO_MARKS,
    })
    open('/series/s1/episodes/e5?from=2026-11-15&to=2026-11-23')
    expect(await screen.findByText('app.plugin.playedToEnd.footMeasured:3')).toBeInTheDocument()
    expect(screen.queryByText('app.analytics.collectedSince')).toBeNull()
  })
})

it('movie page: one finished-views tile fed by the plugin, and played marks', async () => {
  mockApi({ '/api/v2/movies/m1': movieFixture, '/api/v2/movies/m1/sittings': sittingsFixture, '/api/v2/items/m1/marks': movieMarksFixture })
  renderPage('/movies/:id', '/movies/m1', <MoviePage />)
  expect(await screen.findByText('63 %')).toBeInTheDocument()
  expect(screen.getAllByText('app.metric.finishedViews')).toHaveLength(1)
  expect(screen.getByText('app.plugin.playedToEnd.footMixed')).toBeInTheDocument()
  expect(await screen.findByText('app.plugin.marks.manual:3')).toBeInTheDocument()
  expect(screen.getByText('app.plugin.marks.other:1')).toBeInTheDocument()
  expect(screen.getByText('app.plugin.marks.unplayed:1')).toBeInTheDocument()
  expect(screen.getByText('app.plugin.marks.watched:5')).toBeInTheDocument()
})

describe('series page', () => {
  const open = () => renderPage('/series/:id', '/series/s1?season=2', <SeriesPage />)

  it('adds the intro-skip column and the skips card when measured', async () => {
    mockApi({ '/api/v2/series/s1': seriesFixture, '/api/v2/history': seriesHistoryFixture })
    open()
    expect(await screen.findByRole('heading', { name: 'app.plugin.skips.title' })).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: 'app.plugin.skips.intro' })).toBeInTheDocument()
    // Per-episode details in a focusable kit tooltip trigger (no title=).
    const meter = screen.getAllByRole('progressbar', { name: 'app.plugin.skips.intro' })[0]
    expect(meter.closest('[tabindex="0"]')).not.toBeNull()
    expect(meter.closest('[title]')).toBeNull()
  })

  it('hides them without plugin data', async () => {
    const seasons = seriesFixture.seasons.map((s) => ({ ...s, episodes: s.episodes.map((e) => ({ ...e, skipPlays: 0, introRate: null, skipRate: null, avgSkippedSeconds: null })) }))
    mockApi({ '/api/v2/series/s1': { ...seriesFixture, seasons, skips: NO_PLUGIN.skips, pluginSince: null }, '/api/v2/history': seriesHistoryFixture })
    open()
    expect(await screen.findByRole('heading', { name: 'Station Nord' })).toBeInTheDocument()
    expect(screen.queryByRole('columnheader', { name: 'app.plugin.skips.intro' })).toBeNull()
    expect(screen.queryByText('app.plugin.skips.title')).toBeNull()
  })
})

describe('library stats', () => {
  const api = (extra: Record<string, unknown>) => mockApi({
    '/api/v2/libraries/lib-s': libraryFixture,
    '/api/v2/libraries/lib-s/decades': decadesFixture,
    '/api/v2/libraries/lib-s/growth': growthFixture,
    '/api/v2/libraries/lib-s/first-view-delay': firstViewDelayFixture,
    ...extra,
  })
  const open = (q = '') => renderPage('/libraries/:id', `/libraries/lib-s?tab=stats${q}`, <LibraryPage />)

  it('labels real additions and shows the addition → first view delay', async () => {
    api({})
    open('&from=2026-10-26&to=2026-11-23')
    expect(await screen.findByText('app.plugin.growth.partial')).toBeInTheDocument()
    expect(screen.getAllByText('app.plugin.growth.removed').length).toBeGreaterThan(0)
    // Estimated buckets dashed, a marker where real additions start.
    expect(document.querySelector('[data-precise-marker]')).not.toBeNull()
    expect(document.querySelectorAll('path[data-estimated]').length).toBeGreaterThan(0)
    expect(await screen.findByRole('heading', { name: 'app.plugin.firstView.title' })).toBeInTheDocument()
    expect(screen.getByText(/app\.plugin\.firstView\.summary/)).toBeInTheDocument()
    expect(screen.getByText(/app\.plugin\.firstView\.notViewed:5/)).toBeInTheDocument()
  })

  it('falls back to the catalog approximation and hides the delay without events', async () => {
    api({
      '/api/v2/libraries/lib-s/growth': { ...growthFixture, source: 'catalog', preciseFrom: null, points: growthFixture.points.map((p) => ({ ...p, removed: 0, precise: false })) },
      '/api/v2/libraries/lib-s/first-view-delay': { ...firstViewDelayFixture, collectedSince: null, added: 0, viewed: 0, notViewed: 0, medianSeconds: null },
    })
    open()
    expect(await screen.findByText('app.plugin.growth.catalog')).toBeInTheDocument()
    expect(screen.queryByText('app.plugin.growth.removed')).toBeNull()
    expect(document.querySelector('[data-precise-marker]')).toBeNull()
    expect(document.querySelectorAll('path[data-estimated]').length).toBeGreaterThan(0)
    expect(screen.queryByText('app.plugin.firstView.title')).toBeNull()
  })
})
