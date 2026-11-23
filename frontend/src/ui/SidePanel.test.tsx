import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { SidePanel } from './SidePanel'

it('SidePanel is a labelled dialog that closes with the button and Escape', async () => {
  const onOpenChange = vi.fn()
  render(<SidePanel open onOpenChange={onOpenChange} title="Basse lumière" subtitle="Lune Pâle · Halo"><p>contenu</p></SidePanel>)
  const dialog = screen.getByRole('dialog', { name: 'Basse lumière' })
  expect(dialog).toHaveAccessibleDescription('Lune Pâle · Halo')
  expect(screen.getByText('contenu')).toBeInTheDocument()
  await userEvent.click(screen.getByRole('button', { name: 'app.sidePanel.close' }))
  expect(onOpenChange).toHaveBeenCalledWith(false)
  onOpenChange.mockClear()
  await userEvent.keyboard('{Escape}')
  expect(onOpenChange).toHaveBeenCalledWith(false)
})

it('SidePanel renders nothing when closed and unmounts on close', async () => {
  function T() {
    const [open, setOpen] = useState(true)
    return <SidePanel open={open} onOpenChange={setOpen} title="Piste" />
  }
  render(<T />)
  expect(screen.getByRole('dialog')).toBeInTheDocument()
  await userEvent.keyboard('{Escape}')
  expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
})
