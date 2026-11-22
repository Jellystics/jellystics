import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, useLocation } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { SegmentedControl, Tabs, ConfirmDialog, PeriodPicker, Toaster, toast } from '@/ui'
import { useState } from 'react'

it('SegmentedControl selects with mouse and keyboard', async () => {
  function T() { const [v, setV] = useState<'a' | 'b'>('a'); return <SegmentedControl aria-label="Métrique" value={v} onValueChange={setV} options={[{ value: 'a', label: 'Temps' }, { value: 'b', label: 'Vues' }]} /> }
  render(<T />)
  await userEvent.click(screen.getByRole('radio', { name: 'Vues' }))
  expect(screen.getByRole('radio', { name: 'Vues' })).toHaveAttribute('data-state', 'on')
})

it('Tabs expose tab roles and selection', async () => {
  const onChange = vi.fn()
  render(<Tabs value="a" onValueChange={onChange} items={[{ value: 'a', label: 'Aperçu' }, { value: 'b', label: 'Historique', count: 3 }]} />)
  expect(screen.getByRole('tab', { name: /Aperçu/ })).toHaveAttribute('aria-selected', 'true')
  await userEvent.click(screen.getByRole('tab', { name: /Historique/ }))
  expect(onChange).toHaveBeenCalledWith('b')
})

it('ConfirmDialog calls onConfirm', async () => {
  const onConfirm = vi.fn()
  render(<ConfirmDialog open onOpenChange={() => {}} title="Supprimer ?" description="Définitif" confirmLabel="Supprimer" danger onConfirm={onConfirm} />)
  await userEvent.click(screen.getByRole('button', { name: 'Supprimer' }))
  expect(onConfirm).toHaveBeenCalled()
})

it('PeriodPicker writes the range to the URL', async () => {
  const qc = new QueryClient(); qc.setQueryData(['server-config'], { configured: true, timezone: 'Europe/Paris' })
  let search = ''
  function Loc() { search = useLocation().search; return null }
  render(<QueryClientProvider client={qc}><MemoryRouter initialEntries={['/']}><PeriodPicker /><Loc /></MemoryRouter></QueryClientProvider>)
  await userEvent.click(screen.getByRole('button', { name: /app.period.30d/ }))
  await userEvent.click(screen.getByRole('menuitemradio', { name: /app.period.7d/ }))
  expect(search).toBe('?range=7d')
})

it('toast renders messages', async () => {
  render(<Toaster />)
  toast.success('Synchronisation terminée')
  expect(await screen.findByText('Synchronisation terminée')).toBeInTheDocument()
})

it('PeriodPicker custom range stores an exclusive end date', async () => {
  const qc = new QueryClient(); qc.setQueryData(['server-config'], { configured: true, timezone: 'Europe/Paris' })
  let search = ''
  function Loc() { search = useLocation().search; return null }
  render(<QueryClientProvider client={qc}><MemoryRouter initialEntries={['/']}><PeriodPicker /><Loc /></MemoryRouter></QueryClientProvider>)
  await userEvent.click(screen.getByRole('button', { name: /app.period.30d/ }))
  await userEvent.click(screen.getByRole('menuitem', { name: /app.period.custom/ }))
  const [from, to] = Array.from(document.querySelectorAll<HTMLInputElement>('input[type="date"]'))
  await userEvent.clear(from); await userEvent.type(from, '2025-01-01')
  await userEvent.clear(to); await userEvent.type(to, '2025-01-10')
  await userEvent.click(screen.getByRole('button', { name: 'app.period.apply' }))
  expect(search).toBe('?from=2025-01-01&to=2025-01-11')
})
