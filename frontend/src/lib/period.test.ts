import { parsePeriod, serializePeriod, previousPeriod, todayIn } from './period'
const P = (s: string) => new URLSearchParams(s)
const today = '2026-11-22'

it('defaults to the last 30 days, end exclusive', () => {
  expect(parsePeriod(P(''), today)).toEqual({ preset: '30d', from: '2026-10-24', to: '2026-11-23', compare: false })
})
it('presets', () => {
  expect(parsePeriod(P('range=7d'), today)).toMatchObject({ from: '2026-11-16', to: '2026-11-23' })
  expect(parsePeriod(P('range=today'), today)).toMatchObject({ from: '2026-11-22', to: '2026-11-23' })
  expect(parsePeriod(P('range=mtd'), today)).toMatchObject({ from: '2026-11-01', to: '2026-11-23' })
  expect(parsePeriod(P('range=ytd'), today)).toMatchObject({ from: '2026-01-01', to: '2026-11-23' })
  expect(parsePeriod(P('range=all'), today)).toMatchObject({ preset: 'all', from: '2000-01-01', to: '2026-11-23' })
})
it('custom from/to wins and invalid input falls back', () => {
  expect(parsePeriod(P('range=7d&from=2026-10-01&to=2026-11-01'), today)).toEqual({ preset: 'custom', from: '2026-10-01', to: '2026-11-01', compare: false })
  expect(parsePeriod(P('from=nope&to=2026-11-01'), today).preset).toBe('30d')
  expect(parsePeriod(P('from=2026-11-05&to=2026-11-01'), today).preset).toBe('30d')
})
it('compare flag', () => { expect(parsePeriod(P('compare=1'), today).compare).toBe(true) })
it('serializes to clean params', () => {
  expect(serializePeriod({ preset: '30d', from: '', to: '', compare: false })).toEqual({})
  expect(serializePeriod({ preset: '7d', from: '', to: '', compare: true })).toEqual({ range: '7d', compare: '1' })
  expect(serializePeriod({ preset: 'custom', from: '2026-10-01', to: '2026-11-01', compare: false })).toEqual({ from: '2026-10-01', to: '2026-11-01' })
})
it('previous period has the same length', () => {
  expect(previousPeriod({ preset: '7d', from: '2026-11-16', to: '2026-11-23', compare: true })).toEqual({ from: '2026-11-09', to: '2026-11-16' })
})
it('today in a timezone', () => {
  const d = new Date(Date.UTC(2026, 10, 21, 23, 30)) // 00:30 in Paris on the 22nd
  expect(todayIn('Europe/Paris', d)).toBe('2026-11-22')
  expect(todayIn('UTC', d)).toBe('2026-11-21')
})
it('rejects impossible calendar dates', () => {
  expect(parsePeriod(P('from=2026-02-31&to=2026-03-05'), today).preset).toBe('30d')
  expect(parsePeriod(P('from=2026-02-28&to=2026-03-05'), today).preset).toBe('custom')
})
it('ignores compare for the "all" preset', () => {
  expect(parsePeriod(P('range=all&compare=1'), today).compare).toBe(false)
  expect(serializePeriod({ preset: 'all', from: '2000-01-01', to: '2026-11-23', compare: true })).toEqual({ range: 'all' })
})
it('previous period of "all" collapses and of a custom range has the same length', () => {
  expect(previousPeriod({ preset: 'all', from: '2000-01-01', to: '2026-11-23', compare: false })).toEqual({ from: '2000-01-01', to: '2000-01-01' })
  expect(previousPeriod({ preset: 'custom', from: '2026-10-10', to: '2026-10-20', compare: true })).toEqual({ from: '2026-09-30', to: '2026-10-10' })
})
