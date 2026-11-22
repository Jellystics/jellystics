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

function subscribe(cb: () => void) {
  listeners.add(cb)
  const onStorage = (e: StorageEvent) => {
    if (e.key !== null && e.key !== KEY) return // null = storage.clear()
    current = null
    cb()
  }
  window.addEventListener('storage', onStorage)
  return () => { listeners.delete(cb); window.removeEventListener('storage', onStorage) }
}

export function useThemeMode() {
  const mode = useSyncExternalStore(subscribe, get, get)
  useEffect(() => { document.documentElement.dataset.theme = mode }, [mode])
  const toggle = useCallback(() => setMode(get() === 'dark' ? 'light' : 'dark'), [])
  return { mode, toggle, set: setMode }
}

/** test helper */
export function __resetThemeForTests() { current = null }
