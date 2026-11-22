import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { StatTile, BarList, DataTable, SessionCard } from '@/ui'

it('StatTile delta: arrow from sign, color from goodness', () => {
  const { container } = render(<StatTile label="Transcodages" value="18" unit="%" delta={{ text: '−4 pts', direction: 'down', good: true }} comparedTo="vs préc." />)
  const d = container.querySelector('.delta')!
  expect(d).toHaveClass('good')
  expect(d.querySelector('[data-icon="down"]')).not.toBeNull()
})
it('selectable StatTile is a pressed button', async () => {
  const onSelect = vi.fn()
  render(<StatTile label="Vues" value="3 412" selected onSelect={onSelect} />)
  const b = screen.getByRole('button', { name: /Vues/ })
  expect(b).toHaveAttribute('aria-pressed', 'true')
  await userEvent.click(b); expect(onSelect).toHaveBeenCalled()
})
it('BarList rows are links with values', () => {
  render(<MemoryRouter><BarList valueLabel="Vues" rows={[{ key: '1', label: 'Station Nord', value: 412, to: '/series/1' }, { key: '2', label: 'Orbital', value: 133 }]} /></MemoryRouter>)
  expect(screen.getByRole('link', { name: /Station Nord/ })).toHaveAttribute('href', '/series/1')
  expect(screen.getByText('412')).toBeInTheDocument()
})
it('DataTable renders rows as links and paginates', async () => {
  const onPageChange = vi.fn()
  render(<MemoryRouter><DataTable columns={[{ accessorKey: 'name', header: 'Nom' }, { accessorKey: 'n', header: 'Vues', meta: { numeric: true } }]} data={[{ name: 'Léa', n: 3 }]} rowHref={() => '/users/lea'} pagination={{ page: 1, pageSize: 25, total: 60, onPageChange }} /></MemoryRouter>)
  expect(screen.getByRole('link', { name: 'Léa' })).toHaveAttribute('href', '/users/lea')
  await userEvent.click(screen.getByRole('button', { name: 'app.table.next' }))
  expect(onPageChange).toHaveBeenCalledWith(2)
})
it('DataTable sort toggles asc/desc on repeated clicks', async () => {
  const onChange = vi.fn()
  const Wrap = ({ sort }: { sort: { id: string; desc: boolean } }) => (
    <DataTable columns={[{ accessorKey: 'name', header: 'Nom' }]} data={[{ name: 'a' }]} sort={{ ...sort, onChange }} />
  )
  const { rerender } = render(<Wrap sort={{ id: 'name', desc: true }} />)
  await userEvent.click(screen.getByRole('button', { name: 'Nom' }))
  expect(onChange).toHaveBeenLastCalledWith({ id: 'name', desc: false })
  rerender(<Wrap sort={{ id: 'name', desc: false }} />)
  await userEvent.click(screen.getByRole('button', { name: 'Nom' }))
  expect(onChange).toHaveBeenLastCalledWith({ id: 'name', desc: true })
  rerender(<Wrap sort={{ id: 'name', desc: true }} />)
  await userEvent.click(screen.getByRole('button', { name: 'Nom' }))
  expect(onChange).toHaveBeenLastCalledWith({ id: 'name', desc: false })
})
it('SessionCard shows link, user, method, square art, paused badge', () => {
  const { container } = render(<MemoryRouter><SessionCard session={{ id: '1', title: 'Halo', subtitle: 'Piste 3', itemHref: '/albums/1', type: 'music', user: { id: 'u', name: 'Hugo' }, device: 'Finamp', method: 'direct', paused: true, positionSec: 60, runtimeSec: 222 }} /></MemoryRouter>)
  expect(screen.getByRole('link', { name: 'Halo' })).toHaveAttribute('href', '/albums/1')
  expect(screen.getByText('Hugo')).toBeInTheDocument()
  expect(screen.getByText('app.method.direct')).toBeInTheDocument()
  expect(screen.getByText('app.session.paused')).toBeInTheDocument()
  expect(container.querySelector('.square')).not.toBeNull()
  expect(container.querySelector('.art')!.textContent).toContain('H')
})
