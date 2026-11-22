import { act, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { Toaster } from '@/ui'
import { SettingsPage } from './SettingsPage'
import type { SettingsSection } from './sections'

let socketHandlers: Record<string, (d: unknown) => void> = {}
vi.mock('@/lib/socket', () => ({
  default: { connect: vi.fn(), disconnect: vi.fn() },
  useSocketEvent: (ev: string, h: (d: unknown) => void) => { socketHandlers[ev] = h },
}))

interface Call { method: string; url: string; body?: unknown; auth?: string | null }
let calls: Call[]
let routes: Record<string, (c: Call) => unknown>

function stubFetch() {
  calls = []
  vi.stubGlobal('fetch', vi.fn(async (url: string, init?: RequestInit) => {
    const method = init?.method ?? 'GET'
    const headers = new Headers(init?.headers)
    const body = typeof init?.body === 'string' ? JSON.parse(init.body) : init?.body
    const call: Call = { method, url, body, auth: headers.get('Authorization') }
    calls.push(call)
    const key = `${method} ${url.split('?')[0]}`
    const h = routes[key]
    if (!h) return new Response('', { status: 404 })
    const out = h(call)
    if (out instanceof Response) return out
    return new Response(JSON.stringify(out ?? {}), { status: 200 })
  }))
}
const hit = (method: string, url: string) => calls.filter((c) => c.method === method && c.url.split('?')[0] === url)

function renderSection(section: SettingsSection) {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter initialEntries={[`/settings/${section}`]}>
        <Routes><Route path="/settings/:section" element={<SettingsPage section={section} />} /></Routes>
      </MemoryRouter>
      <Toaster />
    </QueryClientProvider>,
  )
}

beforeEach(() => { socketHandlers = {}; routes = {}; localStorage.setItem('jellystics-token', 'TOK'); stubFetch() })
afterEach(() => { vi.unstubAllGlobals(); localStorage.clear() })

it('config: validates the URL, tests the connection and saves', async () => {
  routes = {
    'GET /api/getconfig': () => ({ JF_HOST: 'http://jf:8096', app_url: '' }),
    'GET /api/TrackedLibraries': () => [{ Id: 'l1', Name: 'Films', Tracked: true }],
    'GET /api/users': () => [{ Id: 'u1', Name: 'Léa' }],
    'GET /api/UntrackedUsers': () => ['u1'],
    'POST /proxy/validateSettings': () => ({ isValid: true, serverName: 'MyJelly' }),
    'POST /api/setconfig': () => ({ ok: true }),
    'POST /api/setExcludedLibraries': () => 'ok',
  }
  renderSection('config')
  const url = await screen.findByLabelText('app.settings.config.url')
  await waitFor(() => expect(url).toHaveValue('http://jf:8096'))
  await userEvent.clear(url); await userEvent.type(url, 'nope')
  await userEvent.click(screen.getByRole('button', { name: 'app.settings.action.save' }))
  expect(await screen.findByRole('alert')).toHaveTextContent('app.settings.config.urlInvalid')
  expect(hit('POST', '/api/setconfig')).toHaveLength(0)

  await userEvent.clear(url); await userEvent.type(url, 'http://jf:8096')
  await userEvent.type(screen.getByLabelText('app.settings.config.apiKey'), 'SECRET')
  await userEvent.click(screen.getByRole('button', { name: 'app.settings.config.test' }))
  expect(await screen.findByText('app.settings.config.testOk')).toBeInTheDocument()
  expect(hit('POST', '/proxy/validateSettings')[0].body).toEqual({ url: 'http://jf:8096', apikey: 'SECRET' })

  await userEvent.click(screen.getByRole('button', { name: 'app.settings.action.save' }))
  await waitFor(() => expect(hit('POST', '/api/setconfig')).toHaveLength(1))
  expect(hit('POST', '/api/setconfig')[0].body).toEqual({ JF_HOST: 'http://jf:8096', JF_API_KEY: 'SECRET', app_url: '' })

  await userEvent.click(screen.getByRole('switch', { name: 'Films' }))
  await waitFor(() => expect(hit('POST', '/api/setExcludedLibraries')[0]?.body).toEqual({ libraryID: 'l1' }))
  expect(screen.getByRole('switch', { name: 'Léa' })).toHaveAttribute('aria-checked', 'false')
})

it('tasks: runs a task, saves a cron and streams logs', async () => {
  routes = {
    'GET /api/getTasks': () => [{ name: 'Backup', displayName: 'Backup', running: false, lastRun: null }, { name: 'Full Jellyfin Sync', displayName: 'Full sync', running: false }],
    'GET /api/getTaskSettings': () => ({ Backup: { enabled: true, cronExpression: '0 3 * * *' } }),
    'POST /api/runTask/Backup': () => ({}),
    'POST /api/setTaskSettings': () => ({}),
  }
  renderSection('tasks')
  expect(await screen.findAllByText('app.settings.tasks.neverRun')).toHaveLength(2)
  const runs = screen.getAllByRole('button', { name: 'app.settings.tasks.run' })
  await userEvent.click(runs[0])
  await waitFor(() => expect(hit('POST', '/api/runTask/Backup')).toHaveLength(1))

  const row = within(screen.getByText('Backup').closest('li')!)
  const cron = row.getByLabelText('app.settings.tasks.cron')
  await userEvent.clear(cron); await userEvent.type(cron, 'bad')
  await userEvent.click(row.getByRole('button', { name: 'app.settings.action.save' }))
  expect(await row.findByRole('alert')).toHaveTextContent('app.settings.tasks.cronInvalid')
  await userEvent.clear(cron); await userEvent.type(cron, '5 4 * * *')
  await userEvent.click(row.getByRole('button', { name: 'app.settings.action.save' }))
  await waitFor(() => expect(hit('POST', '/api/setTaskSettings')[0]?.body).toEqual({ taskname: 'Backup', cronExpression: '5 4 * * *' }))

  act(() => socketHandlers.TaskLog('hello from sync'))
  expect(within(screen.getByRole('log')).getByText('hello from sync')).toBeInTheDocument()
})

it('backup: creates, downloads with Authorization, restores and deletes behind confirmation', async () => {
  const createUrl = vi.fn(() => 'blob:x'); const revoke = vi.fn()
  Object.assign(URL, { createObjectURL: createUrl, revokeObjectURL: revoke })
  vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
  routes = {
    'GET /backup/files': () => [{ name: 'b1.json', size: 2048, datecreated: '2026-01-02T03:04:05Z' }],
    'GET /api/getconfig': () => ({ settings: { KeepLogsForDays: 14 } }),
    'GET /backup/beginBackup': () => ({}),
    'GET /backup/files/b1.json': () => new Response('{"a":1}', { status: 200 }),
    'GET /backup/restore/b1.json': () => ({ restored: { a: 2, b: 3 } }),
    'DELETE /backup/files/b1.json': () => ({}),
  }
  renderSection('backup')
  expect(await screen.findByText('b1.json')).toBeInTheDocument()
  await userEvent.click(screen.getByRole('button', { name: 'app.settings.backup.create' }))
  await waitFor(() => expect(hit('GET', '/backup/beginBackup')).toHaveLength(1))

  const actions = () => userEvent.click(screen.getByRole('button', { name: 'app.settings.backup.actionsFor' }))
  await actions()
  await userEvent.click(await screen.findByRole('menuitem', { name: 'app.settings.action.download' }))
  await waitFor(() => expect(createUrl).toHaveBeenCalled())
  const dl = calls.find((c) => c.url === '/backup/files/b1.json')!
  expect(dl.auth).toBe('Bearer TOK')

  await actions()
  await userEvent.click(await screen.findByRole('menuitem', { name: 'app.settings.backup.restore' }))
  const dlg = await screen.findByRole('dialog')
  expect(hit('GET', '/backup/restore/b1.json')).toHaveLength(0)
  await userEvent.click(within(dlg).getByRole('button', { name: 'app.settings.backup.restore' }))
  await waitFor(() => expect(hit('GET', '/backup/restore/b1.json')).toHaveLength(1))

  await actions()
  await userEvent.click(await screen.findByRole('menuitem', { name: 'app.settings.action.delete' }))
  await userEvent.click(within(await screen.findByRole('dialog')).getByRole('button', { name: 'app.settings.action.delete' }))
  await waitFor(() => expect(hit('DELETE', '/backup/files/b1.json')).toHaveLength(1))
})

it('backup: uploads a JSON file', async () => {
  routes = { 'GET /backup/files': () => [], 'GET /api/getconfig': () => ({}), 'POST /backup/upload': () => ({ fileName: 'x.json' }) }
  renderSection('backup')
  await screen.findByText('app.settings.backup.empty')
  const file = new File(['{}'], 'x.json', { type: 'application/json' })
  await userEvent.upload(screen.getByTestId('backup-file'), file)
  await waitFor(() => expect(hit('POST', '/backup/upload')).toHaveLength(1))
  expect(hit('POST', '/backup/upload')[0].auth).toBe('Bearer TOK')
})

it('backup: keeping logs preserves the current app_url', async () => {
  routes = {
    'GET /backup/files': () => [],
    'GET /api/getconfig': () => ({ app_url: 'https://js.example.com', settings: { KeepLogsForDays: 14 } }),
    'POST /api/setconfig': () => ({}),
  }
  renderSection('backup')
  const days = await screen.findByLabelText('app.settings.backup.keepDays')
  await waitFor(() => expect(days).toHaveValue(14))
  await userEvent.clear(days); await userEvent.type(days, '20')
  await userEvent.click(screen.getByRole('button', { name: 'app.settings.action.save' }))
  await waitFor(() => expect(hit('POST', '/api/setconfig')).toHaveLength(1))
  expect(hit('POST', '/api/setconfig')[0].body).toEqual({ KeepLogsForDays: 20, app_url: 'https://js.example.com' })
})

it('backup: a failed download shows the dedicated error', async () => {
  vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
  routes = {
    'GET /backup/files': () => [{ name: 'b1.json', size: 2048, datecreated: '2026-01-02T03:04:05Z' }],
    'GET /api/getconfig': () => ({}),
    'GET /backup/files/b1.json': () => new Response('', { status: 500 }),
  }
  renderSection('backup')
  await screen.findByText('b1.json')
  await userEvent.click(screen.getByRole('button', { name: 'app.settings.backup.actionsFor' }))
  await userEvent.click(await screen.findByRole('menuitem', { name: 'app.settings.action.download' }))
  expect(await screen.findByText('app.settings.backup.downloadError')).toBeInTheDocument()
})

it('config: shows the server timezone, not the browser one', async () => {
  routes = {
    'GET /api/getconfig': () => ({}),
    'GET /api/TrackedLibraries': () => [], 'GET /api/users': () => [], 'GET /api/UntrackedUsers': () => [],
    'GET /auth/isConfigured': () => ({ state: 1, timezone: 'Pacific/Auckland' }),
  }
  renderSection('config')
  expect(await screen.findByText('Pacific/Auckland')).toBeInTheDocument()
})

it('tasks: imports a Playback Reporting file', async () => {
  routes = {
    'GET /api/getTasks': () => [], 'GET /api/getTaskSettings': () => ({}),
    'POST /sync/importPlaybackBackup': () => ({ imported: 3 }),
  }
  renderSection('tasks')
  await screen.findByText('app.settings.tasks.empty')
  const file = new File(['a\tb'], 'pb.tsv', { type: 'text/tab-separated-values' })
  await userEvent.upload(screen.getByTestId('import-file'), file)
  await userEvent.click(await screen.findByRole('button', { name: 'app.settings.tasks.import' }))
  await waitFor(() => expect(hit('POST', '/sync/importPlaybackBackup')).toHaveLength(1))
  expect(hit('POST', '/sync/importPlaybackBackup')[0].auth).toBe('Bearer TOK')
  expect(hit('POST', '/sync/importPlaybackBackup')[0].body).toBeInstanceOf(FormData)
})

it('settings: 401 on upload/download clears the session', async () => {
  const assign = vi.fn()
  const orig = Object.getOwnPropertyDescriptor(window, 'location')!
  Object.defineProperty(window, 'location', { value: { ...window.location, pathname: '/settings/backup', search: '', assign }, writable: true, configurable: true })
  vi.stubGlobal('fetch', vi.fn(async () => new Response('', { status: 401 })))
  const { uploadFile, downloadAuthed } = await import('./net')
  await expect(uploadFile('/backup/upload', new File(['{}'], 'x.json'))).rejects.toMatchObject({ status: 401 })
  expect(localStorage.getItem('jellystics-token')).toBeNull()
  expect(assign).toHaveBeenCalledWith('/login?next=%2Fsettings%2Fbackup')
  localStorage.setItem('jellystics-token', 'TOK'); assign.mockClear()
  await expect(downloadAuthed('/backup/files/x', 'x')).rejects.toMatchObject({ status: 401 })
  expect(localStorage.getItem('jellystics-token')).toBeNull()
  expect(assign).toHaveBeenCalledTimes(1)
  Object.defineProperty(window, 'location', orig)
})

it('webhooks: hides the token and tolerates invalid dates', async () => {
  routes = { 'GET /webhooks/': () => [{ id: 1, name: 'Ops', url: 'https://discord.com/api/webhooks/123/SECRETTOKEN', discord_events: [], enabled: true, last_triggered: 'garbage' }] }
  renderSection('webhooks')
  await screen.findByText('Ops')
  expect(screen.getByText('discord.com/api/webhooks/…')).toBeInTheDocument()
  expect(document.body.textContent).not.toContain('SECRETTOKEN')
  expect(screen.getByText('app.settings.webhooks.lastTriggered')).toBeInTheDocument()
})

it('logs: maps results to levels and filters by search', async () => {
  routes = {
    'GET /logs/getLogs': () => [
      { Name: 'Full sync', Result: 'Success', TimeRun: '2026-01-01T10:00:00Z', Log: '[{"Message":"all good"}]' },
      { Name: 'Backup', Result: 'Failed', TimeRun: '2026-01-01T11:00:00Z', Log: 'disk full' },
    ],
  }
  renderSection('logs')
  expect(await screen.findByText('all good')).toBeInTheDocument()
  expect(screen.getByText('disk full')).toBeInTheDocument()
  await userEvent.type(screen.getByPlaceholderText('app.settings.logs.search'), 'disk')
  expect(screen.queryByText('all good')).not.toBeInTheDocument()
  expect(screen.getByText('disk full')).toBeInTheDocument()
})

it('webhooks: validates, creates, toggles, tests and deletes', async () => {
  const hook = { id: 7, name: 'Ops', url: 'https://discord.com/api/webhooks/1/abc', bot_username: 'bot', discord_events: ['task_failed'], enabled: true }
  routes = {
    'GET /webhooks/': () => [hook],
    'POST /webhooks/': () => ({}),
    'PUT /webhooks/7': () => ({}),
    'POST /webhooks/7/test': () => ({}),
    'DELETE /webhooks/7': () => ({}),
  }
  renderSection('webhooks')
  expect(await screen.findByText('Ops')).toBeInTheDocument()

  await userEvent.click(screen.getByRole('button', { name: 'app.settings.webhooks.add' }))
  const dlg = await screen.findByRole('dialog')
  await userEvent.click(within(dlg).getByRole('button', { name: 'app.settings.action.create' }))
  expect(await within(dlg).findByText('app.settings.webhooks.nameRequired')).toBeInTheDocument()
  expect(within(dlg).getByText('app.settings.webhooks.urlInvalid')).toBeInTheDocument()
  expect(within(dlg).getByText('app.settings.webhooks.eventsRequired')).toBeInTheDocument()
  expect(hit('POST', '/webhooks/')).toHaveLength(0)

  await userEvent.type(within(dlg).getByLabelText('app.settings.webhooks.name'), 'New')
  await userEvent.type(within(dlg).getByLabelText('app.settings.webhooks.url'), 'https://discord.com/api/webhooks/2/xyz')
  await userEvent.click(within(dlg).getByRole('checkbox', { name: 'app.settings.webhooks.events.task_start' }))
  await userEvent.click(within(dlg).getByRole('button', { name: 'app.settings.action.create' }))
  await waitFor(() => expect(hit('POST', '/webhooks/')).toHaveLength(1))
  expect(hit('POST', '/webhooks/')[0].body).toMatchObject({ name: 'New', trigger_type: 'discord', discord_events: ['task_start'] })

  await userEvent.click(screen.getByRole('switch', { name: 'app.settings.webhooks.enabled' }))
  await waitFor(() => expect(hit('PUT', '/webhooks/7')[0]?.body).toMatchObject({ id: 7, enabled: false }))
  await userEvent.click(screen.getByRole('button', { name: 'app.settings.webhooks.test' }))
  await waitFor(() => expect(hit('POST', '/webhooks/7/test')).toHaveLength(1))

  await userEvent.click(screen.getByRole('button', { name: 'app.settings.webhooks.delete' }))
  await userEvent.click(within(await screen.findByRole('dialog')).getByRole('button', { name: 'app.settings.action.delete' }))
  await waitFor(() => expect(hit('DELETE', '/webhooks/7')).toHaveLength(1))
})

it('apikeys: creates a key and shows it once, then deletes', async () => {
  let keys = [{ name: 'old', key: 'abcdefgh12345678', createdAt: '2026-01-01T00:00:00Z' }]
  routes = {
    'GET /api/keys': () => keys,
    'POST /api/keys': (c) => { const k = { name: (c.body as { name: string }).name, key: 'NEWKEY123456', createdAt: '2026-02-02T00:00:00Z' }; keys = [...keys, k]; return k },
    'DELETE /api/keys': () => ({}),
  }
  renderSection('apikeys')
  expect(await screen.findByText('old')).toBeInTheDocument()
  expect(screen.queryByText('abcdefgh12345678')).not.toBeInTheDocument()

  await userEvent.click(screen.getByRole('button', { name: 'app.settings.apikeys.create' }))
  const dlg = await screen.findByRole('dialog')
  await userEvent.click(within(dlg).getByRole('button', { name: 'app.settings.action.create' }))
  expect(await within(dlg).findByText('app.settings.apikeys.nameRequired')).toBeInTheDocument()
  await userEvent.type(within(dlg).getByLabelText('app.settings.apikeys.name'), 'ci')
  await userEvent.click(within(dlg).getByRole('button', { name: 'app.settings.action.create' }))
  expect(await screen.findByTestId('new-key')).toHaveTextContent('NEWKEY123456')
  await userEvent.click(screen.getByRole('button', { name: 'app.settings.action.done' }))
  await waitFor(() => expect(screen.queryByTestId('new-key')).not.toBeInTheDocument())

  await userEvent.click(screen.getAllByRole('button', { name: 'app.settings.apikeys.actionsFor' })[0])
  await userEvent.click(await screen.findByRole('menuitem', { name: 'app.settings.action.delete' }))
  await userEvent.click(within(await screen.findByRole('dialog')).getByRole('button', { name: 'app.settings.action.delete' }))
  await waitFor(() => expect(hit('DELETE', '/api/keys')[0]?.body).toEqual({ key: 'abcdefgh12345678' }))
})

it('about: shows versions and the update link', async () => {
  routes = { 'GET /api/CheckForUpdates': () => ({ current_version: '1.0.0', latest_version: 'v1.2.0', update_available: true, message: 'New version' }) }
  renderSection('about')
  expect(await screen.findByText('v1.0.0')).toBeInTheDocument()
  expect(screen.getByText('v1.2.0')).toBeInTheDocument()
  expect(screen.getByRole('link', { name: 'app.settings.about.download' })).toHaveAttribute('href', expect.stringContaining('/releases/latest'))
})

it('tabs navigate between sections', async () => {
  routes = { 'GET /api/CheckForUpdates': () => ({ current_version: '1.0.0' }), 'GET /logs/getLogs': () => [] }
  render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter initialEntries={['/settings/about']}>
        <Routes><Route path="/settings/:section" element={<Wrapper />} /></Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
  expect(screen.getByRole('tabpanel')).toHaveAccessibleName('app.settings.sections.about')
  await userEvent.click(await screen.findByRole('tab', { name: 'app.settings.sections.logs' }))
  expect(await screen.findByText('app.settings.logs.empty')).toBeInTheDocument()
})

import { useParams } from 'react-router-dom'
function Wrapper() { return <SettingsPage section={useParams().section as SettingsSection} /> }
