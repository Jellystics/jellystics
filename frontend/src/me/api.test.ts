import { errorReason, meFetch, MeApiError } from './api'

describe('errorReason', () => {
  it.each([
    [503, { error: 'notLinked' }, 'notLinked'],
    [503, undefined, 'notLinked'],
    [502, { error: 'unreachable' }, 'unreachable'],
    [504, '', 'unreachable'],
    [500, { error: 'timeout' }, 'unreachable'],
    [401, { error: 'invalid secret' }, 'notLinked'],
    [401, '', 'signedOut'],
    [403, { error: 'unknown user' }, 'forbidden'],
    [500, { error: 'internal error' }, 'error'],
    [400, { error: 'invalid from' }, 'error'],
  ])('%i %j → %s', (status, body, reason) => expect(errorReason(status, body)).toBe(reason))
})

describe('meFetch', () => {
  const session = { token: 'tok', userId: 'u1', serverId: 's' }
  afterEach(() => vi.unstubAllGlobals())

  it('calls ./api/<path> with the Jellyfin token and the query', async () => {
    const fn = vi.fn(async () => new Response('{"ok":true}', { status: 200 }))
    vi.stubGlobal('fetch', fn)
    await expect(meFetch(session, 'history', { page: 2, desc: true, empty: '', no: false, nil: null })).resolves.toEqual({ ok: true })
    const [url, init] = fn.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toBe('./api/history?page=2&desc=1')
    expect((init.headers as Record<string, string>).Authorization).toBe('MediaBrowser Token="tok"')
    expect(url).not.toMatch(/userId/i)
  })

  it('throws a reasoned error on failure', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => new Response('{"error":"unreachable"}', { status: 502 })))
    const err = await meFetch(session, 'profile').catch((e) => e)
    expect(err).toBeInstanceOf(MeApiError)
    expect(err).toMatchObject({ status: 502, reason: 'unreachable' })
  })

  it('treats a 200 HTML page (plugin fallback) as not linked', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => new Response('<!doctype html><p>Statistiques indisponibles</p>', { status: 200 })))
    await expect(meFetch(session, 'profile')).rejects.toMatchObject({ reason: 'notLinked' })
  })
})
