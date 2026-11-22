import type { ReactNode } from 'react'
import { Route, Routes, UNSAFE_RouteContext } from 'react-router-dom'
import { LegacyPage } from './LegacyPage'

/**
 * Renders a legacy page inside a nested <Routes location> so its useParams()/useSearchParams()
 * see the legacy URL shape while the browser URL stays the new one. The parent route context is
 * reset so the nested <Routes> matches from the root (parent paths need no trailing "/*").
 */
export function LegacyAt({ path, location, children }: { path: string; location: { pathname: string; search?: string }; children: ReactNode }) {
  return (
    <UNSAFE_RouteContext.Provider value={{ outlet: null, matches: [], isDataRoute: false }}>
      <Routes location={{ pathname: location.pathname, search: location.search ?? '' }}>
        <Route path={path} element={<LegacyPage>{children}</LegacyPage>} />
      </Routes>
    </UNSAFE_RouteContext.Provider>
  )
}
