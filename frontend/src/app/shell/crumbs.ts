import { createContext, useContext, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useMatches, type UIMatch } from 'react-router-dom'

/** `raw` labels are shown as-is (entity names), others are i18n keys. */
export type Crumb = { label: string; to?: string; raw?: boolean }
/**
 * Named values pages publish for the crumb builders of their route (entity
 * names loaded asynchronously, e.g. `{ series: 'Station Nord', season: 'Saison 2' }`).
 */
export type CrumbLabels = Record<string, string | undefined>
export type NavKey = 'overview' | 'live' | 'history' | 'libraries' | 'users' | 'insights' | 'discover' | 'playback' | 'settings'
export type RouteHandle = { /** Static trail (i18n keys as labels, translated by useCrumbs), refined by the page's labels. */
  crumb?: (match: UIMatch, labels: CrumbLabels) => Crumb | Crumb[] | null; nav?: NavKey
  /** false hides the global period picker (legacy pages with their own date controls). */
  period?: boolean }

const NO_LABELS: CrumbLabels = {}

export const CrumbLabelContext = createContext<{
  label: string | null; set: (l: string | null) => void
  labels?: CrumbLabels; setLabels?: (l: CrumbLabels) => void
}>({ label: null, set: () => {} })

export function useCrumbs(): Crumb[] {
  const { t } = useTranslation()
  const matches = useMatches()
  const { label, labels = NO_LABELS } = useContext(CrumbLabelContext)
  const crumbs = matches
    .flatMap((m) => {
      const c = (m.handle as RouteHandle | undefined)?.crumb?.(m, labels) ?? []
      return Array.isArray(c) ? c : [c]
    })
    .map(({ raw, ...c }) => ({ ...c, label: raw ? c.label : t(c.label) }))
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

/** Publish named labels read by the route's crumb builder (intermediate crumbs). */
export function useSetCrumbLabels(labels: CrumbLabels | undefined | null) {
  const { setLabels } = useContext(CrumbLabelContext)
  const key = JSON.stringify(labels ?? {})
  useEffect(() => {
    if (!setLabels) return
    setLabels(JSON.parse(key) as CrumbLabels)
    return () => setLabels(NO_LABELS)
  }, [key, setLabels])
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
