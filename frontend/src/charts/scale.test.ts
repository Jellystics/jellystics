import { niceMax, ticks, barPath } from './scale'
it('niceMax', () => { expect([0.7, 1, 1.4, 2.2, 3, 7, 12, 260].map(niceMax)).toEqual([1, 1, 2, 2.5, 5, 10, 20, 500]) })
it('ticks are clean', () => { expect(ticks(10)).toEqual([0, 2, 4, 6, 8, 10]); expect(ticks(2.5)).toEqual([0, 0.5, 1, 1.5, 2, 2.5]) })
it('barPath has a square baseline and rounded top', () => {
  const d = barPath(0, 10, 20, 90)
  expect(d.startsWith('M0,100V14')).toBe(true) // starts at bottom-left, square corner
  expect(d).toContain('Q0,10 4,10')
  expect(barPath(0, 0, 20, 0)).toBe('')
})
import { defaultFormat } from './scale'
it('defaultFormat follows the given locale', () => {
  expect(defaultFormat(1500, 'en')).toBe('1.5K')
  expect(defaultFormat(1500, 'fr')).not.toBe('1.5K')
  expect(defaultFormat(0.5, 'en')).toBe('0.5')
})

it('clamp01 keeps out-of-range and non-finite ratios inside the plot', async () => {
  const { clamp01 } = await import('./scale')
  expect(clamp01(-2147483648 / 10)).toBe(0)
  expect(clamp01(3)).toBe(1)
  expect(clamp01(0.4)).toBe(0.4)
  expect(clamp01(Number.NaN)).toBe(0)
})
