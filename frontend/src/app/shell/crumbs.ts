import { createContext, useContext, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useMatches, type UIMatch } from 'react-router-dom'

export type Crumb = { label: string; to?: string }
export type NavKey = 'overview' | 'live' | 'history' | 'libraries' | 'users' | 'insights' | 'discover' | 'playback' | 'settings'
export type RouteHandle = { /** Static trail (i18n keys as labels, translated by useCrumbs). */
  crumb?: (match: UIMatch) => Crumb | Crumb[] | null; nav?: NavKey
  /** false hides the global period picker (legacy pages with their own date controls). */
  period?: boolean }

/** Override of the last crumb label, or of the last crumbs (aligned on the end; null keeps a crumb). */
export type CrumbLabels = string | readonly (string | null)[] | null

export const CrumbLabelContext = createContext<{ label: CrumbLabels; set: (l: CrumbLabels) => void }>({ label: null, set: () => {} })

export function useCrumbs(): Crumb[] {
  const { t } = useTranslation()
  const matches = useMatches()
  const { label } = useContext(CrumbLabelContext)
  const crumbs = matches
    .flatMap((m) => {
      const c = (m.handle as RouteHandle | undefined)?.crumb?.(m) ?? []
      return Array.isArray(c) ? c : [c]
    })
    .map((c) => ({ ...c, label: t(c.label) }))
  const labels = label === null ? [] : typeof label === 'string' ? [label] : label
  labels.forEach((l, i) => {
    const at = crumbs.length - labels.length + i
    if (l && at >= 0) crumbs[at] = { ...crumbs[at], label: l }
  })
  return crumbs
}

/** Override the labels of the last crumbs, e.g. [user name, "Recap 2026"]. */
export function useSetCrumbLabels(labels: readonly (string | null | undefined)[] | null) {
  const { set } = useContext(CrumbLabelContext)
  const key = labels ? JSON.stringify(labels.map((l) => l || null)) : null
  useEffect(() => {
    set(key ? (JSON.parse(key) as (string | null)[]) : null)
    return () => set(null)
  }, [key, set])
}

/** Override the last crumb label (entity names loaded asynchronously). */
export function useSetCrumbLabel(label: string | undefined | null) {
  const { set } = useContext(CrumbLabelContext)
  useEffect(() => {
    set(label || null)
    return () => set(null)
  }, [label, set])
}

export function useShowPeriod(): boolean {
  const matches = useMatches()
  return (matches[matches.length - 1]?.handle as RouteHandle | undefined)?.period !== false
}

export function useActiveNav(): NavKey | undefined {
  const matches = useMatches()
  for (let i = matches.length - 1; i >= 0; i--) {
    const nav = (matches[i].handle as RouteHandle | undefined)?.nav
    if (nav) return nav
  }
  return undefined
}
