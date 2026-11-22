import { screen } from '@testing-library/react'
import { LanguagesCard } from './LanguagesCard'
import { mockApi, renderPage } from '@/app/pages/_shared/testUtils'

beforeEach(() => { vi.restoreAllMocks() })

const DATA = {
  collectedSince: '2026-11-01', captured: 10,
  split: { originalSubtitled: 5, original: 2, dubbed: 3, unknown: 0 },
  audio: [{ label: 'eng', views: 7 }, { label: 'fre', views: 3 }],
  subtitles: [{ label: 'fre', views: 5 }],
  subtitleOff: 5,
}

it('shows the VO/VF split, audio languages and subtitle share for the period', async () => {
  const spy = mockApi({ '/api/v2/users/u1/languages': DATA })
  renderPage(<LanguagesCard userId="u1" />, { path: '/users/:id', url: '/users/u1?from=2026-11-01&to=2026-11-22' })
  expect(await screen.findByText('anglais')).toBeInTheDocument()
  expect(screen.getByText('français')).toBeInTheDocument()
  expect(screen.getByText(/app.languages.subtitles · français/)).toBeInTheDocument()
  expect(screen.getByLabelText(/app.languages.originalSubtitled/)).toBeInTheDocument()
  expect(screen.getByText('app.languages.original')).toBeInTheDocument()
  const url = new URL(String(spy.mock.calls[0][0]), 'http://x')
  expect(url.searchParams.get('from')).toBe('2026-11-01')
})

it('shows the collection empty state without captured data', async () => {
  mockApi({ '/api/v2/users/u1/languages': { ...DATA, collectedSince: null, captured: 0, audio: [], subtitles: [], subtitleOff: 0 } })
  renderPage(<LanguagesCard userId="u1" />, { path: '/users/:id', url: '/users/u1' })
  expect(await screen.findByText('app.analytics.notCollected')).toBeInTheDocument()
  expect(screen.getByText('app.analytics.collectedNext')).toBeInTheDocument()
})
