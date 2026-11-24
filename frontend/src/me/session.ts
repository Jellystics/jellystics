// Jellyfin session of the « Mes statistiques » page: the page is served by the Jellystics plugin under
// <basePath>/Jellystics/Me/, on the jellyfin-web origin, so it reads the token jellyfin-web keeps in
// localStorage. Everything here is pure (storage / location passed in) so it is unit tested.

export const CREDENTIALS_KEY = 'jellyfin_credentials'
const ME_SEGMENT = /\/jellystics\/me(?:\/|$)/i

/** Jellyfin base path ("" or "/jellyfin"): the part of the page path before /Jellystics/Me. */
export function basePathFrom(pathname: string): string {
  const m = ME_SEGMENT.exec(pathname)
  if (!m) return ''
  return pathname.slice(0, m.index).replace(/\/+$/, '')
}

export interface JellyfinServer {
  Id?: string
  Name?: string
  AccessToken?: string | null
  UserId?: string | null
  ManualAddress?: string | null
  LocalAddress?: string | null
  RemoteAddress?: string | null
  DateLastAccessed?: number | string | null
}

export interface Session { token: string; userId: string | null; serverId: string | null }

/** Server entries of localStorage.jellyfin_credentials; [] when absent or not JSON. */
export function parseCredentials(raw: string | null | undefined): JellyfinServer[] {
  if (!raw) return []
  try {
    const servers = (JSON.parse(raw) as { Servers?: unknown })?.Servers
    return Array.isArray(servers) ? servers.filter((s): s is JellyfinServer => !!s && typeof s === 'object') : []
  } catch {
    return []
  }
}

/** "http://Host:8096/jellyfin/" → "http://host:8096/jellyfin" (null when not a URL). */
export function normalizeAddress(addr: string | null | undefined): string | null {
  if (!addr) return null
  try {
    const u = new URL(addr)
    return `${u.protocol}//${u.host}${u.pathname.replace(/\/+$/, '')}`.toLowerCase()
  } catch {
    return null
  }
}

const accessed = (s: JellyfinServer) => {
  const v = s.DateLastAccessed
  const n = typeof v === 'number' ? v : typeof v === 'string' ? Date.parse(v) || Number(v) : NaN
  return Number.isFinite(n) ? n : 0
}

/**
 * Server entry of this page: the one whose address is this origin + base path, else the most recently used
 * one holding a token, else the first one with a token.
 */
export function pickServer(servers: JellyfinServer[], origin: string, basePath: string): JellyfinServer | null {
  const withToken = servers.filter((s) => typeof s.AccessToken === 'string' && s.AccessToken !== '')
  if (withToken.length === 0) return null
  const here = normalizeAddress(`${origin}${basePath}`)
  const match = withToken.find((s) => [s.ManualAddress, s.LocalAddress, s.RemoteAddress].some((a) => normalizeAddress(a) === here))
  if (match) return match
  const dated = withToken.filter((s) => accessed(s) > 0)
  if (dated.length) return dated.reduce((best, s) => (accessed(s) > accessed(best) ? s : best))
  return withToken[0]
}

/** Session read from the jellyfin-web credentials, or null when the user is not signed in to Jellyfin. */
export function readSession(storage: Pick<Storage, 'getItem'> | null, loc: { origin: string; pathname: string }): Session | null {
  let raw: string | null = null
  try { raw = storage?.getItem(CREDENTIALS_KEY) ?? null } catch { /* storage blocked */ }
  const server = pickServer(parseCredentials(raw), loc.origin, basePathFrom(loc.pathname))
  if (!server?.AccessToken) return null
  return { token: server.AccessToken, userId: server.UserId || null, serverId: server.Id || null }
}

/** Authorization header value understood by Jellyfin. */
export const authHeader = (token: string) => `MediaBrowser Token="${token.replace(/"/g, '')}"`

/** Jellyfin user picture (same origin, public). */
export const userImageUrl = (basePath: string, userId: string, width: number, tag?: string | null) =>
  `${basePath}/Users/${encodeURIComponent(userId)}/Images/Primary?fillWidth=${width}&quality=90${tag ? `&tag=${encodeURIComponent(tag)}` : ''}`

/** Jellyfin item poster (same origin, public). */
export const itemImageUrl = (basePath: string, itemId: string, width: number) =>
  `${basePath}/Items/${encodeURIComponent(itemId)}/Images/Primary?fillWidth=${width}&quality=90`

/** Jellyfin web home of this server. */
export const jellyfinHome = (basePath: string) => `${basePath}/web/`
