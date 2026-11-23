import { formatDateTime, timeTicks, tzOffset, zonedDate } from './timeScale'

const PARIS = 'Europe/Paris'

it('resolves local midnights and offsets in the server time zone (DST aware)', () => {
  expect(zonedDate('2026-03-28', PARIS)).toBe(Date.parse('2026-03-27T23:00:00Z'))
  expect(zonedDate('2026-03-30', PARIS)).toBe(Date.parse('2026-03-29T22:00:00Z'))
  expect(tzOffset(Date.parse('2026-01-15T12:00:00Z'), PARIS)).toBe(3_600_000)
  expect(tzOffset(Date.parse('2026-07-15T12:00:00Z'), PARIS)).toBe(7_200_000)
  expect(zonedDate('2026-03-28', 'UTC')).toBe(Date.parse('2026-03-28T00:00:00Z'))
})

it('places daily ticks on local midnights with day labels', () => {
  const from = zonedDate('2026-03-01', PARIS), to = zonedDate('2026-03-08', PARIS)
  const { ticks, format } = timeTicks(from + 3_600_000, to, 10, 'fr', PARIS)
  expect(ticks).toHaveLength(7)
  ticks.forEach((t) => expect(tzOffset(t, PARIS) + t).toBe(Math.round((tzOffset(t, PARIS) + t) / 86_400_000) * 86_400_000))
  expect(format(ticks[0])).toBe('2 mars')
})

it('switches to weeks (Mondays), months and hours with the span', () => {
  const weeks = timeTicks(zonedDate('2026-03-01', PARIS), zonedDate('2026-05-01', PARIS), 10, 'fr', PARIS)
  weeks.ticks.forEach((t) => expect(new Intl.DateTimeFormat('en-US', { weekday: 'short', timeZone: PARIS }).format(t)).toBe('Mon'))
  expect(weeks.ticks[0]).toBe(zonedDate('2026-03-02', PARIS))

  const months = timeTicks(zonedDate('2025-01-10', PARIS), zonedDate('2026-01-10', PARIS), 6, 'fr', PARIS)
  expect(months.ticks.map((t) => months.format(t))).toEqual(['avr. 25', 'juil. 25', 'oct. 25', 'janv. 26'])

  const hours = timeTicks(Date.parse('2026-03-28T20:10:00Z'), Date.parse('2026-03-29T08:00:00Z'), 6, 'fr', PARIS)
  expect(hours.ticks.map((t) => hours.format(t))).toEqual(['00:00', '03:00', '06:00', '09:00'].map((h, i) => (i === 0 ? '29 mars' : h)))
})

it('formats tooltip date-times in the server zone', () => {
  expect(formatDateTime(Date.parse('2026-03-28T23:30:00Z'), 'fr', PARIS)).toBe('29 mars 2026, 00:30')
})
