import { formatDuration, formatPosition, formatNumber, formatCompact, formatPercent, formatHour, formatDay, formatRelative, formatDelta, weekdayLabels, formatBytes, formatHoursAxis, formatAxisValue, formatWatchKpi } from './format'
import { niceMax, ticks } from '@/charts/scale'

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

describe('formatHoursAxis', () => {
  it.each([
    [0, '0 h'], [0.5, '30 min'], [1, '1 h'], [1.5, '1 h 30'], [2.4, '2 h 24'], [0.025, '1,5 min'], [1 / 60, '1 min'],
    [10, '10 h'], [12.5, '12,5 h'], [250, '250 h'], [0.005, '18 s'],
  ])('%f h → %s', (h, out) => expect(nb(formatHoursAxis(h, 'fr'))).toBe(out))

  it('never repeats adjacent axis tick labels', () => {
    for (const max of [0.004, 0.01, 0.02, 0.025, 0.05, 0.1, 0.2, 0.25, 0.5, 1, 1.2, 2, 2.4, 2.5, 3, 4, 5, 8, 10, 20, 25, 50, 100, 250, 1000]) {
      const labels = ticks(niceMax(max)).map((v) => formatHoursAxis(v, 'fr'))
      labels.slice(1).forEach((l, i) => expect(l, `max ${max}: ${labels.join(' | ')}`).not.toBe(labels[i]))
    }
  })
})

describe('formatAxisValue / formatWatchKpi', () => {
  it('keeps adjacent scatter ticks distinct for any nice maximum', () => {
    for (const p of [0.001, 0.01, 0.1, 1, 10, 100, 1000]) {
      for (const n of [1, 2, 2.5, 5, 10]) {
        const labels = ticks(n * p, 5).map((v) => formatAxisValue(v, 'fr'))
        expect(new Set(labels).size, labels.join(' | ')).toBe(labels.length)
      }
    }
  })
  it('reads short plays with decimals', () => expect(ticks(1, 5).map((v) => formatAxisValue(v, 'en'))).toEqual(['0', '0.2', '0.4', '0.6', '0.8', '1']))
  it('shows small watch totals as a duration, large ones as hours', () => {
    expect(formatWatchKpi(100, 'h', 'fr')).toEqual({ value: '2 min' })
    expect(formatWatchKpi(40, 'h', 'fr')).toEqual({ value: '40 s' })
    expect(formatWatchKpi(0, 'h', 'fr')).toEqual({ value: '0', unit: 'h' })
    expect(formatWatchKpi(7200, 'h', 'fr')).toEqual({ value: '2', unit: 'h' })
    expect(formatWatchKpi(3600 * 1234, 'h', 'en')).toEqual({ value: '1,234', unit: 'h' })
  })
})
