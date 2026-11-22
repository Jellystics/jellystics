import { apiFetch, ApiError } from './api'
import { setSession, getToken } from './auth'

const originalLocation = Object.getOwnPropertyDescriptor(window, 'location')!

beforeEach(() => { localStorage.clear(); vi.restoreAllMocks() })
afterEach(() => { Object.defineProperty(window, 'location', originalLocation) })

it('sends the bearer token and query params, skipping undefined', async () => {
  setSession('tok', 'lea')
  const spy = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({ ok: 1 }), { status: 200, headers: { 'Content-Type': 'application/json' } }))
  const res = await apiFetch<{ ok: number }>('/api/v2/overview', { params: { from: '2026-11-01', userId: undefined, compare: true } })
  expect(res).toEqual({ ok: 1 })
  const [url, init] = spy.mock.calls[0]
  expect(String(url)).toBe('/api/v2/overview?from=2026-11-01&compare=true')
  expect((init!.headers as Record<string, string>).Authorization).toBe('Bearer tok')
})

it('throws ApiError with status', async () => {
  vi.spyOn(globalThis, 'fetch').mockImplementation(async () => new Response('{"error":"x"}', { status: 500 }))
  await expect(apiFetch('/x')).rejects.toMatchObject({ status: 500 })
  await expect(apiFetch('/x')).rejects.toBeInstanceOf(ApiError)
})

it('clears the session on 401', async () => {
  setSession('tok', 'lea')
  const assign = vi.fn()
  Object.defineProperty(window, 'location', { value: { ...window.location, pathname: '/users', search: '?range=7d', assign }, writable: true, configurable: true })
  vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response('', { status: 401 }))
  await expect(apiFetch('/x')).rejects.toMatchObject({ status: 401 })
  expect(getToken()).toBeNull()
  expect(assign).toHaveBeenCalledWith('/login?next=%2Fusers%3Frange%3D7d')
})

it('rejects a 2xx response whose body is not JSON', async () => {
  vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response('<html>oops</html>', { status: 200 }))
  await expect(apiFetch('/x')).rejects.toBeInstanceOf(ApiError)
})

it('setSession survives blocked storage', () => {
  vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => { throw new Error('blocked') })
  expect(() => setSession('tok', 'lea')).not.toThrow()
})

it('does not redirect when already on /login', async () => {
  const assign = vi.fn()
  Object.defineProperty(window, 'location', { value: { ...window.location, pathname: '/login', search: '', assign }, writable: true, configurable: true })
  vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response('', { status: 401 }))
  await expect(apiFetch('/x')).rejects.toMatchObject({ status: 401 })
  expect(assign).not.toHaveBeenCalled()
})
