import { useCallback, useEffect, useSyncExternalStore } from 'react'

export type ThemeMode = 'light' | 'dark'
const KEY = 'jellystics-theme-mode'
const listeners = new Set<() => void>()

function read(): ThemeMode {
  try {
    const v = localStorage.getItem(KEY)
    if (v === 'light' || v === 'dark') return v
  } catch { /* storage blocked */ }
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

let current: ThemeMode | null = null
function get(): ThemeMode { return (current ??= read()) }
function setMode(m: ThemeMode) {
  current = m
  try { localStorage.setItem(KEY, m) } catch { /* ignore */ }
  document.documentElement.dataset.theme = m
  listeners.forEach((l) => l())
}

export function useThemeMode() {
  const mode = useSyncExternalStore((cb) => { listeners.add(cb); return () => listeners.delete(cb) }, get, get)
  useEffect(() => { document.documentElement.dataset.theme = mode }, [mode])
  const toggle = useCallback(() => setMode(get() === 'dark' ? 'light' : 'dark'), [])
  return { mode, toggle, set: setMode }
}

/** test helper */
export function __resetThemeForTests() { current = null }
