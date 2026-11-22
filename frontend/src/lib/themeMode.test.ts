import { renderHook, act } from '@testing-library/react'
import { useThemeMode, __resetThemeForTests } from './themeMode'

describe('useThemeMode', () => {
  beforeEach(() => {
    localStorage.clear()
    delete document.documentElement.dataset.theme
    __resetThemeForTests()
  })

  it('defaults to the OS preference (light in jsdom) and stamps <html>', () => {
    const { result } = renderHook(() => useThemeMode())
    expect(result.current.mode).toBe('light')
    expect(document.documentElement.dataset.theme).toBe('light')
  })

  it('restores the legacy stored choice', () => {
    localStorage.setItem('jellystics-theme-mode', 'dark')
    const { result } = renderHook(() => useThemeMode())
    expect(result.current.mode).toBe('dark')
  })

  it('toggles and persists', () => {
    const { result } = renderHook(() => useThemeMode())
    act(() => result.current.toggle())
    expect(result.current.mode).toBe('dark')
    expect(localStorage.getItem('jellystics-theme-mode')).toBe('dark')
    expect(document.documentElement.dataset.theme).toBe('dark')
  })

  it('set() applies the mode and notifies every subscriber', () => {
    const a = renderHook(() => useThemeMode())
    const b = renderHook(() => useThemeMode())
    act(() => a.result.current.set('dark'))
    expect(a.result.current.mode).toBe('dark')
    expect(b.result.current.mode).toBe('dark')
  })

  it('follows changes made in another tab', () => {
    const { result } = renderHook(() => useThemeMode())
    expect(result.current.mode).toBe('light')
    act(() => {
      localStorage.setItem('jellystics-theme-mode', 'dark')
      window.dispatchEvent(new StorageEvent('storage', { key: 'jellystics-theme-mode', newValue: 'dark' }))
    })
    expect(result.current.mode).toBe('dark')
    expect(document.documentElement.dataset.theme).toBe('dark')
  })

  it('ignores storage events for other keys', () => {
    const { result } = renderHook(() => useThemeMode())
    act(() => {
      localStorage.setItem('jellystics-theme-mode', 'dark')
      window.dispatchEvent(new StorageEvent('storage', { key: 'other', newValue: 'x' }))
    })
    expect(result.current.mode).toBe('light')
  })
})
