import { formatDuration, formatPosition, formatNumber, formatCompact, formatPercent, formatHour, formatDay, formatRelative, formatDelta, weekdayLabels, formatBytes } from './format'

const nb = (s: string) => s.replace(/[\u{a0}\u{202f}]/gu, ' ') // normalise fr spaces (NBSP, narrow NBSP)

describe('formatDuration', () => {
  it.each([[0, '0 s'], [45, '45 s'], [59.6, '1 min'], [720, '12 min'], [3900, '1 h 05'], [3600 * 99 + 59 * 60, '99 h 59'], [3600 * 1248, '1 248 h']])('%s s → %s', (s, out) => {
    expect(nb(formatDuration(s, 'fr'))).toBe(out)
  })
})
describe('formatPosition', () => {
  it('formats media positions', () => {
    expect(formatPosition(309)).toBe('5:09')
    expect(formatPosition(3912)).toBe('1:05:12')
  })
})
describe('numbers', () => {
  it('uses the locale grouping and compact only from 10 000', () => {
    expect(nb(formatNumber(12480, 'fr'))).toBe('12 480')
    expect(nb(formatCompact(9999, 'fr'))).toBe('9 999')
    expect(nb(formatCompact(12500, 'fr'))).toBe('12,5 k')
    expect(formatNumber(12480, 'en')).toBe('12,480')
  })
  it('percent: 0 decimals unless < 10 %', () => {
    expect(nb(formatPercent(0.873, 'fr'))).toBe('87 %')
    expect(nb(formatPercent(0.042, 'fr'))).toBe('4,2 %')
    expect(formatPercent(0.873, 'en')).toBe('87%')
  })
})
describe('dates', () => {
  const now = new Date(2026, 10, 22, 15, 0)
  it('hour with 2 digits', () => { expect(formatHour(7)).toBe('07 h'); expect(formatHour(21)).toBe('21 h') })
  it('day: year only when different', () => {
    expect(formatDay(new Date(2026, 10, 22), 'fr', now)).toBe('22 nov.')
    expect(formatDay(new Date(2025, 10, 22), 'fr', now)).toBe('22 nov. 2025')
  })
  it('relative', () => {
    expect(formatRelative(new Date(2026, 10, 22, 14, 57), 'fr', now)).toBe('il y a 3 minutes')
  })
  it('weekdays start on Monday', () => {
    expect(weekdayLabels('fr')[0]).toBe('lun.')
    expect(weekdayLabels('fr')[6]).toBe('dim.')
  })
})
describe('formatDelta', () => {
  it('signs and direction', () => {
    const up = formatDelta(0.12, 'fr')
    expect(nb(up.text)).toBe('+12 %')
    expect(up.direction).toBe('up')
    expect(nb(formatDelta(-0.031, 'fr').text)).toBe('−3,1 %')
    expect(formatDelta(-0.031, 'fr').direction).toBe('down')
    expect(formatDelta(0, 'fr').direction).toBe('flat')
  })
})
describe('rounding edge cases', () => {
  it('decides decimals on the rounded value', () => {
    expect(nb(formatPercent(0.0999, 'fr'))).toBe('10 %')
    expect(nb(formatPercent(0.0949, 'fr'))).toBe('9,5 %')
  })
  it('a tiny ratio is flat, not +0,0 %', () => {
    expect(formatDelta(0.0001, 'fr').direction).toBe('flat')
    expect(formatDelta(-0.0001, 'fr').direction).toBe('flat')
  })
})

describe('formatBytes', () => {
  it('uses decimal units with one decimal under 100', () => {
    expect(formatBytes(18_200_000_000, 'fr')).toBe('18,2 Go')
    expect(formatBytes(4_000_000_000, 'fr')).toBe('4 Go')
    expect(formatBytes(412_400_000_000, 'fr')).toBe('412 Go')
    expect(formatBytes(1_234_000_000_000, 'en')).toBe('1.2 TB')
    expect(formatBytes(950, 'en')).toBe('950 B')
    expect(formatBytes(-5, 'en')).toBe('0 B')
  })
})
