import { renderHook, act } from '@testing-library/react'
import { useThemeMode, __resetThemeForTests } from './theme'

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
})
