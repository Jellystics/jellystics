import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient } from '@tanstack/react-query'
import { MeApp } from './MeApp'
import { meResponse, ME_USER_ID } from './fixtures'

const session = { token: 'tok', userId: ME_USER_ID, serverId: 'srv' }
type Override = Record<string, { status: number; body?: unknown }>

/** Stubs fetch for ./api/<path> with the contract fixtures (or a status per path). */
function stubApi(override: Override = {}) {
  const fn = vi.fn(async (input: RequestInfo | URL) => {
    const url = new URL(String(input), 'http://jf.local/jellyfin/Jellystics/Me/')
    const path = url.pathname.replace('/jellyfin/Jellystics/Me/api/', '')
    const o = override[path] ?? override['*']
    if (o) return new Response(o.body === undefined ? '' : JSON.stringify(o.body), { status: o.status })
    const body = meResponse(path, url.searchParams)
    return body === undefined ? new Response('', { status: 404 }) : new Response(JSON.stringify(body), { status: 200 })
  })
  vi.stubGlobal('fetch', fn)
  const calls = (path: string) => fn.mock.calls
    .map((c) => new URL(String(c[0]), 'http://jf.local/jellyfin/Jellystics/Me/'))
    .filter((u) => u.pathname.endsWith(`/api/${path}`))
  return { fn, calls }
}

function renderMe(s: typeof session | null = session) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<MeApp session={s} basePath="/jellyfin" client={client} />)
}

const last = <T,>(xs: T[]): T => xs[xs.length - 1]

afterEach(() => vi.unstubAllGlobals())

describe('MePage', () => {
  it('renders every section from the contract fixtures', async () => {
    const api = stubApi()
    const { container } = renderMe()

    expect(await screen.findByRole('heading', { level: 2, name: 'Alice' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { level: 1, name: 'app.me.title' })).toBeInTheDocument()
    // KPIs, including the measured / estimated foot of the finished views
    expect(await screen.findByText('app.metric.completionAvg')).toBeInTheDocument()
    expect(screen.getByText('app.metric.finishedViews', { selector: 'span' })).toBeInTheDocument()
    expect(screen.getByText('app.plugin.playedToEnd.footMixed')).toBeInTheDocument()
    expect(screen.getByText('app.me.kpi.record:7')).toBeInTheDocument()
    // charts and cards
    for (const title of ['app.me.daily.titleDay', 'app.me.top.title', 'app.me.types.title', 'app.me.habits.title', 'app.me.plays.title', 'app.me.devices.title', 'app.me.clients.title', 'app.me.history.title']) {
      expect(await screen.findByRole('heading', { level: 3, name: title })).toBeInTheDocument()
    }
    expect(screen.getByText('Les Rivages', { selector: 'span' })).toBeInTheDocument()
    expect(screen.getByText('Inception', { selector: 'span' })).toBeInTheDocument()
    expect(screen.getByText('Living room TV')).toBeInTheDocument()
    expect(screen.getByText('app.me.habits.peak')).toBeInTheDocument()
    expect(await screen.findByText('Episode 1')).toBeInTheDocument()

    // posters and avatar come from Jellyfin, same origin, under the base path
    const srcs = [...container.querySelectorAll('img')].map((i) => i.getAttribute('src'))
    expect(srcs).toContain('/jellyfin/Items/s1/Images/Primary?fillWidth=64&quality=90')
    expect(srcs).toContain(`/jellyfin/Users/${ME_USER_ID}/Images/Primary?fillWidth=64&quality=90&tag=tag1`)
    // the only link goes back to Jellyfin: nothing points to the dashboard
    expect([...container.querySelectorAll('a')].map((a) => a.getAttribute('href'))).toEqual(['/jellyfin/web/'])
    // every call is relative to the page, carries the token and never a user id
    for (const [input, init] of api.fn.mock.calls as unknown as [string, RequestInit][]) {
      expect(input).toMatch(/^\.\/api\/[a-z]+(\?|$)/)
      expect(input).not.toMatch(/user/i)
      expect((init.headers as Record<string, string>).Authorization).toBe('MediaBrowser Token="tok"')
    }
    expect(api.calls('overview')[0].searchParams.get('compare')).toBe('1')
  })

  it('changes the period of every call', async () => {
    const api = stubApi()
    renderMe()
    await screen.findByRole('heading', { level: 2, name: 'Alice' })
    await userEvent.click(screen.getByRole('radio', { name: 'app.me.period.all' }))
    await waitFor(() => expect(api.calls('overview').some((u) => u.searchParams.get('from') === '2000-01-01' && !u.searchParams.has('compare'))).toBe(true))
    await waitFor(() => expect(api.calls('history').some((u) => u.searchParams.get('from') === '2000-01-01')).toBe(true))
    expect(window.location.search).toContain('range=all')
    await userEvent.click(screen.getByRole('radio', { name: 'app.me.period.30d' }))
    expect(window.location.search).not.toContain('range=')
  })

  it('paginates the history', async () => {
    const api = stubApi()
    renderMe()
    expect(await screen.findByText('Episode 1')).toBeInTheDocument()
    expect(screen.getByText('app.table.range')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'app.table.next' }))
    expect(await screen.findByText('Episode 27')).toBeInTheDocument()
    expect(screen.queryByText('Episode 1')).toBeNull()
    expect(last(api.calls('history')).searchParams.get('page')).toBe('2')
    expect(screen.getByRole('button', { name: 'app.table.next' })).toBeDisabled()

    await userEvent.click(screen.getByRole('combobox', { name: 'app.table.perPage' }))
    await userEvent.click(await screen.findByRole('option', { name: '50' }))
    await waitFor(() => expect(last(api.calls('history')).searchParams.get('pageSize')).toBe('50'))
    expect(last(api.calls('history')).searchParams.get('page')).toBe('1')
    expect(await screen.findByText('Episode 30')).toBeInTheDocument()
    expect(screen.getByText('Episode 1')).toBeInTheDocument()
  })

  it('shows empty states when there is no data', async () => {
    stubApi({ top: { status: 200, body: { items: [] } }, history: { status: 200, body: { rows: [], total: 0, page: 1, pageSize: 25 } }, scatter: { status: 200, body: { total: 0, limit: 5000, capped: false, points: [] } } })
    renderMe()
    expect(await screen.findByText('app.me.top.empty')).toBeInTheDocument()
    expect(await screen.findByText('app.me.history.empty')).toBeInTheDocument()
    expect(await screen.findByText('app.me.plays.empty')).toBeInTheDocument()
  })

  it('keeps the page when one section fails', async () => {
    stubApi({ devices: { status: 500, body: { error: 'internal error' } } })
    renderMe()
    expect(await screen.findByText('Les Rivages', { selector: 'span' })).toBeInTheDocument()
    const alerts = await screen.findAllByRole('alert', undefined, { timeout: 3000 })
    expect(alerts.length).toBe(2) // devices + apps cards
    expect(within(alerts[0]).getByText('app.me.sectionError')).toBeInTheDocument()
  })
})

describe('unavailable states', () => {
  it.each([
    [503, { error: 'notLinked' }, 'app.me.unavailable.notLinked'],
    [502, { error: 'unreachable' }, 'app.me.unavailable.unreachable'],
    [502, { error: 'secretMismatch' }, 'app.me.unavailable.relinking'],
    [504, undefined, 'app.me.unavailable.unreachable'],
    [401, { error: 'invalid secret' }, 'app.me.unavailable.notLinked'],
    [403, { error: 'unknown user' }, 'app.me.unavailable.forbidden'],
    [500, { error: 'internal error' }, 'app.me.unavailable.error'],
  ])('%i %j shows the reason', async (status, body, text) => {
    stubApi({ '*': { status, body } })
    renderMe()
    const alert = await screen.findByRole('alert', undefined, { timeout: 3000 }) // a plain error is retried once
    expect(within(alert).getByText('app.me.unavailable.title')).toBeInTheDocument()
    expect(within(alert).getByText(text)).toBeInTheDocument()
    expect(within(alert).getByRole('button', { name: 'app.action.retry' })).toBeInTheDocument()
    expect(within(alert).getByRole('link', { name: 'app.me.back' })).toHaveAttribute('href', '/jellyfin/web/')
  })

  it('retries', async () => {
    const api = stubApi({ '*': { status: 502, body: { error: 'unreachable' } } })
    renderMe()
    await userEvent.click(await screen.findByRole('button', { name: 'app.action.retry' }))
    await waitFor(() => expect(api.calls('profile').length).toBe(2))
  })

  it('asks to sign in to Jellyfin when the token is rejected', async () => {
    stubApi({ '*': { status: 401 } })
    renderMe()
    expect(await screen.findByText('app.me.signedOut.title')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'app.me.signedOut.action' })).toHaveAttribute('href', '/jellyfin/web/')
  })

  it('asks to sign in to Jellyfin without a token, without calling the API', () => {
    const api = stubApi()
    renderMe(null)
    expect(screen.getByText('app.me.signedOut.title')).toBeInTheDocument()
    expect(screen.getByText('app.me.signedOut.description')).toBeInTheDocument()
    expect(api.fn).not.toHaveBeenCalled()
  })
})
