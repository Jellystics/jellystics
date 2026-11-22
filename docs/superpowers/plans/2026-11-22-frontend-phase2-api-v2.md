# Phase 2 — API v2 socle + typed frontend client

**Spec:** `docs/superpowers/specs/2026-11-22-frontend-redesign-design.md` (§7 pages, §8.1–8.2). **Visual reference:** `design/jellystics-design.html` (onglets Pages + Nouveautés show what each page displays).

**Goal:** a clean, tested `/api/v2` that gives the phase 3–4 pages everything they need, plus a typed React Query client in the frontend so pages never hand-build URLs.

## Global constraints
- Commits: one-line Conventional Commits, no attribution line. Tests green before each commit: `cd backend && POSTGRES_PORT=55432 go test ./...`; `cd frontend && pnpm test && pnpm exec tsc --noEmit && pnpm lint`.
- Reuse the existing SQL building blocks: `backend/internal/statsql` (`MediaFilter`, `StartedAt`, `LocalTS`, `AtLocal`, `CompletionRatio`, `Timezone`, `StartOfDay`) and `backend/internal/handler/stats_common.go` (`actsSelect`, `acts()` filter builder, `parseRange`). Definitions (do not reinvent): plays/views = `IsView`, clicks = all rows, watch time = `SUM(PlaybackDuration)` seconds, only `MediaType IN ('Movie','Episode','Audio')`, timestamps from `StartedAt` bucketed in `TZ`.
- All SQL parameterised. Errors: log + generic message (no `err.Error()` in responses), 400 on invalid params, 404 on unknown ids.
- New handlers in `backend/internal/handler/v2_*.go`, registered on the `authed` group under `/api/v2`. Keep `GET /api/v2/items/:id/route` (already exists).
- Raw numbers in responses (seconds, counts, ticks, ISO timestamps with offset, dates `YYYY-MM-DD`); formatting is the frontend's job. JSON field names camelCase.
- No migration in this phase (phase 6 owns migration 011).

## 1. Common query contract (`backend/internal/handler/v2_params.go`)
`parseV2(c) (V2Params, error)`: `from`, `to` (`YYYY-MM-DD`, interpreted as local midnight in `TZ`, `to` exclusive; default last 30 days ending tomorrow-midnight local; reject from ≥ to or impossible dates), `compare` (bool → also compute the previous period of equal length), `userId`, `libraryId`, `type` (`movie|series|music` → MediaType), `page`/`pageSize` (default 1/25, max 200), `sort`, `desc`, `q`. Table-driven tests (TZ=Europe/Paris around midnight and DST).

## 2. Endpoints (each with Go tests on seeded data)
- `GET /api/v2/overview` → `{ totals: {watchSeconds, views, clicks, activeUsers, completion}, previous?: same, daily: [{date, watchSeconds, views, clicks, byType:{movie,series,music}}], previousDaily?: [...], topItems: [{id, kind:'movie'|'series'|'album', name, views, watchSeconds, artId}], topUsers: [{id, name, watchSeconds, views}], byMethod: {direct, stream, transcode}, weekHour: number[7][24] (Monday first, views), concurrencyPeak: [{date, peak}], onThisDay: [{itemId, kind, name, users:[name], date}] }`.
- `GET /api/v2/history` → paged rows `{ id, startedAt, endedAt, user:{id,name}, item:{id, kind, name, parentId?, parentName?, season?, episode?}, mediaType, watchSeconds, completion (0..1|null), client, device, method, isView, pausedSeconds }` + `total`; filters `userId, type, client, device, method, views=1, q`; sort by `startedAt|watchSeconds|completion`. `GET /api/v2/history.csv` same filters, CSV stream. `GET /api/v2/history/calendar?month=YYYY-MM` → `[{date, views, watchSeconds, items:[{id,name,kind,user}] (max 6)}]`.
- `GET /api/v2/search?q=` → `{ items:[{id, kind, name, year?, parentName?}], users:[{id,name}] }` (max 8 each, ILIKE on names).
- `GET /api/v2/users` → `[{id, name, watchSeconds, views, byType:{movie,series,music}, lastActivity, trend:number[14]}]` in range.
- `GET /api/v2/users/:id` → `{ id, name, memberSince?, favoriteDevice?, totals, previous?, calendar:[{date, watchSeconds}] (last 365 days, independent of range), inProgress:[{seriesId, name, nextSeason, nextEpisode, lastPlayed, progress}], hours:number[24], profile:{regularity, discovery, concentration, completion}, genres:{movie:[{name,watchSeconds}], series:[...], music:[...]}, devices:[{device, client, version, views}], streak:{current, best} }`.
- `GET /api/v2/movies/:id` → `{ item:{id,name,year,runtimeSeconds,genres,sizeBytes,resolution?}, totals:{views, clicks, viewers, completedRatio, transcodeRatio, watchSeconds}, funnel:{clicks, views, half, completed}, stopPositions:number[24] (histogram over runtime), viewers:[{user, views, lastWatched, bestCompletion}], history: first 50 rows (same shape as /history) }`.
- `GET /api/v2/series/:id` → `{ item:{id,name,years,seasons, episodes, genres}, totals:{watchSeconds, views, clicks, viewers, retention, binges}, retention:[[viewersPerEpisode...] per season], seasons:[{number, episodes:[{id, number, name, views, viewers, avgCompletion, abandons}]}], viewers:[{user, episodesWatched, lastWatched}], cadence:[{bucket:'<1min'|'1-15min'|'15-60min'|'sameDay'|'nextDay'|'2-7d'|'>7d', count}], releaseLag:[{bucket:'<24h'|'1-7d'|'1-4w'|'1-6m'|'6-12m'|'>1y', count}] }`.
- `GET /api/v2/episodes/:id` → `{ item:{id, name, seriesId, seriesName, season, episode, runtimeSeconds, premiereDate?, prevId?, nextId?}, totals:{views, viewers, avgCompletion, pausedSecondsAvg}, funnel, stopPositions:number[20], viewers:[{user, startedAt, watchSeconds, completion, device}] }`.
- `GET /api/v2/artists/:id` → `{ item:{id,name,genres,albums,tracks}, totals:{listens, watchSeconds, listeners, skipRate}, topTracks:[{id,name,albumId,albumName,views,clicks,skipRate,listeners}], albums:[{id,name,year,tracks,listens}], skipByHour:number[24] }`.
- `GET /api/v2/albums/:id` → `{ item:{id,name,artistId,artistName,year,tracks,runtimeSeconds}, totals:{listens, fullListenRatio, fullAlbumListens, listeners, skipRate}, tracks:[{id, number, name, runtimeSeconds, views, clicks, skipRate, listeners}] }`.
- `GET /api/v2/tracks/:id` → `{ item:{id,name,albumId,artistId}, totals:{views, clicks, skipRate, listeners}, recent:[{user, startedAt, watchSeconds}] }` (side panel).
- `GET /api/v2/libraries` and `GET /api/v2/libraries/:id` → cards `{id,name,type:'movies'|'tvshows'|'music', items, sizeBytes, totals}`; detail adds `items:[{id, kind, name, views, viewers, watchSeconds, retention?, trend:number[14]}]` (paged), `genres`, `methods`, `unwatched:{count, list}`, `timeToWatch`.
All endpoints honour the common contract (range, userId, type, libraryId where meaningful; entity pages use the range for stats but return the catalog item regardless).

## 3. Frontend typed client (`frontend/src/api/`)
- `types.ts`: TypeScript interfaces mirroring every response above.
- `keys.ts`: query-key factory (`qk.overview(params)` …) including `from`, `to`, `compare` and filters.
- One hook per endpoint (`useOverview`, `useHistory`, `useHistoryCalendar`, `useSearch`, `useUsers`, `useUser`, `useMovie`, `useSeries`, `useEpisode`, `useArtist`, `useAlbum`, `useTrack`, `useLibraries`, `useLibrary`) built on `apiFetch` and `usePeriod()` (pages pass extra filters), with `placeholderData: keepPreviousData`.
- Vitest tests for the key factory and one hook with mocked fetch (asserts URL + params incl. period).
- Wire `CommandPalette` search results to `useSearch` (debounced 200 ms), results navigate through `AppLink` routes (`/movies/:id`, `/series/:id`, `/albums/:id`, `/users/:id`, track → `/albums/:albumId?track=`).

## 4. Commit plan
`feat: add v2 api params contract`, `feat: add v2 overview and history endpoints`, `feat: add v2 entity endpoints`, `feat: add v2 search users and libraries endpoints`, `feat: add typed api client`, `feat: wire command palette search`.
