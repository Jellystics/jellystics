import { render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RequireAuth } from './RequireAuth'
import { setSession } from '@/lib/auth'

function Where() { const l = useLocation(); return <p>login at {l.pathname}{l.search}</p> }

function setup(config: { configured: boolean } | undefined, path = '/users?range=7d') {
  const qc = new QueryClient()
  if (config) qc.setQueryData(['server-config'], { ...config, timezone: 'UTC' })
  render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/login" element={<Where />} />
          <Route path="/users" element={<RequireAuth><p>secret</p></RequireAuth>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => { localStorage.clear() })

it('redirects to /login without a token, remembering the target', () => {
  setup({ configured: true })
  expect(screen.getByText('login at /login?next=%2Fusers%3Frange%3D7d')).toBeInTheDocument()
})

it('renders children for a configured server with a session', () => {
  setSession('tok', 'lea')
  setup({ configured: true })
  expect(screen.getByText('secret')).toBeInTheDocument()
})

it('goes to /login (no next) when the server is not configured, even with a stale token', () => {
  setSession('tok', 'lea')
  setup({ configured: false })
  expect(screen.getByText('login at /login')).toBeInTheDocument()
})
