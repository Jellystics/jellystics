import { Navigate, useLocation, useParams, useSearchParams, type RouteObject } from 'react-router-dom'
import { AppShell } from './shell/AppShell'
import type { Crumb, RouteHandle } from './shell/crumbs'
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
import SettingsPage from '@/features/settings/SettingsPage'
import AboutPage from '@/features/about/AboutPage'
import InsightsPage from './pages/insights/InsightsPage'
import DiscoverPage from './pages/discover/DiscoverPage'
import PlaybackPage from './pages/playback/PlaybackPage'
import RecapPage from './pages/recap/RecapPage'

/** Redirect keeping the current query string (and optionally overriding it). */
function To({ to, search }: { to: string; search?: string }) {
  const loc = useLocation()
  return <Navigate to={{ pathname: to, search: search ?? loc.search }} replace />
}

function TimelineRedirect() {
  const loc = useLocation()
  const sp = new URLSearchParams(loc.search); sp.set('view', 'calendar')
  return <Navigate to={{ pathname: '/history', search: `?${sp}` }} replace />
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
  if (!SETTINGS_SECTIONS.includes(section)) return <NotFoundPage />
  return <LegacyAt path="/settings" location={{ pathname: '/settings', search: `?${sp}` }}><SettingsPage /></LegacyAt>
}

type Match = Parameters<NonNullable<RouteHandle['crumb']>>[0]
const nav = (n: NonNullable<RouteHandle['nav']>, crumb?: RouteHandle['crumb']): { handle: RouteHandle } => ({ handle: { nav: n, crumb } })
/** Legacy pages that ship their own date controls: hide the global period picker. */
const ownPeriod = (n: NonNullable<RouteHandle['nav']>): { handle: RouteHandle } => ({ handle: { nav: n, period: false } })
const LIBS = { label: 'app.nav.libraries', to: '/libraries' }
const USERS = { label: 'app.nav.users', to: '/users' }
const trail = (...items: Crumb[]) => () => items
const SETTINGS_SECTIONS = ['config', 'tasks', 'backup', 'logs', 'webhooks', 'apikeys', 'about']
const settingsTrail = (m: Match): Crumb[] => [{ label: 'app.nav.settings', to: '/settings' }, ...(SETTINGS_SECTIONS.includes(m.params.section ?? 'config') ? [{ label: `app.crumb.settings.${m.params.section ?? 'config'}` }] : [])]
const legacy = (el: React.ReactElement) => <LegacyPage>{el}</LegacyPage>

export const routes: RouteObject[] = [
  { path: '/login', element: <LoginPage /> },
  { path: '/setup', element: <To to="/login" search="" /> },
  {
    element: <RequireAuth><AppShell /></RequireAuth>,
    children: [
      { path: '/', ...ownPeriod('overview'), element: legacy(<DashboardPage />) },
      { path: '/live', ...nav('live'), element: <ComingSoon nav="live" /> },
      { path: '/history', ...ownPeriod('history'), element: <History /> },
      { path: '/libraries', ...nav('libraries'), element: legacy(<LibrariesPage />) },
      { path: '/libraries/:id', ...nav('libraries', trail(LIBS, { label: 'app.crumb.library' })), element: legacy(<LibraryDetailPage />) },
      { path: '/movies/:id', ...nav('libraries', trail(LIBS, { label: 'app.crumb.movie' })), element: <ItemPage /> },
      { path: '/series/:id', ...nav('libraries', trail(LIBS, { label: 'app.crumb.series' })), element: <ItemPage /> },
      { path: '/series/:id/episodes/:episodeId', ...nav('libraries', (m) => [LIBS, { label: 'app.crumb.series', to: `/series/${m.params.id}` }, { label: 'app.crumb.episode' }]), element: <ItemPage /> },
      { path: '/artists/:id', ...nav('libraries', trail(LIBS, { label: 'app.crumb.artist' })), element: <ComingSoon nav="libraries" /> },
      { path: '/albums/:id', ...nav('libraries', trail(LIBS, { label: 'app.crumb.album' })), element: <AlbumPage /> },
      { path: '/users', ...nav('users'), element: legacy(<UsersPage />) },
      { path: '/users/:id', ...nav('users', trail(USERS, { label: 'app.crumb.user' })), element: legacy(<UserDetailPage />) },
      { path: '/users/:id/recap/:period', handle: { ...nav('users', (m) => [USERS, { label: 'app.crumb.user', to: `/users/${m.params.id}` }, { label: 'app.crumb.recap' }]).handle, period: false }, element: <RecapPage /> },
      { path: '/insights', ...nav('insights'), element: <InsightsPage /> },
      { path: '/discover', ...nav('discover'), element: <DiscoverPage /> },
      { path: '/playback', ...nav('playback'), element: <PlaybackPage /> },
      { path: '/settings', ...nav('settings'), element: <SettingsIndex /> },
      { path: '/settings/about', ...nav('settings', settingsTrail), element: legacy(<AboutPage />) },
      { path: '/settings/:section', ...nav('settings', settingsTrail), element: <SettingsSection /> },
      { path: '/items/:id', element: <ItemRedirect /> },
      { path: '/libraries/:lib/items/:id', element: <ItemRedirect /> },
      { path: '/libraries/:lib/albums/:id', element: <ParamTo build={(p) => `/albums/${p.id}`} /> },
      { path: '/statistics', element: <To to="/insights" /> },
      { path: '/activity', element: <To to="/history" /> },
      { path: '/activity/timeline', element: <TimelineRedirect /> },
      { path: '/about', element: <To to="/settings/about" /> },
      { path: '*', element: <NotFoundPage /> },
    ],
  },
]
