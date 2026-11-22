import { clearSession, getToken } from './auth'

export class ApiError extends Error {
  constructor(public status: number, public body: unknown) { super(`HTTP ${status}`) }
}
/** Drop the session and bounce to the login page (once). */
export function handleUnauthorized() {
  clearSession()
  if (window.location.pathname !== '/login') window.location.assign(`/login?next=${encodeURIComponent(window.location.pathname + window.location.search)}`)
}
type Params = Record<string, string | number | boolean | undefined | null>

export async function apiFetch<T>(path: string, opts: { method?: string; params?: Params; body?: unknown; signal?: AbortSignal } = {}): Promise<T> {
  const qs = new URLSearchParams()
  Object.entries(opts.params ?? {}).forEach(([k, v]) => { if (v !== undefined && v !== null && v !== '') qs.set(k, String(v)) })
  const url = qs.toString() ? `${path}?${qs}` : path
  const headers: Record<string, string> = { Accept: 'application/json' }
  const token = getToken(); if (token) headers.Authorization = `Bearer ${token}`
  let body: BodyInit | undefined
  if (opts.body !== undefined) { headers['Content-Type'] = 'application/json'; body = JSON.stringify(opts.body) }
  const res = await fetch(url, { method: opts.method ?? (body ? 'POST' : 'GET'), headers, body, signal: opts.signal })
  const text = await res.text()
  let data: unknown = undefined
  try { data = text ? JSON.parse(text) : undefined } catch { data = text }
  if (res.status === 401) handleUnauthorized()
  if (!res.ok) throw new ApiError(res.status, data)
  return data as T
}
