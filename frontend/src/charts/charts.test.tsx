import { render, screen, fireEvent } from '@testing-library/react'
import { LineChart, Columns, ChartFrame } from '@/charts'
import { toCsv } from './ChartFrame'

it('LineChart is an accessible image with one hit zone per label', () => {
  const { container } = render(<div style={{ width: 600 }}><LineChart labels={['a', 'b', 'c']} series={[{ name: 'Temps', color: 'var(--accent)', values: [1, 3, 2] }]} /></div>)
  expect(screen.getByRole('img', { name: 'Temps' })).toBeInTheDocument()
  expect(container.querySelectorAll('[data-hit]')).toHaveLength(3)
})
it('Columns emphasis labels the emphasised bar only', () => {
  render(<Columns labels={['00 h', '01 h']} values={[3, 9]} name="Vues" emphasis={1} />)
  expect(screen.getByText('9')).toBeInTheDocument()
  expect(screen.queryByText('3')).toBeNull()
})
it('tooltip appears on hover and on focus', () => {
  const { container } = render(<Columns labels={['lun.']} values={[4]} name="Vues" />)
  const hit = container.querySelector('[data-hit]')!
  fireEvent.mouseMove(hit, { clientX: 10, clientY: 10 })
  expect(screen.getByRole('tooltip')).toHaveTextContent('lun.')
  fireEvent.mouseLeave(hit)
  expect(screen.queryByRole('tooltip')).toBeNull()
  fireEvent.focus(hit)
  expect(screen.getByRole('tooltip')).toHaveTextContent('Vues')
  fireEvent.blur(hit)
  expect(screen.queryByRole('tooltip')).toBeNull()
})
it('ChartFrame always shows the chart, with its data as a screen-reader table', () => {
  render(<ChartFrame title="Vues" csvName="vues" table={{ columns: ['Jour', 'Vues'], rows: [['lun.', 4]] }}><div>the-chart</div></ChartFrame>)
  expect(screen.getByText('the-chart')).toBeInTheDocument()
  expect(screen.queryByRole('radio')).toBeNull()
  const table = screen.getByRole('table', { name: 'Vues' })
  expect(table.closest('.sr-only')).not.toBeNull()
  expect(table).toHaveTextContent('lun.')
  expect(screen.getByRole('button', { name: 'app.chart.csv' })).toBeInTheDocument()
})
it('toCsv escapes quotes, commas and newlines', () => {
  expect(toCsv({ columns: ['a', 'b'], rows: [['x"y', 'p,q'], ['l1\nl2', 3]] })).toBe('a,b\n"x""y","p,q"\n"l1\nl2",3')
})
it('toCsv quotes carriage returns', () => {
  expect(toCsv({ columns: ['a'], rows: [['x\ry']] })).toBe('a\n"x\ry"')
})
it('Columns bars activate from the keyboard and showXLabel picks the x labels', () => {
  const onBarClick = vi.fn()
  const { container } = render(<Columns labels={['A', 'B', 'C']} values={[1, 2, 3]} name="Vues" onBarClick={onBarClick} showXLabel={(i) => i === 2} />)
  const hits = container.querySelectorAll('[data-hit]')
  fireEvent.keyDown(hits[1], { key: 'Enter' })
  expect(onBarClick).toHaveBeenCalledWith(1)
  expect(container.querySelector('svg')).not.toHaveTextContent('A')
  expect(container.querySelector('svg')).toHaveTextContent('C')
})
