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
import LibrariesPage from './pages/libraries/LibrariesPage'
import LibraryPage from './pages/library/LibraryPage'
import MoviePage from './pages/movie/MoviePage'
import SeriesPage from './pages/series/SeriesPage'
import EpisodePage from './pages/episode/EpisodePage'
import ArtistPage from './pages/artist/ArtistPage'
import AlbumPage from './pages/album/AlbumPage'
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
      { path: '/libraries', ...nav('libraries'), element: <LibrariesPage /> },
      { path: '/libraries/:id', ...nav('libraries', trail(LIBS, { label: 'app.crumb.library' })), element: <LibraryPage /> },
      { path: '/movies/:id', ...nav('libraries', trail(LIBS, { label: 'app.crumb.movie' })), element: <MoviePage /> },
      { path: '/series/:id', ...nav('libraries', trail(LIBS, { label: 'app.crumb.series' })), element: <SeriesPage /> },
      { path: '/series/:id/episodes/:episodeId', ...nav('libraries', (m) => [LIBS, { label: 'app.crumb.series', to: `/series/${m.params.id}` }, { label: 'app.crumb.episode' }]), element: <EpisodePage /> },
      { path: '/artists/:id', ...nav('libraries', trail(LIBS, { label: 'app.crumb.artist' })), element: <ArtistPage /> },
      { path: '/albums/:id', ...nav('libraries', trail(LIBS, { label: 'app.crumb.album' })), element: <AlbumPage /> },
      { path: '/users', ...nav('users'), element: legacy(<UsersPage />) },
      { path: '/users/:id', ...nav('users', trail(USERS, { label: 'app.crumb.user' })), element: legacy(<UserDetailPage />) },
      { path: '/users/:id/recap/:period', ...nav('users', (m) => [USERS, { label: 'app.crumb.user', to: `/users/${m.params.id}` }, { label: 'app.crumb.recap' }]), element: <ComingSoon nav="users" /> },
      { path: '/insights', ...ownPeriod('insights'), element: legacy(<StatisticsPage />) },
      { path: '/discover', ...nav('discover'), element: <ComingSoon nav="discover" /> },
      { path: '/playback', ...nav('playback'), element: <ComingSoon nav="playback" /> },
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
