import { ApiError, handleUnauthorized } from '@/lib/api'
import { getToken } from '@/lib/auth'

/** Authenticated multipart upload (apiFetch only speaks JSON). */
export async function uploadFile<T>(path: string, file: File): Promise<T> {
  const form = new FormData()
  form.append('file', file)
  const headers: Record<string, string> = { Accept: 'application/json' }
  const token = getToken(); if (token) headers.Authorization = `Bearer ${token}`
  const res = await fetch(path, { method: 'POST', headers, body: form })
  const text = await res.text()
  let data: unknown
  try { data = text ? JSON.parse(text) : undefined } catch { data = text }
  if (res.status === 401) handleUnauthorized()
  if (!res.ok) throw new ApiError(res.status, data)
  return data as T
}

/** Authenticated download: backups contain the Jellyfin API key, so a plain link would be refused. */
export async function downloadAuthed(path: string, filename: string): Promise<void> {
  const headers: Record<string, string> = {}
  const token = getToken(); if (token) headers.Authorization = `Bearer ${token}`
  const res = await fetch(path, { headers })
  if (res.status === 401) handleUnauthorized()
  if (!res.ok) throw new ApiError(res.status, undefined)
  const url = URL.createObjectURL(await res.blob())
  const a = document.createElement('a')
  a.href = url; a.download = filename
  document.body.appendChild(a); a.click(); document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

/** Server-provided error message when there is one. */
export function errorMessage(e: unknown, fallback: string): string {
  if (e instanceof ApiError && e.body && typeof e.body === 'object' && 'error' in e.body && typeof (e.body as { error: unknown }).error === 'string') return (e.body as { error: string }).error
  return fallback
}

export const enc = encodeURIComponent

/** Array-or-empty guard: the backend (and mocks) sometimes answer null or an object. */
export function asArray<T>(v: unknown): T[] { return Array.isArray(v) ? (v as T[]) : [] }

/** Locale date-time, or the raw string when it does not parse. */
export function formatDateTime(s: string, locale: string): string {
  const d = new Date(s)
  return Number.isNaN(d.getTime()) ? s : new Intl.DateTimeFormat(locale || 'fr', { dateStyle: 'medium', timeStyle: 'short' }).format(d)
}
