import { createContext, useContext, useEffect } from 'react'
import { useMatches, type UIMatch } from 'react-router-dom'

export type Crumb = { label: string; to?: string }
export type NavKey = 'overview' | 'live' | 'history' | 'libraries' | 'users' | 'insights' | 'discover' | 'playback' | 'settings'
export type RouteHandle = { crumb?: (match: UIMatch) => Crumb | null; nav?: NavKey }

export const CrumbLabelContext = createContext<{ label: string | null; set: (l: string | null) => void }>({ label: null, set: () => {} })

export function useCrumbs(): Crumb[] {
  const matches = useMatches()
  const { label } = useContext(CrumbLabelContext)
  const crumbs = matches
    .map((m) => (m.handle as RouteHandle | undefined)?.crumb?.(m) ?? null)
    .filter((c): c is Crumb => c !== null)
  if (label && crumbs.length) crumbs[crumbs.length - 1] = { ...crumbs[crumbs.length - 1], label }
  return crumbs
}

/** Override the last crumb label (entity names loaded asynchronously). */
export function useSetCrumbLabel(label: string | undefined | null) {
  const { set } = useContext(CrumbLabelContext)
  useEffect(() => {
    set(label || null)
    return () => set(null)
  }, [label, set])
}

export function useActiveNav(): NavKey | undefined {
  const matches = useMatches()
  for (let i = matches.length - 1; i >= 0; i--) {
    const nav = (matches[i].handle as RouteHandle | undefined)?.nav
    if (nav) return nav
  }
  return undefined
}
