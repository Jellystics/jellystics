import { describe, it, expect } from 'vitest'
describe('test runner', () => {
  it('runs with jsdom', () => {
    expect(document.createElement('div')).toBeInstanceOf(HTMLElement)
  })
})
