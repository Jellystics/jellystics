import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import type { LibrariesDaily, OverviewPlays } from '@/api'
import { LibrariesScatterCard, PlaysScatterCard } from './ScatterCards'
import { librarySeries } from './librarySeries'
import { renderPage, stubFetch } from './renderPage'

afterEach(() => { vi.unstubAllGlobals() })

const plays: OverviewPlays = {
  total: 3, limit: 5000, capped: false,
  points: [
    { id: 'p3', startedAt: '2026-11-20T21:00:00+01:00', watchSeconds: 2760, type: 'series', item: { id: 'e1', kind: 'episode', name: 'Le phare', parentId: 's1', parentName: 'Station Nord' }, user: { id: 'u2', name: 'Léa' } },
    { id: 'p2', startedAt: '2026-11-19T08:00:00+01:00', watchSeconds: 200, type: 'music', item: { id: 't1', kind: 'track', name: 'Lueur', parentId: 'al1', parentName: 'Halo' }, user: { id: 'u1', name: 'Céline' } },
    { id: 'p1', startedAt: '2026-11-18T20:00:00+01:00', watchSeconds: 7080, type: 'movie', item: { id: 'm1', kind: 'movie', name: 'Les Rivages' }, user: { id: 'u1', name: 'Céline' } },
  ],
}

const card = (title: string) => screen.getByText(title).closest('section')!

it('All plays: one coloured dot per play, tooltip, sr table, click opens the entity page with the period', async () => {
  const api = stubFetch({ '/api/v2/overview/plays': plays })
  const { container, router } = renderPage(<PlaysScatterCard />, { path: '/?range=7d&compare=1' })
  await waitFor(() => expect(container.querySelectorAll('circle[data-hit]')).toHaveLength(3))
  // Period without compare.
  const p = api.calls('/api/v2/overview/plays')[0].searchParams
  expect(p.get('from')).toMatch(/^\d{4}-\d{2}-\d{2}$/)
  expect(p.has('compare')).toBe(false)

  const dots = [...container.querySelectorAll('circle[data-hit]')]
  // Oldest first: movie, music, series; colours = media type tokens.
  expect(dots.map((d) => d.getAttribute('fill'))).toEqual(['var(--t-movie)', 'var(--t-music)', 'var(--t-series)'])
  const c = card('app.overview.plays.title')
  expect(within(c).getAllByText('app.type.movie').length).toBeGreaterThan(0) // legend
  expect(within(c).queryByText('app.overview.plays.capped')).toBeNull()
  // sr-only table: one row per play.
  expect(within(c).getByRole('table').querySelectorAll('tbody tr')).toHaveLength(3)
  expect(within(c).getByRole('cell', { name: 'Station Nord › Le phare' })).toBeInTheDocument()

  fireEvent.mouseMove(dots[2], { clientX: 5, clientY: 5 })
  const tip = screen.getByRole('tooltip')
  expect(tip).toHaveTextContent('Station Nord › Le phare')
  expect(tip).toHaveTextContent('Léa')
  expect(tip).toHaveTextContent('20 nov. 2026, 21:00')
  expect(tip).toHaveTextContent('46 min')

  fireEvent.keyDown(dots[2], { key: 'Enter' })
  await waitFor(() => expect(router.state.location.pathname).toBe('/series/s1/episodes/e1'))
  expect(router.state.location.search).toBe('?range=7d&compare=1')
})

it('All plays: capped note, empty and error states', async () => {
  stubFetch({ '/api/v2/overview/plays': { ...plays, total: 6000, capped: true } })
  const first = renderPage(<PlaysScatterCard />)
  expect(await screen.findByText('app.overview.plays.capped')).toBeInTheDocument()
  first.unmount()

  stubFetch({ '/api/v2/overview/plays': { total: 0, limit: 5000, capped: false, points: [] } })
  const second = renderPage(<PlaysScatterCard />)
  expect(await screen.findByText('app.overview.plays.empty')).toBeInTheDocument()
  second.unmount()

  stubFetch({ '/api/v2/overview/plays': { status: 500 } })
  renderPage(<PlaysScatterCard />)
  expect(await screen.findByText('app.error.generic')).toBeInTheDocument()
})

type LibType = LibrariesDaily['libraries'][number]['type']
/** Libraries by views desc; names run backwards (Lib J, Lib I, …) so name order differs from views order. */
const libs = (types: LibType[]): LibrariesDaily => {
  const libraries = types.map((type, i) => ({ id: `l${i}`, name: `Lib ${String.fromCharCode(74 - i)}`, type, views: 100 - i }))
  return {
    bucket: 'day', libraries,
    points: libraries.flatMap((l) => [{ date: '2026-11-19', libraryId: l.id, views: 2 }, { date: '2026-11-20', libraryId: l.id, views: 3 }]),
  }
}
const colours = (s: { name: string; color: string }[]) => s.map((x) => [x.name, x.color])

it('librarySeries: one library per type takes the type colour, others take --cat-4..8, then "other"', () => {
  const one = librarySeries(libs(['movie', 'series', 'music']), 'Autres')
  expect(colours(one.series)).toEqual([['Lib H', 'var(--t-music)'], ['Lib I', 'var(--t-series)'], ['Lib J', 'var(--t-movie)']])
  expect(one.points).toHaveLength(6)

  // First movie library by name (Lib I) gets the movie colour; the second one and the "other" one take extra slots by name.
  const mixed = librarySeries(libs(['movie', 'movie', 'other', 'series']), 'Autres')
  expect(colours(mixed.series)).toEqual([['Lib G', 'var(--t-series)'], ['Lib H', 'var(--cat-4)'], ['Lib I', 'var(--t-movie)'], ['Lib J', 'var(--cat-5)']])

  // 9 extra libraries: the 5 most viewed keep a slot (fixed order by name, never cycled), the 4 smallest fold.
  const many = librarySeries(libs(['movie', ...Array<LibType>(9).fill('other')]), 'Autres')
  expect(colours(many.series)).toEqual([
    ['Lib E', 'var(--cat-4)'], ['Lib F', 'var(--cat-5)'], ['Lib G', 'var(--cat-6)'], ['Lib H', 'var(--cat-7)'], ['Lib I', 'var(--cat-8)'],
    ['Lib J', 'var(--t-movie)'], ['Autres', 'var(--axis)'],
  ])
  expect(many.points.filter((p) => p.seriesId === '__other__')).toEqual([{ date: '2026-11-19', seriesId: '__other__', views: 8 }, { date: '2026-11-20', seriesId: '__other__', views: 12 }])
})

it('Plays by library: one dot per (day, library), legend, weekly subtitle, empty state', async () => {
  stubFetch({ '/api/v2/overview/libraries-daily': libs(['movie', 'series']) })
  const first = renderPage(<LibrariesScatterCard />)
  await waitFor(() => expect(first.container.querySelectorAll('circle[data-hit]')).toHaveLength(4))
  const c = card('app.overview.libraries.title')
  expect(within(c).getByText('app.overview.libraries.subtitle')).toBeInTheDocument()
  expect(within(c).getAllByText('Lib J').length).toBeGreaterThan(0)
  expect(within(c).getAllByText('Lib I').length).toBeGreaterThan(0)
  const dots = [...first.container.querySelectorAll('circle[data-hit]')]
  expect(new Set(dots.map((d) => d.getAttribute('fill')))).toEqual(new Set(['var(--t-movie)', 'var(--t-series)']))
  // Same day → same x (local midnight in the server zone).
  expect(dots[0].getAttribute('cx')).toBe(dots[2].getAttribute('cx'))
  expect(dots[0].getAttribute('cx')).not.toBe(dots[1].getAttribute('cx'))
  fireEvent.mouseMove(dots[0], { clientX: 5, clientY: 5 })
  expect(screen.getByRole('tooltip')).toHaveTextContent('19 nov.')
  first.unmount()

  stubFetch({ '/api/v2/overview/libraries-daily': { ...libs(['movie', 'series']), bucket: 'week' } })
  const second = renderPage(<LibrariesScatterCard />)
  expect(await screen.findByText('app.overview.libraries.subtitleWeek')).toBeInTheDocument()
  second.unmount()

  stubFetch({ '/api/v2/overview/libraries-daily': { bucket: 'day', libraries: [], points: [] } })
  renderPage(<LibrariesScatterCard />)
  expect(await screen.findByText('app.overview.libraries.empty')).toBeInTheDocument()
})
