import { render, screen, act, fireEvent } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { Chip, ConfirmDialog, DataTable, Field, Input, SegBar, Toaster, toast } from '@/ui'

it('Chip remove button names the chip', () => {
  render(<Chip onRemove={() => {}}>Jazz</Chip>)
  expect(screen.getByRole('button', { name: 'app.action.removeNamed' })).toBeInTheDocument()
})

it('SegBar parts are list items', () => {
  render(<SegBar parts={[{ label: 'Direct', value: 3, color: 'red' }, { label: 'Transcode', value: 1, color: 'blue' }]} thin />)
  expect(screen.getAllByRole('listitem')).toHaveLength(2)
})

it('Field links its hint and error to the control', () => {
  const { rerender } = render(<Field label="URL" htmlFor="u" hint="https://…"><Input id="u" /></Field>)
  const input = screen.getByLabelText('URL')
  expect(input.getAttribute('aria-describedby')).toBe('u-note')
  expect(document.getElementById('u-note')).toHaveTextContent('https://…')
  rerender(<Field label="URL" htmlFor="u" error="Invalid"><Input id="u" /></Field>)
  expect(screen.getByLabelText('URL')).toHaveAttribute('aria-invalid', 'true')
})

it('ConfirmDialog stays open when onConfirm rejects', async () => {
  const onOpenChange = vi.fn()
  render(<ConfirmDialog open onOpenChange={onOpenChange} title="t" description="d" confirmLabel="Go" onConfirm={() => Promise.reject(new Error('boom'))} />)
  await userEvent.click(screen.getByRole('button', { name: 'Go' }))
  expect(onOpenChange).not.toHaveBeenCalled()
  expect(screen.getByRole('button', { name: 'Go' })).toBeEnabled()
})

describe('toast', () => {
  beforeEach(() => { vi.useFakeTimers() })
  afterEach(() => { act(() => { vi.runAllTimers() }); vi.useRealTimers() })

  it('auto-dismisses, pauses on focus and announces errors', () => {
    render(<Toaster />)
    act(() => { toast.error('Échec') })
    expect(screen.getByRole('alert')).toHaveTextContent('Échec')
    fireEvent.focus(screen.getByRole('alert'))
    act(() => { vi.advanceTimersByTime(10_000) })
    expect(screen.getByText('Échec')).toBeInTheDocument()
    fireEvent.blur(screen.getByRole('alert'))
    act(() => { vi.advanceTimersByTime(5_100) })
    expect(screen.queryByText('Échec')).toBeNull()
  })

  it('keeps at most five toasts', () => {
    render(<Toaster />)
    act(() => { for (let i = 0; i < 8; i++) toast.info(`m${i}`) })
    expect(screen.queryByText('m2')).toBeNull()
    expect(screen.getByText('m3')).toBeInTheDocument()
    expect(screen.getByText('m7')).toBeInTheDocument()
  })
})

describe('DataTable states', () => {
  const cols = [{ accessorKey: 'name', header: 'Nom' }]
  it('shows skeleton rows while loading and no empty state', () => {
    const { container } = render(<DataTable columns={cols} data={[]} loading />)
    expect(container.querySelectorAll('tbody tr')).toHaveLength(5)
    expect(screen.queryByText('app.table.empty')).toBeNull()
  })
  it('shows the empty state', () => {
    render(<DataTable columns={cols} data={[]} />)
    expect(screen.getByText('app.table.empty')).toBeInTheDocument()
  })
  it('bounds the pager on first and last page', () => {
    const { rerender } = render(<MemoryRouter><DataTable columns={cols} data={[{ name: 'a' }]} pagination={{ page: 1, pageSize: 10, total: 25, onPageChange: () => {} }} /></MemoryRouter>)
    expect(screen.getByRole('button', { name: 'app.table.prev' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'app.table.next' })).toBeEnabled()
    rerender(<MemoryRouter><DataTable columns={cols} data={[{ name: 'a' }]} pagination={{ page: 3, pageSize: 10, total: 25, onPageChange: () => {} }} /></MemoryRouter>)
    expect(screen.getByRole('button', { name: 'app.table.next' })).toBeDisabled()
  })
})
