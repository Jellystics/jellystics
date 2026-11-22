# Phases 3–5 — Pages

**Spec:** `docs/superpowers/specs/2026-11-22-frontend-redesign-design.md` (§5 components, §6 routes, §7 pages, §9 bugs, §10 a11y). **Visual source of truth:** `design/jellystics-design.html` — tab « Pages » (one mock per page) and tab « Nouveautés » (NEW widgets, each card names its destination page). Rebuild each page to match its mock in light and dark, 390–1440 px.

## Global constraints (all phases)
- Commits: one-line Conventional Commits, no attribution line. Before each commit: `cd frontend && pnpm test && pnpm exec tsc --noEmit && pnpm lint`; backend changes: `cd backend && POSTGRES_PORT=55432 go test ./...`. Before finishing a phase also `pnpm build && pnpm test:e2e`.
- Use only the kit: `@/ui` (components, AppLink, ButtonLink), `@/charts`, `@/lib` (usePeriod, useSearchParam, useFormat, formatters), `@/api` (typed React Query hooks from phase 2). No MUI/recharts/axios (ESLint enforces). Pages live in `frontend/src/app/pages/<page>/` (page component + small local components + `*.test.tsx`).
- Every page: `PageHeader` or `EntityHeader`; data via `@/api` hooks keyed on the period; loading = Skeleton shaped like the content (keep previous data on refetch); empty = `EmptyState`; error = `ErrorState` with retry; every entity/user name is an `AppLink`; charts wrapped in `ChartFrame` (table view + CSV); breadcrumbs: set the dynamic label with `useSetCrumbLabel(name)`.
- i18n: every string through `t()` with `app.*` keys added to fr/en/es.
- When a page is rewritten, replace its `LegacyPage` mount in `frontend/src/app/routes.tsx`, remove the route's `period: false` flag if present (the top-bar PeriodPicker becomes the only time control), and add the route to `frontend/e2e/smoke.spec.ts` (plus one dynamic-id route per entity page with realistic mock payloads in `e2e/mockApi.ts`).
- If an endpoint field a widget needs is missing from `/api/v2`, add it to the backend handler (with a Go test) and to `frontend/src/api/types.ts` in the same phase.
- Page tests (Vitest, mocked `apiFetch` or React Query cache): renders key figures, links point to the right routes and keep the period, empty and error states.

## Phase 3 — core pages
1. **Vue d'ensemble** `/` (mock « Vue d'ensemble » + NEW « Pic de flux simultanés », « Ce jour-là »): StatStrip selectable (time/views/clicks drive the main LineChart with previous period when `compare`; `?metric=` in URL), Top contenus BarList with type SegmentedControl, En direct (2 SessionCards, link `/live`), Utilisateurs BarList, Par type + méthode SegBars (method uses `--m-*` ramp), WeekHeatmap, concurrency Columns, On this day list.
2. **En direct** `/live` (mock « En direct » + NEW frise, débit): live data from the socket `sessions` event (`useSocketEvent`) with `/sessions/current` as initial fetch; StatStrip (lectures, transcodages, débit if available, pic du jour), SessionCard grid, simultaneous plays today (Columns), SessionTimeline for today.
3. **Historique** `/history` (mock « Historique »): FilterBar chips bound to URL params, DataTable with server pagination/sort (`useHistory`), CSV export link, `?view=calendar` month calendar (`useHistoryCalendar`, `?month=`). Replaces legacy ActivityPage and TimelinePage.
4. **Utilisateurs** `/users` (mock « Utilisateurs » + NEW rétention, part de chacun): DataTable with SegBar thin by type, trend Sparkline; retention Matrix and share small multiples below.
5. **Utilisateur** `/users/:id` (mock « Utilisateur » + NEW comparaison, appareils, VO/VF, série & records): tabs `overview|history|taste` (`?tab=`), StatStrip, CalendarHeatmap, En cours shelf, hour Columns (horloge), profile meters, devices table, compare panel (`?compare=:otherId`, uses `useUser` twice), StreakCard; history tab reuses the history table filtered by user; taste tab = genres per type (BarList).

## Phase 4 — entity pages
1. **Bibliothèques** `/libraries` and **Bibliothèque** `/libraries/:libraryId` (mock « Bibliothèque » + NEW décennies, croissance): cards; tabs `items|stats|activity`; items DataTable (grid/table layout toggle, Sparkline, retention meter for series), stats (genres, methods, unwatched, time to watch, decades, growth vs consumption), activity (history filtered by library).
2. **Film** `/movies/:id` (mock « Film » + NEW séances pour finir): EntityHeader, StatStrip, Funnel, StopHistogram, viewers table, history.
3. **Série** `/series/:id` (mock « Série » + NEW cadence, délai sortie→vue): EntityHeader, StatStrip, RetentionChart (click → episode page), season SegmentedControl (`?season=`), episodes DataTable linking to `/series/:id/episodes/:eid`, viewers tab, history tab (episode names shown), cadence and release-lag Columns.
4. **Épisode** `/series/:id/episodes/:episodeId` (mock « Épisode »): breadcrumb Série › Saison › Épisode, prev/next ButtonLinks, StatStrip, Funnel, StopHistogram, viewers table.
5. **Artiste** `/artists/:id` (mock « Artiste » + NEW skip selon l'heure): StatStrip incl. skip rate, top tracks table (views, clicks, skip Meter, listeners), albums grid, skip-by-hour Columns.
6. **Album** `/albums/:id` (mock « Album » + NEW albums en entier): tracks table with skip; clicking a row opens a `SidePanel` (new ui component, Radix Dialog as right drawer) bound to `?track=` showing `useTrack` data.
Remove the `ItemPage`/`AlbumPage` legacy adapters once replaced; delete `ComingSoon` usage for artists.

## Phase 5 — analyses + NEW views
Add v2 endpoints (backend, tested) under `/api/v2/insights/*`, `/api/v2/discover/*`, `/api/v2/playback/*`, `/api/v2/recap/:userId/:period`, then the pages:
1. **Insights** `/insights` (mock « Insights » + NEW durée des séances, reprises vs début, revisionnages, genres dans le temps, moments par type): completion distribution, completion by type, binges, pauses, week rhythm, session length histogram, resume share, rewatch list, genres StackedArea, SmallMultiples by type. Replaces legacy StatisticsPage (remove `period:false`).
2. **Découverte** `/discover` (mock « Découverte » + NEW stockage vs visionnage): never watched, abandoned, added this month, Scatter size vs views.
3. **Lecture & transcodage** `/playback` (mock « Lecture & transcodage » + NEW faux départs, résolution, HDR, raisons, matrice codecs, vidéo/audio, local/distant + lieux, débit): uses phase 6 columns (`TranscodeReasons`, `Bitrate`, `HwAccel`, `VideoRange`, `AudioLanguage`, `SubtitleLanguage`, `IsRemote`, `PauseCount`) and `ip_geo_cache`. Show an `EmptyState` « Données collectées à partir du <date> » when the new columns have no data yet.
4. **Récap** `/users/:id/recap/:period` (mock « Récap » + NEW découverte d'artistes, classement avec évolution): hero figure, tops per type, by month Columns, records, artist discovery, ranked list with delta vs previous period; period switch 2026 / months.
5. **VO/VF & sous-titres** on the Utilisateur page (phase 3 placeholder → real data from `AudioLanguage`/`SubtitleLanguage`).
