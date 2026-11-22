export { getDeviceId, getDeviceName } from './device'
const TOKEN = 'jellystics-token', USER = 'jellystics-username'

export function getToken(): string | null { try { return localStorage.getItem(TOKEN) } catch { return null } }
export function getUsername(): string { try { return localStorage.getItem(USER) ?? '' } catch { return '' } }
export function setSession(token: string, username: string) {
  try { localStorage.setItem(TOKEN, token); localStorage.setItem(USER, username) } catch { /* storage blocked: session lasts until reload */ }
}
export function clearSession() { try { localStorage.removeItem(TOKEN); localStorage.removeItem(USER) } catch { /* ignore */ } }
export function getUserId(): string {
  const t = getToken(); if (!t) return ''
  try { return JSON.parse(atob(t.split('.')[1].replace(/-/g, '+').replace(/_/g, '/'))).userId ?? '' } catch { return '' }
}
export async function logout() {
  const t = getToken()
  if (t) await fetch('/auth/logout', { method: 'POST', headers: { Authorization: `Bearer ${t}` } }).catch(() => undefined)
  clearSession()
  window.location.assign('/login')
}
