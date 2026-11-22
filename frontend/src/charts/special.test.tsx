import { render, screen, fireEvent } from '@testing-library/react'
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
