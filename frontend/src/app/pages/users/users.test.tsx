import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import UsersPage from './UsersPage'
import { retentionFixture, shareFixture, usersFixture } from '../overview/fixtures'
import { renderPage, stubFetch } from '../overview/renderPage'

afterEach(() => { vi.unstubAllGlobals() })

const base = () => ({ '/api/v2/users': usersFixture(), '/api/v2/users/retention': retentionFixture(), '/api/v2/users/share': shareFixture() })

it('lists users with time, views and rows linking to the user page with the period', async () => {
  const api = stubFetch(base())
  renderPage(<UsersPage />, { path: '/users?range=90d', route: '/users' })
  const table = (await screen.findByText('Inès')).closest('table')!
  const links = within(table).getAllByRole('link')
  expect(links.map((a) => a.getAttribute('href'))).toEqual(['/users/u1?range=90d', '/users/u2?range=90d', '/users/u3?range=90d'])
  expect(within(table).getByText('980')).toBeInTheDocument()
  expect(within(table).getByText('app.users.never')).toBeInTheDocument()
  expect(screen.getByText('app.users.subtitle:3')).toBeInTheDocument()
  expect(api.calls('/api/v2/users')[0].searchParams.get('from')).toMatch(/^\d{4}-\d{2}-\d{2}$/)
})

it('sorts client-side through ?sort', async () => {
  stubFetch(base())
  const { search } = renderPage(<UsersPage />, { path: '/users', route: '/users' })
  await screen.findByText('Inès')
  await userEvent.click(screen.getByRole('button', { name: 'app.users.col.user' }))
  await waitFor(() => expect(search().get('sort')).toBe('name'))
  const names = within(screen.getByText('Inès').closest('table')!).getAllByRole('link').map((a) => a.getAttribute('href'))
  expect(names).toEqual(['/users/u1', '/users/u3', '/users/u2'])
  await userEvent.click(screen.getByRole('button', { name: 'app.users.col.user' }))
  await waitFor(() => expect(search().get('desc')).toBe(null))
  expect(within(screen.getByText('Inès').closest('table')!).getAllByRole('link')[0]).toHaveAttribute('href', '/users/u2')
})

it('renders the retention matrix and each user’s share', async () => {
  stubFetch(base())
  renderPage(<UsersPage />, { path: '/users', route: '/users' })
  expect(await screen.findByText('app.users.retention.title')).toBeInTheDocument()
  expect(screen.getByText('app.users.share.title')).toBeInTheDocument()
  // Visible in the matrix (not only in the chart's screen-reader table).
  expect(screen.getAllByText('62 %').some((el) => !el.closest('.sr-only'))).toBe(true)
  expect(screen.getAllByRole('link', { name: 'Léa' }).some((a) => a.getAttribute('href') === '/users/u2')).toBe(true)
})

it('shows empty and error states', async () => {
  stubFetch({ '/api/v2/users': [], '/api/v2/users/retention': { offsets: 5, rows: [] }, '/api/v2/users/share': { bucket: 'week', periods: [], users: [] } })
  const first = renderPage(<UsersPage />, { path: '/users', route: '/users' })
  expect(await screen.findByText('app.users.empty')).toBeInTheDocument()
  expect(await screen.findByText('app.users.retention.empty')).toBeInTheDocument()
  expect(screen.getByText('app.users.share.empty')).toBeInTheDocument()
  first.unmount()
  stubFetch({ '/api/v2/users': { status: 500 }, '/api/v2/users/retention': { status: 500 }, '/api/v2/users/share': { status: 500 } })
  renderPage(<UsersPage />, { path: '/users', route: '/users' })
  expect(await screen.findByText('app.users.error')).toBeInTheDocument()
  expect(await screen.findByText('app.users.retention.error')).toBeInTheDocument()
})
