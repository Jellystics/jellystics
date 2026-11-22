import { niceMax, ticks, barPath } from './scale'
it('niceMax', () => { expect([0.7, 1, 1.4, 2.2, 3, 7, 12, 260].map(niceMax)).toEqual([1, 1, 2, 2.5, 5, 10, 20, 500]) })
it('ticks are clean', () => { expect(ticks(10)).toEqual([0, 2, 4, 6, 8, 10]); expect(ticks(2.5)).toEqual([0, 0.5, 1, 1.5, 2, 2.5]) })
it('barPath has a square baseline and rounded top', () => {
  const d = barPath(0, 10, 20, 90)
  expect(d.startsWith('M0,100V14')).toBe(true) // starts at bottom-left, square corner
  expect(d).toContain('Q0,10 4,10')
  expect(barPath(0, 0, 20, 0)).toBe('')
})
