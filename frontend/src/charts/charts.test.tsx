import { render, screen, fireEvent } from '@testing-library/react'
import { LineChart, Columns, ChartFrame } from '@/charts'

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
it('ChartFrame toggles to a table view', () => {
  render(<ChartFrame title="Vues" csvName="vues" table={{ columns: ['Jour', 'Vues'], rows: [['lun.', 4]] }}><div /></ChartFrame>)
  fireEvent.click(screen.getByRole('radio', { name: 'app.chart.table' }))
  expect(screen.getByRole('table')).toHaveTextContent('lun.')
})
import { toCsv } from './ChartFrame'
it('toCsv escapes quotes, commas and newlines', () => {
  expect(toCsv({ columns: ['a', 'b'], rows: [['x"y', 'p,q'], ['l1\nl2', 3]] })).toBe('a,b\n"x""y","p,q"\n"l1\nl2",3')
})
