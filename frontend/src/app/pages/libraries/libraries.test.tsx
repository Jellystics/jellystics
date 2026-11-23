import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import LibrariesPage from './LibrariesPage'
import { mockApi, renderPage } from '../library/testing'
import { librariesFixture } from '../library/fixtures'

afterEach(() => { vi.unstubAllGlobals() })

it('lists the libraries as links keeping the period', async () => {
  const spy = mockApi({ '/api/v2/libraries': librariesFixture })
  renderPage('/libraries', '/libraries?range=7d', <LibrariesPage />)
  const link = await screen.findByRole('link', { name: /Séries/ })
  expect(link).toHaveAttribute('href', '/libraries/lib-s?range=7d')
  expect(screen.getByRole('link', { name: /Films/ })).toHaveTextContent('app.libraries.items.movie:248')
  expect(screen.getByText('app.libraries.subtitle:3')).toBeInTheDocument()
  // period sent to the API
  expect(new URL(String(spy.mock.calls[0][0]), 'http://x').searchParams.get('from')).toMatch(/^\d{4}-\d{2}-\d{2}$/)
})

it('shows an empty state without libraries', async () => {
  mockApi({ '/api/v2/libraries': [] })
  renderPage('/libraries', '/libraries', <LibrariesPage />)
  expect(await screen.findByText('app.libraries.empty')).toBeInTheDocument()
})

it('shows an error with retry', async () => {
  const spy = mockApi({ '/api/v2/libraries': { __status: 500 } })
  renderPage('/libraries', '/libraries', <LibrariesPage />)
  expect(await screen.findByRole('alert')).toHaveTextContent('app.entity.error')
  await userEvent.click(screen.getByRole('button', { name: 'app.action.retry' }))
  expect(spy.mock.calls.length).toBeGreaterThan(1)
})
