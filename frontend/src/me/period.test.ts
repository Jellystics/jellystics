import { isPreset, localToday, meRange, todayIn } from './period'

describe('meRange', () => {
  it('ends tomorrow (exclusive) and covers n days', () => {
    expect(meRange('7d', '2026-11-24')).toEqual({ preset: '7d', from: '2026-11-18', to: '2026-11-25', compare: true })
    expect(meRange('30d', '2026-11-24').from).toBe('2026-10-26')
    expect(meRange('1y', '2026-11-24').from).toBe('2025-11-25')
  })
  it('has no previous period for all time', () => expect(meRange('all', '2026-11-24')).toEqual({ preset: 'all', from: '2000-01-01', to: '2026-11-25', compare: false }))
  it('validates presets', () => { expect(isPreset('90d')).toBe(true); expect(isPreset('mtd')).toBe(false); expect(isPreset(null)).toBe(false) })
  it('formats the local date', () => expect(localToday(new Date(2026, 0, 5, 23, 30))).toBe('2026-01-05'))
})

describe('todayIn', () => {
  // 2026-11-24 23:30 UTC: already the 25th in Paris, still the 24th in New York.
  const now = new Date(Date.UTC(2026, 10, 24, 23, 30))
  it('uses the server display zone', () => {
    expect(todayIn('Europe/Paris', now)).toBe('2026-11-25')
    expect(todayIn('America/New_York', now)).toBe('2026-11-24')
    expect(todayIn('UTC', now)).toBe('2026-11-24')
  })
  it('falls back to the browser date', () => {
    expect(todayIn(undefined, now)).toBe(localToday(now))
    expect(todayIn('Not/AZone', now)).toBe(localToday(now))
  })
})
