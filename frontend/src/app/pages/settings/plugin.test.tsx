import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { Toaster } from '@/ui'
import { PLUGIN_MANIFEST_URL } from '@/api/plugin'
import { DataQualityCard, PluginCard } from './sections/PluginCards'

interface Call { method: string; url: string }
let calls: Call[]
let routes: Record<string, () => unknown>

beforeEach(() => {
  calls = []; routes = {}
  vi.stubGlobal('fetch', vi.fn(async (url: string, init?: RequestInit) => {
    const method = init?.method ?? 'GET'
    calls.push({ method, url })
    const h = routes[`${method} ${url.split('?')[0]}`]
    if (!h) return new Response('', { status: 404 })
    return new Response(JSON.stringify(h() ?? {}), { status: 200 })
  }))
})
afterEach(() => { vi.unstubAllGlobals() })
const hit = (method: string, url: string) => calls.filter((c) => c.method === method && c.url.split('?')[0] === url)

function renderCard(card: React.ReactNode) {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter initialEntries={['/settings/config']}>{card}</MemoryRouter>
      <Toaster />
    </QueryClientProvider>,
  )
}

const detected = {
  detected: true, pluginVersion: '1.0.0.0', schemaVersion: 1, supportedSchemaVersion: 1, cursorPlays: 42,
  lastCheckedAt: new Date().toISOString(), lastSuccessAt: new Date().toISOString(), lastError: null, healthy: true, playsSynced: 1234,
}

it('without the plugin: explains it and copies the manifest URL', async () => {
  const writeText = vi.fn().mockResolvedValue(undefined)
  Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true })
  routes = { 'GET /api/plugin/status': () => ({ ...detected, detected: false, healthy: false, playsSynced: 0 }) }
  renderCard(<PluginCard />)
  expect(await screen.findByText('app.settings.plugin.state.absent')).toBeInTheDocument()
  expect(screen.getByText('app.settings.plugin.absent.b1')).toBeInTheDocument()
  expect(screen.getByText(PLUGIN_MANIFEST_URL)).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: 'app.settings.plugin.push.start' })).not.toBeInTheDocument()
  await userEvent.click(screen.getByRole('button', { name: 'app.settings.plugin.absent.copy' }))
  expect(writeText).toHaveBeenCalledWith(PLUGIN_MANIFEST_URL)
})

it('with the plugin: shows the status, pushes the history and resyncs behind a confirmation', async () => {
  routes = {
    'GET /api/plugin/status': () => detected,
    'GET /api/plugin/push-history/status': () => ({ state: 'done', total: 10, sent: 10, imported: 8, skipped: 2, rejected: 0, resumable: false }),
    'POST /api/plugin/push-history': () => ({ state: 'running', total: 10, sent: 0, imported: 0, skipped: 0, rejected: 0, resumable: false }),
    'POST /api/plugin/resync': () => ({ message: 'resync started' }),
  }
  renderCard(<PluginCard />)
  expect(await screen.findByText('app.settings.plugin.state.healthy')).toBeInTheDocument()
  expect(screen.getByText('1.0.0.0')).toBeInTheDocument()
  expect(screen.getByText(/1\s?234/)).toBeInTheDocument()
  expect(await screen.findByText('app.settings.plugin.push.done')).toBeInTheDocument()

  await userEvent.click(screen.getByRole('button', { name: 'app.settings.plugin.push.start' }))
  await waitFor(() => expect(hit('POST', '/api/plugin/push-history')).toHaveLength(1))
  expect(await screen.findByRole('progressbar', { name: 'app.settings.plugin.push.title' })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'app.settings.plugin.push.start' })).toBeDisabled()

  await userEvent.click(screen.getByRole('button', { name: 'app.settings.plugin.resync.title' }))
  const dialog = await screen.findByRole('dialog')
  expect(hit('POST', '/api/plugin/resync')).toHaveLength(0)
  await userEvent.click(within(dialog).getByRole('button', { name: 'app.settings.plugin.resync.confirm' }))
  await waitFor(() => expect(hit('POST', '/api/plugin/resync')).toHaveLength(1))
})

it('offers to resume an interrupted push and shows sync errors', async () => {
  routes = {
    'GET /api/plugin/status': () => ({ ...detected, healthy: false, lastError: 'Plays: status 500' }),
    'GET /api/plugin/push-history/status': () => ({ state: 'interrupted', total: 10, sent: 5, imported: 5, skipped: 0, rejected: 0, resumable: true }),
  }
  renderCard(<PluginCard />)
  expect(await screen.findByText('app.settings.plugin.state.unhealthy')).toBeInTheDocument()
  expect(screen.getByText(/Plays: status 500/)).toBeInTheDocument()
  expect(await screen.findByRole('button', { name: 'app.settings.plugin.push.resume' })).toBeEnabled()
  expect(screen.getByText('app.settings.plugin.push.interrupted')).toBeInTheDocument()
})

it('data quality: lists the sources of the period, most precise first, with their precision', async () => {
  const src = (key: string, plays: number, watchSeconds: number, watchShare: number) => ({
    key, plays, views: plays, watchSeconds, playShare: 0, watchShare, repaired: 0, captured: 0, positions: 0, streams: 0, transcodes: 0, transcodeDetailed: 0,
  })
  routes = {
    'GET /api/v2/data-quality': () => ({
      totals: { plays: 10, views: 10, watchSeconds: 5000, repaired: 2, captured: 4, positions: 8, streams: 4, transcodes: 2, transcodeDetailed: 1 },
      sources: [src('tracker', 3, 1000, 0.2), src('plugin', 5, 3000, 0.6), src('playbackReporting', 2, 1000, 0.2), src('restore', 0, 0, 0)],
      pluginCollectedSince: '2026-11-01', collectedSince: '2026-10-01', undated: 3,
    }),
  }
  renderCard(<DataQualityCard />)
  const items = await screen.findAllByRole('listitem')
  const labels = items.map((li) => li.textContent).filter((s) => s?.includes('.label'))
  expect(labels[0]).toContain('app.settings.quality.sources.plugin.label')
  expect(labels[1]).toContain('app.settings.quality.sources.tracker.label')
  expect(labels[2]).toContain('app.settings.quality.sources.playbackReporting.label')
  expect(screen.queryByText('app.settings.quality.sources.restore.label')).not.toBeInTheDocument()
  expect(screen.getByText('app.settings.quality.sources.plugin.precision')).toBeInTheDocument()
  expect(screen.getByText('app.settings.quality.plays:5')).toBeInTheDocument()
  expect(screen.getByText('app.settings.quality.repaired:2')).toBeInTheDocument()
  expect(screen.getByText('app.settings.quality.undated:3')).toBeInTheDocument()
  expect(screen.getByText('app.settings.quality.pluginSince')).toBeInTheDocument()
  expect(screen.getByRole('progressbar', { name: 'app.settings.quality.details.transcode' })).toHaveAttribute('aria-valuenow', '50')
  expect(hit('GET', '/api/v2/data-quality')[0].url).toMatch(/from=\d{4}-\d{2}-\d{2}/)
})

it('data quality: empty period', async () => {
  routes = { 'GET /api/v2/data-quality': () => ({ totals: { plays: 0 }, sources: [], pluginCollectedSince: null, collectedSince: null, undated: 0 }) }
  renderCard(<DataQualityCard />)
  expect(await screen.findByText('app.settings.quality.empty')).toBeInTheDocument()
})
