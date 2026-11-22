import { cleanParams, qk } from './keys'

it('normalises params: drops empty values and false, booleans as 1, sorted keys', () => {
  const p = cleanParams({ to: '2026-11-23', from: '2026-10-24', compare: true, userId: undefined, q: '', views: false, page: 2, libraryId: null })
  expect(p).toEqual({ compare: 1, from: '2026-10-24', page: 2, to: '2026-11-23' })
  expect(Object.keys(p)).toEqual(['compare', 'from', 'page', 'to'])
})

it('sends desc explicitly as 0/1 so ascending sorts are reachable', () => {
  expect(cleanParams({ sort: 'views', desc: false })).toEqual({ desc: 0, sort: 'views' })
  expect(cleanParams({ sort: 'views', desc: true })).toEqual({ desc: 1, sort: 'views' })
  expect(cleanParams({ sort: 'views' })).toEqual({ sort: 'views' })
})

it('builds keys that carry period, compare and filters', () => {
  const p = cleanParams({ from: '2026-11-01', to: '2026-11-08', compare: true, type: 'movie' })
  expect(qk.overview(p)).toEqual(['v2', 'overview', { compare: 1, from: '2026-11-01', to: '2026-11-08', type: 'movie' }])
  expect(qk.movie('m1', p)).toEqual(['v2', 'movie', 'm1', p])
  expect(qk.historyCalendar({ month: '2026-11' })).toEqual(['v2', 'history', 'calendar', { month: '2026-11' }])
  expect(qk.search('abc')).toEqual(['v2', 'search', 'abc'])
})

it('equal requests share a key; different periods do not', () => {
  const a = cleanParams({ from: '2026-11-01', to: '2026-11-08', compare: false })
  const b = cleanParams({ to: '2026-11-08', from: '2026-11-01' })
  const c = cleanParams({ from: '2026-11-02', to: '2026-11-08' })
  expect(JSON.stringify(qk.users(a))).toBe(JSON.stringify(qk.users(b)))
  expect(JSON.stringify(qk.users(a))).not.toBe(JSON.stringify(qk.users(c)))
  expect(qk.all).toEqual(['v2'])
})
