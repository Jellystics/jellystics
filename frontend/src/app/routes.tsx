import { Navigate, useLocation, useParams, useSearchParams, type RouteObject } from 'react-router-dom'
import { AppShell } from './shell/AppShell'
import type { RouteHandle } from './shell/crumbs'
import { RequireAuth } from './RequireAuth'
import { LegacyAt } from './legacy/LegacyAt'
import { LegacyPage } from './legacy/LegacyPage'
import LoginPage from './pages/LoginPage'
import { NotFoundPage } from './pages/NotFoundPage'
import { ItemRedirect } from './pages/ItemRedirect'
import { ComingSoon } from './pages/ComingSoon'
import DashboardPage from '@/features/dashboard/DashboardPage'
import ActivityPage from '@/features/activity/ActivityPage'
import TimelinePage from '@/features/activity/TimelinePage'
import LibrariesPage from '@/features/libraries/LibrariesPage'
import LibraryDetailPage from '@/features/libraries/LibraryDetailPage'
import ItemDetailPage from '@/features/libraries/ItemDetailPage'
import AlbumDetailPage from '@/features/libraries/AlbumDetailPage'
import UsersPage from '@/features/users/UsersPage'
import UserDetailPage from '@/features/users/UserDetailPage'
import StatisticsPage from '@/features/statistics/StatisticsPage'
import SettingsPage from '@/features/settings/SettingsPage'
import AboutPage from '@/features/about/AboutPage'

/** Redirect keeping the current query string (and optionally overriding it). */
function To({ to, search }: { to: string; search?: string }) {
  const loc = useLocation()
  return <Navigate to={{ pathname: to, search: search ?? loc.search }} replace />
}

function ParamTo({ build }: { build: (p: Record<string, string | undefined>) => string }) {
  const p = useParams()
  return <To to={build(p)} />
}

function History() {
  const [sp] = useSearchParams()
  return <LegacyPage>{sp.get('view') === 'calendar' ? <TimelinePage /> : <ActivityPage />}</LegacyPage>
}

function SettingsIndex() {
  const [sp] = useSearchParams()
  const view = sp.get('view')
  const rest = new URLSearchParams(sp); rest.delete('view')
  const search = rest.toString()
  return <To to={`/settings/${view || 'config'}`} search={search ? `?${search}` : ''} />
}

function ItemPage() {
  const { id, episodeId } = useParams()
  const itemId = episodeId ?? id ?? ''
  return <LegacyAt path="/items/:itemId" location={{ pathname: `/items/${itemId}` }}><ItemDetailPage /></LegacyAt>
}

function AlbumPage() {
  const { id = '' } = useParams()
  return <LegacyAt path="/libraries/_/albums/:albumId" location={{ pathname: `/libraries/_/albums/${id}` }}><AlbumDetailPage /></LegacyAt>
}

function SettingsSection() {
  const { section = 'config' } = useParams()
  const sp = new URLSearchParams(useLocation().search); sp.set('view', section)
  return <LegacyAt path="/settings" location={{ pathname: '/settings', search: `?${sp}` }}><SettingsPage /></LegacyAt>
}

const nav = (n: NonNullable<RouteHandle['nav']>): { handle: RouteHandle } => ({ handle: { nav: n } })
const legacy = (el: React.ReactElement) => <LegacyPage>{el}</LegacyPage>

export const routes: RouteObject[] = [
  { path: '/login', element: <LoginPage /> },
  {
    element: <RequireAuth><AppShell /></RequireAuth>,
    children: [
      { path: '/', ...nav('overview'), element: legacy(<DashboardPage />) },
      { path: '/live', ...nav('live'), element: <ComingSoon nav="live" /> },
      { path: '/history', ...nav('history'), element: <History /> },
      { path: '/libraries', ...nav('libraries'), element: legacy(<LibrariesPage />) },
      { path: '/libraries/:id', ...nav('libraries'), element: legacy(<LibraryDetailPage />) },
      { path: '/movies/:id', ...nav('libraries'), element: <ItemPage /> },
      { path: '/series/:id', ...nav('libraries'), element: <ItemPage /> },
      { path: '/series/:id/episodes/:episodeId', ...nav('libraries'), element: <ItemPage /> },
      { path: '/artists/:id', ...nav('libraries'), element: <ComingSoon nav="libraries" /> },
      { path: '/albums/:id', ...nav('libraries'), element: <AlbumPage /> },
      { path: '/users', ...nav('users'), element: legacy(<UsersPage />) },
      { path: '/users/:id', ...nav('users'), element: legacy(<UserDetailPage />) },
      { path: '/users/:id/recap/:period', ...nav('users'), element: <ComingSoon nav="users" /> },
      { path: '/insights', ...nav('insights'), element: legacy(<StatisticsPage />) },
      { path: '/discover', ...nav('discover'), element: <ComingSoon nav="discover" /> },
      { path: '/playback', ...nav('playback'), element: <ComingSoon nav="playback" /> },
      { path: '/settings', ...nav('settings'), element: <SettingsIndex /> },
      { path: '/settings/about', ...nav('settings'), element: legacy(<AboutPage />) },
      { path: '/settings/:section', ...nav('settings'), element: <SettingsSection /> },
      { path: '/items/:id', element: <ItemRedirect /> },
      { path: '/libraries/:lib/items/:id', element: <ItemRedirect /> },
      { path: '/libraries/:lib/albums/:id', element: <ParamTo build={(p) => `/albums/${p.id}`} /> },
      { path: '/statistics', element: <To to="/insights" /> },
      { path: '/activity', element: <To to="/history" /> },
      { path: '/activity/timeline', element: <To to="/history" search="?view=calendar" /> },
      { path: '/about', element: <To to="/settings/about" /> },
      { path: '/setup', element: <To to="/login" search="" /> },
      { path: '*', element: <NotFoundPage /> },
    ],
  },
]
