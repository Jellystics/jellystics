# Phase 1 — suivis différés

Points mineurs relevés pendant les relectures de l'étape 1, à traiter dans les étapes suivantes (la relecture finale a jugé qu'ils ne bloquent pas).

- Task 1: minor (deferred): very new resolved versions (lucide-react 1.77, jest-dom 8, jsdom 31) — evidence limited to tsc + smoke test
- Task 1: minor (deferred): src/test/setup.ts dense one-liners (verbatim from plan)
- Task 2: minor (deferred): themeMode cache ignores cross-tab storage events; no test for set()/multi-subscriber
- Task 2: minor (deferred): legacy setThemeMode and new hook keep separate caches (resolved when legacy is removed in phase 7)
- Task 5: minor (deferred): apiFetch returns raw text on non-JSON 2xx; setSession not wrapped in try/catch
- Task 3: minor (deferred): formatPercent threshold on raw ratio (0.0999 → "10,0 %"); formatDelta tiny ratio → "+0,0 %" up; useFormat returns new closures each render; limited en coverage
- Task 4: minor (deferred): period valid() accepts impossible dates (2026-02-31 rolls over) — fix: toStr(toDate(s)) === s; todayIn not refreshed past midnight; missing tests for push option / previousPeriod(all|custom)
- Task 6: minor (deferred): iconOnly aria-label not type-enforced; SegBar part spans need role; Chip remove label lacks chip name; ErrorState no role=alert; .lg.iconOnly width; Art hash simple char sum
- Task 7: minor (deferred): toast test act() warning on stderr (wrap in act); toast pause only on hover not focus, errors not role=alert; custom-range popover doesn't restore focus to trigger; "vs" label nonsensical for preset all; ConfirmDialog onConfirm rejection unhandled; Field hint/error not aria-describedby; Checkbox imports Switch.module.css; #fff literal in Switch; Tooltip provider per instance; thin coverage (auto-dismiss, cap, invalid range)
- Task 8: minor (deferred): StatTile global unhashed classes (delta good bad flat) beside module classes; sticky th ineffective inside overflow wrapper; no tests for table loading/empty/pagination bounds
- Task 9: minor (deferred): Columns uses i % xEvery (not showXLabel); hit rects have aria-label without role; CSV doesn't escape \r; tooltip flip uses fixed 160px width estimate; no chart-level test of locale hook path; toCsv import mid-file in test
- Task 10: minor (deferred): chart hit rects with onClick lack Enter/Space handlers (RetentionChart, Columns); Matrix text < 11px when container narrower than natural width; fixed label widths in Funnel/SessionTimeline; Funnel % label x uses 7px/char estimate
- Task 11: minor (deferred): command palette listbox lacks combobox/aria-activedescendant; no skip link; mobile drawer stays open past 900px resize; @/ui imports @/app/AppLink (layering inversion); TaskToast ×6 instances; legacy shell also connects socket (never mounted together); palette reset runs post-paint
- Task 12: minor (deferred): LegacyAt uses UNSAFE_RouteContext (tested; revisit when pages are rewritten); AlbumPage drops ?track=; legacy Settings setSearchParams takes an extra redirect hop; setup.ts global Request wrapper drops AbortSignal for all tests; RequireAuth unconfigured branch untested; v2_items test not through auth middleware; 500 body returns err.Error()
- Task 13: minor (deferred): e2e mock catch-all returns 200 [] for unknown endpoints; smoke list has no dynamic/entity routes; react-hooks v7 compiler rules disabled (flag CommandPalette, RetentionChart, DataTable); legacy /movies/:id throws on [] mock (mock artefact)

## Tests à ajouter en phase 3
- usePeriod avec push, plages personnalisées, retour d'une plage perso vers un preset
- previousPeriod pour « all » et les plages personnalisées
- DataTable : chargement, vide, bornes de pagination
- RequireAuth quand le serveur n'est pas configuré
- un test e2e par route dynamique (/movies/:id, /series/:id/episodes/:eid, redirection /items/:id)
