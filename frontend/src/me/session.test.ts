import { authHeader, basePathFrom, itemImageUrl, jellyfinHome, normalizeAddress, parseCredentials, pickServer, readSession, userImageUrl, type JellyfinServer } from './session'

describe('basePathFrom', () => {
  it.each([
    ['/Jellystics/Me/', ''],
    ['/Jellystics/Me', ''],
    ['/jellyfin/Jellystics/Me/', '/jellyfin'],
    ['/a/b/jellystics/me/index.html', '/a/b'],
    ['/me/', ''],
    ['/', ''],
    ['/x/Jellystics/Mesh/', ''],
  ])('%s → %j', (path, base) => expect(basePathFrom(path)).toBe(base))
})

describe('normalizeAddress', () => {
  it('lowercases and drops the trailing slash', () => expect(normalizeAddress('http://Host:8096/JF/')).toBe('http://host:8096/jf'))
  it('drops the default port', () => expect(normalizeAddress('https://media.example.org:443')).toBe('https://media.example.org'))
  it('rejects non URLs', () => { expect(normalizeAddress('nope')).toBeNull(); expect(normalizeAddress(null)).toBeNull() })
})

describe('parseCredentials', () => {
  it('reads Servers', () => expect(parseCredentials(JSON.stringify({ Servers: [{ Id: 'a' }] }))).toEqual([{ Id: 'a' }]))
  it.each([null, '', 'not json', '{"Servers":3}', 'null'])('%j → []', (raw) => expect(parseCredentials(raw)).toEqual([]))
})

describe('pickServer', () => {
  const a: JellyfinServer = { Id: 'a', AccessToken: 'tok-a', ManualAddress: 'http://other:8096', DateLastAccessed: 100 }
  const b: JellyfinServer = { Id: 'b', AccessToken: 'tok-b', LocalAddress: 'http://media.lan:8096/jellyfin/', DateLastAccessed: 50 }
  const c: JellyfinServer = { Id: 'c', AccessToken: 'tok-c', RemoteAddress: 'https://far.example', DateLastAccessed: 300 }
  const noToken: JellyfinServer = { Id: 'n', AccessToken: null, ManualAddress: 'http://media.lan:8096/jellyfin' }

  it('prefers the server at this origin + base path', () => {
    expect(pickServer([a, b, c], 'http://media.lan:8096', '/jellyfin')?.Id).toBe('b')
  })
  it('matches the remote address too', () => {
    expect(pickServer([a, b, c], 'https://far.example', '')?.Id).toBe('c')
  })
  it('ignores a matching server without a token', () => {
    expect(pickServer([noToken, a], 'http://media.lan:8096', '/jellyfin')?.Id).toBe('a')
  })
  it('falls back to the most recently accessed server with a token', () => {
    expect(pickServer([a, b, c], 'http://unknown', '')?.Id).toBe('c')
  })
  it('accepts DateLastAccessed as a date string', () => {
    const d: JellyfinServer = { Id: 'd', AccessToken: 't', DateLastAccessed: '2030-01-01T00:00:00Z' }
    expect(pickServer([a, d], 'http://unknown', '')?.Id).toBe('d')
  })
  it('falls back to the first server with a token when none is dated', () => {
    expect(pickServer([noToken, { Id: 'x', AccessToken: 'x' }, { Id: 'y', AccessToken: 'y' }], 'http://unknown', '')?.Id).toBe('x')
  })
  it('returns null without any token', () => expect(pickServer([noToken], 'http://media.lan:8096', '/jellyfin')).toBeNull())
})

describe('readSession', () => {
  const store = (v: string | null) => ({ getItem: (k: string) => (k === 'jellyfin_credentials' ? v : null) })
  const creds = JSON.stringify({ Servers: [{ Id: 'srv', AccessToken: 'abc', UserId: 'u1', ManualAddress: 'http://localhost:8097' }] })

  it('returns the token and user of the matching server', () => {
    expect(readSession(store(creds), { origin: 'http://localhost:8097', pathname: '/Jellystics/Me/' })).toEqual({ token: 'abc', userId: 'u1', serverId: 'srv' })
  })
  it('returns null when signed out', () => {
    expect(readSession(store(null), { origin: 'http://localhost:8097', pathname: '/Jellystics/Me/' })).toBeNull()
    expect(readSession(store(JSON.stringify({ Servers: [{ Id: 'srv', UserId: 'u1' }] })), { origin: 'http://x', pathname: '/' })).toBeNull()
  })
  it('survives a blocked storage', () => {
    expect(readSession({ getItem: () => { throw new Error('blocked') } }, { origin: 'http://x', pathname: '/' })).toBeNull()
    expect(readSession(null, { origin: 'http://x', pathname: '/' })).toBeNull()
  })
})

describe('urls', () => {
  it('builds the Jellyfin auth header', () => expect(authHeader('abc')).toBe('MediaBrowser Token="abc"'))
  it('builds same-origin image and home urls under the base path', () => {
    expect(itemImageUrl('/jf', 'm 1', 64)).toBe('/jf/Items/m%201/Images/Primary?fillWidth=64&quality=90')
    expect(userImageUrl('', 'u1', 64, 't')).toBe('/Users/u1/Images/Primary?fillWidth=64&quality=90&tag=t')
    expect(jellyfinHome('/jf')).toBe('/jf/web/')
  })
})
