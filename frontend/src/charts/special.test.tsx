import { act, render, screen, fireEvent } from '@testing-library/react'
import { WeekHeatmap, CalendarHeatmap, Funnel, RetentionChart, StopHistogram, Matrix, Scatter, StackedArea, SessionTimeline, SmallMultiples } from '@/charts'

it('WeekHeatmap: 7×24 focusable cells with tooltip', () => {
  const m = Array.from({ length: 7 }, () => Array.from({ length: 24 }, (_, h) => h))
  const { container } = render(<WeekHeatmap matrix={m} unit="vues" />)
  expect(screen.getByRole('img', { name: 'app.chart.weekHeatmap' })).toBeInTheDocument()
  const cells = container.querySelectorAll('[data-hit]')
  expect(cells).toHaveLength(168)
  fireEvent.mouseMove(cells[23], { clientX: 1, clientY: 1 })
  expect(screen.getByRole('tooltip')).toHaveTextContent('23 h')
})

it('CalendarHeatmap: one cell per day, Monday-first columns', () => {
  const days = Array.from({ length: 10 }, (_, i) => ({ date: `2024-03-${String(i + 1).padStart(2, '0')}`, seconds: i * 600 }))
  const { container } = render(<CalendarHeatmap days={days} />)
  expect(screen.getByRole('img', { name: 'app.chart.calendar' })).toBeInTheDocument()
  const cells = container.querySelectorAll('[data-hit]')
  expect(cells).toHaveLength(10)
  // 2024-03-01 is a Friday (row 4); 2024-03-04 is a Monday (row 0) in the next column; 03-10 is a Sunday (row 6)
  const y = (i: number) => +cells[i].getAttribute('y')!, x = (i: number) => +cells[i].getAttribute('x')!
  expect(y(0)).toBeGreaterThan(y(3))
  expect(y(9)).toBeGreaterThan(y(0))
  expect(x(3)).toBeGreaterThan(x(0))
  expect(x(9)).toBe(x(3))
})

it('Funnel shows the share vs the first stage', () => {
  const { container } = render(<Funnel stages={[{ label: 'Clics', value: 60 }, { label: 'Vues', value: 48 }]} />)
  expect(screen.getByRole('img', { name: 'app.chart.funnel' })).toBeInTheDocument()
  expect(container.querySelectorAll('path[data-hit]')).toHaveLength(2)
  expect(screen.getByText('80 %')).toBeInTheDocument()
  expect(screen.queryByText('100 %')).toBeNull()
})

it('RetentionChart: one hit zone per episode, retention label, click', () => {
  const onClick = vi.fn()
  const { container } = render(<RetentionChart seasons={[[100, 80, 60], [50, 40]]} onEpisodeClick={onClick} />)
  expect(screen.getByRole('img', { name: 'app.chart.retention' })).toBeInTheDocument()
  const hits = container.querySelectorAll('[data-hit]')
  expect(hits).toHaveLength(5)
  expect(container.querySelector('svg')).toHaveTextContent('40 %')
  fireEvent.click(hits[3])
  expect(onClick).toHaveBeenCalledWith(2, 1)
  expect(hits[3]).toHaveAttribute('aria-label', 'app.chart.episodeShort')
})

it('RetentionChart hit zones are keyboard operable (Enter and Space)', () => {
  const onClick = vi.fn()
  const { container } = render(<RetentionChart seasons={[[100, 80], [50]]} onEpisodeClick={onClick} />)
  const hits = container.querySelectorAll('[data-hit]')
  expect(hits[2]).toHaveAttribute('role', 'button')
  fireEvent.keyDown(hits[2], { key: 'Enter' })
  expect(onClick).toHaveBeenLastCalledWith(2, 1)
  fireEvent.keyDown(hits[1], { key: ' ' })
  expect(onClick).toHaveBeenLastCalledWith(1, 2)
  fireEvent.keyDown(hits[0], { key: 'a' })
  expect(onClick).toHaveBeenCalledTimes(2)
})

it('StopHistogram renders one bar zone per bin', () => {
  const { container } = render(<StopHistogram bins={[1, 2, 3, 9]} runtimeMinutes={100} />)
  expect(screen.getByRole('img', { name: 'app.chart.stops' })).toBeInTheDocument()
  expect(container.querySelectorAll('[data-hit]')).toHaveLength(4)
  expect(screen.getByText('75 min')).toBeInTheDocument()
})

it('Matrix leaves null cells empty', () => {
  const { container } = render(<Matrix rows={['HEVC']} cols={['H.264', 'HEVC']} values={[[41, null]]} unit="Lectures" />)
  expect(screen.getByRole('img', { name: 'app.chart.matrix' })).toBeInTheDocument()
  expect(container.querySelectorAll('text.mx-t')).toHaveLength(1)
  expect(container.querySelectorAll('[data-hit]')).toHaveLength(1)
  // drawn at natural size (no shrinking text) inside a horizontal scroller
  expect(container.querySelector('svg')!.getAttribute('width')).not.toBe('100%')
  expect((container.firstElementChild as HTMLElement).style.overflowX).toBe('auto')
})

it('Scatter: one point per entry with tooltip', () => {
  const { container } = render(<Scatter points={[{ x: 1, y: 2, label: 'Alice' }, { x: 3, y: 4, label: 'Bob', color: 'red' }]} xLabel="Sessions" yLabel="Heures" note="Fans" />)
  expect(screen.getByRole('img', { name: 'Heures / Sessions' })).toBeInTheDocument()
  const pts = container.querySelectorAll('circle[data-hit]')
  expect(pts).toHaveLength(2)
  expect(screen.getByText('Fans')).toBeInTheDocument()
  fireEvent.mouseMove(pts[1], { clientX: 5, clientY: 5 })
  expect(screen.getByRole('tooltip')).toHaveTextContent('Bob')
})

it('Scatter time axis: min..max domain, date ticks in the server zone, coloured series', () => {
  const t0 = Date.parse('2026-03-01T10:00:00Z'), t1 = Date.parse('2026-03-08T10:00:00Z')
  const { container } = render(
    <Scatter
      timeAxis timeZone="Europe/Paris" radius={4}
      points={[
        { x: t1, y: 30, label: 'Station Nord › Le phare', color: 'var(--t-series)', rows: [{ label: 'Utilisateur', value: 'Léa' }] },
        { x: t0, y: 120, label: 'Les Rivages', color: 'var(--t-movie)' },
      ]}
      xLabel="Début" yLabel="Minutes"
    />,
  )
  const pts = container.querySelectorAll('circle[data-hit]')
  expect(pts).toHaveLength(2)
  // Domain is min..max (not 0..max): the oldest point sits on the left edge, the newest on the right one.
  expect(Number(pts[1].getAttribute('cx'))).toBe(46)
  expect(Number(pts[0].getAttribute('cx'))).toBe(600 - 12 - 4) // right margin + radius
  expect(pts[0].getAttribute('fill')).toBe('var(--t-series)')
  expect(pts[0].getAttribute('r')).toBe('4')
  // Daily ticks on Paris midnights, labelled as dates.
  const labels = [...container.querySelectorAll('text')].map((n) => n.textContent)
  expect(labels).toContain('2 mars')
  expect(labels).toContain('8 mars')
  fireEvent.mouseMove(pts[0], { clientX: 5, clientY: 5 })
  expect(screen.getByRole('tooltip')).toHaveTextContent('Station Nord › Le phare')
  expect(screen.getByRole('tooltip')).toHaveTextContent('Léa')
  fireEvent.mouseLeave(pts[0])
  // Default tooltip formats x as a date-time in the zone.
  fireEvent.mouseMove(pts[1], { clientX: 5, clientY: 5 })
  expect(screen.getByRole('tooltip')).toHaveTextContent('1 mars 2026, 11:00')
})

it('Scatter: clickable points, one tab stop, arrows move in x order', () => {
  const onPointClick = vi.fn()
  const { container } = render(
    <Scatter
      timeAxis timeZone="UTC" onPointClick={onPointClick}
      points={[{ x: 3000, y: 1, label: 'c' }, { x: 1000, y: 1, label: 'a' }, { x: 2000, y: 1, label: 'b' }]}
      xLabel="x" yLabel="y"
    />,
  )
  const pts = [...container.querySelectorAll<SVGElement>('circle[data-hit]')]
  expect(pts.map((p) => p.getAttribute('tabindex'))).toEqual(['-1', '0', '-1'])
  expect(pts[1]).toHaveAttribute('role', 'button')
  act(() => pts[1].focus())
  fireEvent.keyDown(pts[1], { key: 'ArrowRight' })
  expect(document.activeElement).toBe(pts[2])
  fireEvent.keyDown(pts[2], { key: 'End' })
  expect(document.activeElement).toBe(pts[0])
  fireEvent.keyDown(pts[0], { key: 'Enter' })
  expect(onPointClick).toHaveBeenCalledWith(0)
  fireEvent.click(pts[2])
  expect(onPointClick).toHaveBeenLastCalledWith(2)
})

it('StackedArea: one band per series, one hit zone per label', () => {
  const { container } = render(<StackedArea labels={['a', 'b', 'c']} series={[{ name: 'Films', color: 'red', values: [1, 2, 3] }, { name: 'Séries', color: 'blue', values: [3, 2, 1] }]} />)
  expect(screen.getByRole('img', { name: 'Films, Séries' })).toBeInTheDocument()
  expect(container.querySelectorAll('[data-hit]')).toHaveLength(3)
  expect(container.querySelectorAll('path')).toHaveLength(2)
  fireEvent.mouseMove(container.querySelectorAll('[data-hit]')[0], { clientX: 1, clientY: 1 })
  expect(screen.getByRole('tooltip')).toHaveTextContent('Séries')
})

it('SessionTimeline draws one bar per span', () => {
  const { container } = render(<SessionTimeline fromHour={18} toHour={24} rows={[{ user: 'Léa', spans: [{ start: 20, end: 21, type: 'series', label: 'S2É4' }, { start: 21, end: 22, type: 'series', label: 'S2É5' }] }]} />)
  expect(screen.getByRole('img', { name: 'app.chart.timeline' })).toBeInTheDocument()
  expect(container.querySelectorAll('[data-hit]')).toHaveLength(2)
  expect(screen.getByText('Léa')).toBeInTheDocument()
})

it('SmallMultiples: one chart per item on a shared scale', () => {
  const small = Array.from({ length: 24 }, () => 2), big = Array.from({ length: 24 }, () => 10)
  render(<SmallMultiples items={[{ name: 'Léa', color: 'red', values: small }, { name: 'Tom', color: 'blue', values: big }]} />)
  const charts = screen.getAllByRole('img')
  expect(charts).toHaveLength(2)
  // top of a bar = the V coordinate of its path (smaller y = taller)
  const topOf = (svg: Element) => Math.min(...[...svg.querySelectorAll('path')].map((p) => +/V([\d.]+)/.exec(p.getAttribute('d')!)![1]))
  expect(topOf(charts[0])).toBeGreaterThan(topOf(charts[1]))
})
