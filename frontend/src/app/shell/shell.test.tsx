import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AppShell } from './AppShell'
import { AppLink } from '../AppLink'

function renderAt(path: string) {
  const qc = new QueryClient(); qc.setQueryData(['server-config'], { configured: true, timezone: 'Europe/Paris' })
  const router = createMemoryRouter([{ element: <AppShell />, children: [
    { path: '/', handle: { nav: 'overview' }, element: <AppLink to="/users">Utilisateurs</AppLink> },
    { path: '/users', handle: { nav: 'users', crumb: () => ({ label: 'app.nav.users' }) }, element: <p>users</p> },
  ] }], { initialEntries: [path] })
  render(<QueryClientProvider client={qc}><RouterProvider router={router} /></QueryClientProvider>)
  return router
}

it('marks the active nav entry with aria-current', () => {
  renderAt('/users')
  expect(screen.getByRole('link', { name: 'app.nav.users' })).toHaveAttribute('aria-current', 'page')
})
it('AppLink keeps the period', () => {
  renderAt('/?range=7d')
  expect(screen.getByRole('link', { name: 'Utilisateurs' })).toHaveAttribute('href', '/users?range=7d')
})
it('sidebar nav links keep the period too', () => {
  renderAt('/?range=90d')
  expect(screen.getByRole('link', { name: 'app.nav.history' })).toHaveAttribute('href', '/history?range=90d')
})
it('opens the command palette with Ctrl+K', async () => {
  renderAt('/')
  await userEvent.keyboard('{Control>}k{/Control}')
  expect(screen.getByRole('dialog', { name: 'app.command.title' })).toBeInTheDocument()
})
it('renders breadcrumbs from route handles', () => {
  renderAt('/users')
  expect(screen.getByText('app.nav.users', { selector: '[aria-current="page"]:not(a)' })).toBeInTheDocument()
})
it('resets the palette query when reopened', async () => {
  renderAt('/')
  await userEvent.keyboard('{Control>}k{/Control}')
  await userEvent.type(screen.getByRole('searchbox'), 'xyz')
  await userEvent.keyboard('{Control>}k{/Control}')
  await userEvent.keyboard('{Control>}k{/Control}')
  expect(screen.getByRole('searchbox')).toHaveValue('')
})
