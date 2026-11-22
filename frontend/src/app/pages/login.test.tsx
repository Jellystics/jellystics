import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { afterEach, expect, it, vi } from 'vitest'
import LoginPage from './LoginPage'

function setup(state: number, loginStatus = 200, next = '/x') {
  window.history.replaceState({}, '', '/login')
  const calls: { url: string; body?: unknown }[] = []
  vi.stubGlobal('fetch', vi.fn(async (url: string, init?: RequestInit) => {
    calls.push({ url, body: init?.body ? JSON.parse(String(init.body)) : undefined })
    if (url === '/auth/isConfigured') return new Response(JSON.stringify({ state, jellyfinUrl: state === 1 ? 'http://jf' : undefined }))
    if (url === '/auth/check-server') return new Response(JSON.stringify({ serverName: 'MyJelly', url: 'http://jf' }))
    if (url === '/auth/configSetup') return new Response('{}')
    if (url === '/auth/jellyfin-login') return new Response(loginStatus === 200 ? JSON.stringify({ token: 'T' }) : '{}', { status: loginStatus })
    return new Response('', { status: 404 })
  }))
  const router = createMemoryRouter([{ path: '/login', element: <LoginPage /> }, { path: '/', element: <p>home</p> }, { path: '/x', element: <p>x-page</p> }], { initialEntries: [`/login?next=${encodeURIComponent(next)}`] })
  render(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}><RouterProvider router={router} /></QueryClientProvider>)
  return { calls, router }
}
afterEach(() => { vi.unstubAllGlobals(); localStorage.clear() })

it('signs in and navigates to next', async () => {
  const { calls, router } = setup(1)
  expect(await screen.findByText('MyJelly')).toBeInTheDocument()
  await userEvent.type(screen.getByLabelText('app.login.username'), 'bob')
  await userEvent.type(screen.getByLabelText('app.login.password'), 'pw')
  await userEvent.click(screen.getByRole('button', { name: 'app.login.signIn' }))
  await waitFor(() => expect(router.state.location.pathname).toBe('/x'))
  expect(localStorage.getItem('jellystics-token')).toBe('T')
  expect(calls.find((c) => c.url === '/auth/jellyfin-login')?.body).toMatchObject({ username: 'bob', password: 'pw', deviceId: expect.any(String), deviceName: expect.any(String) })
})
it('shows invalid credentials on 401', async () => {
  setup(1, 401)
  await screen.findByText('MyJelly')
  await userEvent.type(screen.getByLabelText('app.login.username'), 'bob')
  await userEvent.type(screen.getByLabelText('app.login.password'), 'pw')
  await userEvent.click(screen.getByRole('button', { name: 'app.login.signIn' }))
  expect(await screen.findByRole('alert')).toHaveTextContent('app.login.invalidCredentials')
})
it('shows admin required on 403', async () => {
  setup(1, 403)
  await screen.findByText('MyJelly')
  await userEvent.type(screen.getByLabelText('app.login.username'), 'bob')
  await userEvent.type(screen.getByLabelText('app.login.password'), 'pw')
  await userEvent.click(screen.getByRole('button', { name: 'app.login.signIn' }))
  expect(await screen.findByRole('alert')).toHaveTextContent('app.login.adminRequired')
})
it('runs the setup wizard when unconfigured', async () => {
  const { calls } = setup(0)
  await userEvent.type(await screen.findByLabelText('app.login.jellyfinUrl'), 'http://jf')
  await userEvent.type(screen.getByLabelText('app.login.apiKey'), 'KEY')
  await userEvent.click(screen.getByRole('button', { name: 'app.login.connect' }))
  expect(await screen.findByText('MyJelly')).toBeInTheDocument()
  expect(calls.find((c) => c.url === '/auth/configSetup')?.body).toEqual({ JF_HOST: 'http://jf', JF_API_KEY: 'KEY' })
})

it.each(['//evil.com', '/\\evil.com', 'https://evil.com', '/\\\\x', '/login', '/login?next=%2Fx', '/setup'])('ignores hostile next=%s', async (next) => {
  const { router } = setup(1, 200, next)
  await screen.findByText('MyJelly')
  await userEvent.type(screen.getByLabelText('app.login.username'), 'bob')
  await userEvent.type(screen.getByLabelText('app.login.password'), 'pw')
  await userEvent.click(screen.getByRole('button', { name: 'app.login.signIn' }))
  await waitFor(() => expect(router.state.location.pathname).toBe('/'))
})
