# Frontend redesign — Phase 1 « Fondations » Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Put in place the new frontend foundation (tokens, UI kit, chart kit, formatting, URL state, API client, app shell, routing, login) so that every later phase only has to build pages, while the not-yet-rewritten pages keep working inside the new shell.

**Architecture:** New code lives in `frontend/src/{styles,lib,ui,charts,app}`; it never imports MUI. The existing pages under `frontend/src/features/**` are mounted unchanged inside the new shell through a `LegacyPage` wrapper that provides the MUI theme/providers they need (MUI is removed in phase 7). Visual source of truth: `design/jellystics-design.html` (CSS in its `<style>` block, chart algorithms in its `<script>` block).

**Tech Stack:** React 19, Vite, TypeScript strict, React Router 7, @tanstack/react-query 5, @tanstack/react-table 8, Radix UI primitives, lucide-react, date-fns + date-fns-tz, i18next, socket.io-client, Vitest + Testing Library + jsdom, Playwright.

**Spec:** `docs/superpowers/specs/2026-11-22-frontend-redesign-design.md`

## Global Constraints

- Branch `feat/frontend-redesign`. Commits: one-line Conventional Commits (`feat: …`, `fix: …`, `test: …`, `chore: …`), **no** `Co-Authored-By` or any attribution line.
- Package manager: `pnpm` (run from `frontend/`). Never `npm install`.
- New code (`src/styles`, `src/lib`, `src/ui`, `src/charts`, `src/app`) must not import `@mui/*`, `@emotion/*`, `recharts`, `@fluentui/*`, `@tabler/*`, `axios`, `notistack`.
- Radii: only `var(--r-sm)` 4px, `var(--r-md)` 6px, `var(--r-lg)` 8px, `var(--r-full)`. No other radius value anywhere in new CSS.
- Font sizes: only the `--fs-*` tokens (11/12/13/14/16/20/24/48 px). Nothing under 11 px.
- Status colors (`--good/--warning/--serious/--critical`) only on badge icons/dots and alerts; badges have neutral background and ink text. Play-method charts use `--m-direct/--m-stream/--m-transcode`.
- Every link is a real `<a>` (React Router `<Link>`/`AppLink`) and preserves the global period params `range`, `from`, `to`, `compare`.
- Dates sent to the API are local `YYYY-MM-DD` strings computed in the server timezone; never `toISOString()`.
- All user-visible strings go through i18next `t()` with keys under the new `app.` namespace prefix, present in `public/locales/{fr,en,es}/translation.json`. No inline default values.
- Tests must pass: `pnpm test` (Vitest), `pnpm exec tsc --noEmit`, `pnpm lint`, and for backend changes `cd backend && POSTGRES_PORT=55432 go test ./...` (a Postgres for tests runs in container `jellystics-testdb` on port 55432).

---

## File structure (created in this phase)

```
frontend/
  vitest.config.ts                 test runner config (jsdom, setup file, @ alias)
  playwright.config.ts             e2e config (vite preview on :4173, mocked API)
  e2e/smoke.spec.ts                every route renders, no console error, no h-overflow, light+dark, 390/1360
  e2e/mockApi.ts                   page.route() handlers returning fixtures
  src/test/setup.ts                jest-dom matchers, matchMedia polyfill
  src/styles/tokens.css            design tokens (light/dark)
  src/styles/base.css              reset, typography, focus ring, scrollbars, utilities (.num, .muted, .sr-only)
  src/lib/theme.ts                 useThemeMode(): 'light'|'dark', toggle, persists, sets <html data-theme>
  src/lib/format.ts                all value formatters (durations, numbers, dates, hours, deltas)
  src/lib/period.ts                period model: parse/serialize ?range|from|to, previous period
  src/lib/useUrlState.ts           typed read/write of search params; usePeriod(); useTab()
  src/lib/auth.ts                  token storage, deviceId, logout
  src/lib/api.ts                   apiFetch<T>(), ApiError, 401 → /login
  src/lib/queryClient.ts           React Query client + defaults
  src/lib/serverConfig.ts          useServerConfig(): { configured, jellyfinUrl, timezone }
  src/lib/socket.ts                socket.io client (moved), useSocketEvent()
  src/lib/i18n.ts                  (moved from old lib/i18n.ts, unchanged behaviour) + document lang sync
  src/ui/*.tsx + *.module.css      UI kit (see tasks 6–8)
  src/ui/index.ts                  barrel export
  src/charts/*.tsx                 chart kit (see tasks 9–10)
  src/charts/index.ts              barrel export
  src/app/App.tsx                  providers (QueryClient, Toaster, theme), RouterProvider
  src/app/routes.tsx               route tree, redirects, 404, legacy mounting
  src/app/AppLink.tsx              <Link> that preserves period params
  src/app/shell/AppShell.tsx       sidebar + top bar + outlet
  src/app/shell/Sidebar.tsx, TopBar.tsx, UserMenu.tsx, MobileNav.tsx, CommandPalette.tsx
  src/app/shell/crumbs.ts          route handle → breadcrumb resolution
  src/app/legacy/LegacyPage.tsx    MUI ThemeProvider + LocalizationProvider + SnackbarProvider + DateRangeProvider wrapper
  src/app/pages/NotFoundPage.tsx
  src/app/pages/LoginPage.tsx      new login + setup wizard
  src/main.tsx                     renders src/app/App
backend/internal/handler/frontend_auth.go   IsConfigured returns "timezone"
```

`frontend/src/App.tsx`, `src/lib/router.tsx`, `src/shared/components/Layout/*`, `src/features/auth/*` stop being used at the end of this phase (deleted in Task 13). Legacy pages under `src/features/{dashboard,activity,libraries,users,statistics,settings,about}` stay and are mounted via `LegacyPage`.

---

### Task 1: Tooling — dependencies, Vitest, Playwright skeleton

**Files:**
- Modify: `frontend/package.json` (deps + scripts)
- Create: `frontend/vitest.config.ts`, `frontend/src/test/setup.ts`, `frontend/src/test/smoke.test.ts`
- Modify: `frontend/tsconfig.json` (`"types": ["node", "vitest/globals", "@testing-library/jest-dom"]`)

**Interfaces:**
- Produces: `pnpm test` (Vitest, jsdom, globals), `pnpm test:e2e` (Playwright), alias `@` → `src` in tests.

- [ ] **Step 1: Add dependencies** (run in `frontend/`)

```bash
pnpm add @tanstack/react-query@^5 lucide-react @radix-ui/react-dialog @radix-ui/react-dropdown-menu @radix-ui/react-popover @radix-ui/react-tooltip @radix-ui/react-tabs @radix-ui/react-toggle-group @radix-ui/react-switch @radix-ui/react-checkbox @radix-ui/react-select @radix-ui/react-visually-hidden date-fns-tz
pnpm add -D vitest@^3 @vitest/coverage-v8@^3 jsdom @testing-library/react @testing-library/jest-dom @testing-library/user-event @playwright/test
```

- [ ] **Step 2: Add scripts to `package.json`**

```json
"test": "vitest run",
"test:watch": "vitest",
"test:e2e": "playwright test"
```

- [ ] **Step 3: Create `vitest.config.ts`**

```ts
import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react-swc'
import path from 'path'

export default defineConfig({
  plugins: [react()],
  resolve: { alias: { '@': path.resolve(__dirname, './src') } },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    include: ['src/**/*.test.{ts,tsx}'],
    css: { modules: { classNameStrategy: 'non-scoped' } },
  },
})
```

- [ ] **Step 4: Create `src/test/setup.ts`**

```ts
import '@testing-library/jest-dom/vitest'

// jsdom has no matchMedia
if (!window.matchMedia) {
  window.matchMedia = (query: string) =>
    ({ matches: false, media: query, onchange: null, addEventListener() {}, removeEventListener() {}, addListener() {}, removeListener() {}, dispatchEvent: () => false }) as unknown as MediaQueryList
}
// Radix needs ResizeObserver / pointer capture in jsdom
class RO { observe() {} unobserve() {} disconnect() {} }
;(globalThis as unknown as { ResizeObserver: typeof RO }).ResizeObserver ??= RO
Element.prototype.hasPointerCapture ??= () => false
Element.prototype.scrollIntoView ??= () => {}
```

- [ ] **Step 5: Write a smoke test `src/test/smoke.test.ts`**

```ts
import { describe, it, expect } from 'vitest'
describe('test runner', () => {
  it('runs with jsdom', () => {
    expect(document.createElement('div')).toBeInstanceOf(HTMLElement)
  })
})
```

- [ ] **Step 6: Run** `pnpm test` → Expected: 1 passed. Run `pnpm exec tsc --noEmit` → no error.

- [ ] **Step 7: Commit**

```bash
git add frontend/package.json frontend/pnpm-lock.yaml frontend/vitest.config.ts frontend/src/test frontend/tsconfig.json
git commit -m "chore: add vitest, playwright and new frontend dependencies"
```

---

### Task 2: Design tokens, base styles, theme mode

**Files:**
- Create: `frontend/src/styles/tokens.css`, `frontend/src/styles/base.css`, `frontend/src/lib/theme.ts`, `frontend/src/lib/theme.test.ts`

**Interfaces:**
- Produces: CSS custom properties (exact names from `design/jellystics-design.html` `:root` block, lines 12–116), `useThemeMode(): { mode: 'light'|'dark', toggle(): void, set(m): void }`. Storage key `jellystics-theme-mode` (same key as the legacy app so the user's choice carries over). Sets `document.documentElement.dataset.theme`.

- [ ] **Step 1: Create `tokens.css`** — copy verbatim from `design/jellystics-design.html` the three token blocks: `:root { … }` (lines 12–79), `@media (prefers-color-scheme: dark) { :root:where(:not([data-theme="light"])) { … } }` and `:root[data-theme="dark"] { … }` (lines 80–116), **including** the play-method ramp (`--m-direct`, `--m-stream`, `--m-transcode` and their `-ink`) present in all three blocks. Do not add or rename tokens.

- [ ] **Step 2: Create `base.css`** — copy the `/* ---------- base ---------- */` section of the design file (lines 118–131) and add:

```css
html, body, #root { height: 100%; }
body { overflow-x: hidden; }
* { scrollbar-width: thin; scrollbar-color: var(--axis) transparent; }
.sr-only { position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip: rect(0 0 0 0); white-space: nowrap; border: 0; }
@media (prefers-reduced-motion: reduce) { *, *::before, *::after { animation-duration: 0.01ms !important; transition-duration: 0.01ms !important; } }
```

- [ ] **Step 3: Write the failing test `src/lib/theme.test.ts`**

```ts
import { renderHook, act } from '@testing-library/react'
import { useThemeMode } from './theme'

describe('useThemeMode', () => {
  beforeEach(() => { localStorage.clear(); delete document.documentElement.dataset.theme })

  it('defaults to the OS preference (light in jsdom) and stamps <html>', () => {
    const { result } = renderHook(() => useThemeMode())
    expect(result.current.mode).toBe('light')
    expect(document.documentElement.dataset.theme).toBe('light')
  })

  it('restores the legacy stored choice', () => {
    localStorage.setItem('jellystics-theme-mode', 'dark')
    const { result } = renderHook(() => useThemeMode())
    expect(result.current.mode).toBe('dark')
  })

  it('toggles and persists', () => {
    const { result } = renderHook(() => useThemeMode())
    act(() => result.current.toggle())
    expect(result.current.mode).toBe('dark')
    expect(localStorage.getItem('jellystics-theme-mode')).toBe('dark')
    expect(document.documentElement.dataset.theme).toBe('dark')
  })
})
```

- [ ] **Step 4: Run** `pnpm test src/lib/theme.test.ts` → FAIL (module not found).

- [ ] **Step 5: Implement `src/lib/theme.ts`**

```ts
import { useCallback, useEffect, useSyncExternalStore } from 'react'

export type ThemeMode = 'light' | 'dark'
const KEY = 'jellystics-theme-mode'
const listeners = new Set<() => void>()

function read(): ThemeMode {
  try {
    const v = localStorage.getItem(KEY)
    if (v === 'light' || v === 'dark') return v
  } catch { /* storage blocked */ }
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

let current: ThemeMode | null = null
function get(): ThemeMode { return (current ??= read()) }
function setMode(m: ThemeMode) {
  current = m
  try { localStorage.setItem(KEY, m) } catch { /* ignore */ }
  document.documentElement.dataset.theme = m
  listeners.forEach((l) => l())
}

export function useThemeMode() {
  const mode = useSyncExternalStore((cb) => { listeners.add(cb); return () => listeners.delete(cb) }, get, get)
  useEffect(() => { document.documentElement.dataset.theme = mode }, [mode])
  const toggle = useCallback(() => setMode(get() === 'dark' ? 'light' : 'dark'), [])
  return { mode, toggle, set: setMode }
}

/** test helper */
export function __resetThemeForTests() { current = null }
```

Add `__resetThemeForTests()` to the test `beforeEach`.

- [ ] **Step 6: Run** `pnpm test src/lib/theme.test.ts` → 3 passed.

- [ ] **Step 7: Commit** — `git commit -m "feat: add design tokens, base styles and theme mode"`

---

### Task 3: Formatters

**Files:**
- Create: `frontend/src/lib/format.ts`, `frontend/src/lib/format.test.ts`

**Interfaces:**
- Produces (all pure, `locale` defaults to `i18n.language` resolved by callers via `useFormat()`):
  - `formatDuration(seconds: number, locale?: string): string` → `45 s`, `12 min`, `1 h 05`, `1 248 h`
  - `formatPosition(seconds: number): string` → `1:05:12` / `5:09`
  - `formatNumber(n: number, locale?: string): string`; `formatCompact(n, locale?)` (compact only ≥ 10 000)
  - `formatPercent(ratio: number, locale?: string): string` → `87 %`, `4,2 %` (fr), 0 decimals unless < 10 %
  - `formatHour(h: number): string` → `00 h` … `23 h`
  - `formatDay(date: Date, locale?, now = new Date()): string` → `22 nov.` / `22 nov. 2025`
  - `formatRelative(date: Date, locale?, now = new Date()): string` → `il y a 3 min`
  - `formatDelta(ratio: number, locale?): { text: string; direction: 'up'|'down'|'flat' }` → `+12 %`, `−3 %` (U+2212)
  - `weekdayLabels(locale?): string[]` Monday first, short (`lun.` … `dim.`)
  - `useFormat()` hook returning all of the above bound to the current i18n language.

- [ ] **Step 1: Write failing tests `format.test.ts`**

```ts
import { formatDuration, formatPosition, formatNumber, formatCompact, formatPercent, formatHour, formatDay, formatRelative, formatDelta, weekdayLabels } from './format'

const nb = (s: string) => s.replace(/[  ]/g, ' ') // normalise fr spaces

describe('formatDuration', () => {
  it.each([[0, '0 s'], [45, '45 s'], [59.6, '1 min'], [720, '12 min'], [3900, '1 h 05'], [3600 * 99 + 59 * 60, '99 h 59'], [3600 * 1248, '1 248 h']])('%s s → %s', (s, out) => {
    expect(nb(formatDuration(s, 'fr'))).toBe(out)
  })
})
describe('formatPosition', () => {
  it('formats media positions', () => {
    expect(formatPosition(309)).toBe('5:09')
    expect(formatPosition(3912)).toBe('1:05:12')
  })
})
describe('numbers', () => {
  it('uses the locale grouping and compact only from 10 000', () => {
    expect(nb(formatNumber(12480, 'fr'))).toBe('12 480')
    expect(nb(formatCompact(9999, 'fr'))).toBe('9 999')
    expect(nb(formatCompact(12500, 'fr'))).toBe('12,5 k')
    expect(formatNumber(12480, 'en')).toBe('12,480')
  })
  it('percent: 0 decimals unless < 10 %', () => {
    expect(nb(formatPercent(0.873, 'fr'))).toBe('87 %')
    expect(nb(formatPercent(0.042, 'fr'))).toBe('4,2 %')
    expect(formatPercent(0.873, 'en')).toBe('87%')
  })
})
describe('dates', () => {
  const now = new Date(2026, 10, 22, 15, 0)
  it('hour with 2 digits', () => { expect(formatHour(7)).toBe('07 h'); expect(formatHour(21)).toBe('21 h') })
  it('day: year only when different', () => {
    expect(formatDay(new Date(2026, 10, 22), 'fr', now)).toBe('22 nov.')
    expect(formatDay(new Date(2025, 10, 22), 'fr', now)).toBe('22 nov. 2025')
  })
  it('relative', () => {
    expect(formatRelative(new Date(2026, 10, 22, 14, 57), 'fr', now)).toBe('il y a 3 minutes')
  })
  it('weekdays start on Monday', () => {
    expect(weekdayLabels('fr')[0]).toBe('lun.')
    expect(weekdayLabels('fr')[6]).toBe('dim.')
  })
})
describe('formatDelta', () => {
  it('signs and direction', () => {
    expect(formatDelta(0.12, 'fr')).toEqual({ text: '+12 %', direction: 'up' })
    expect(nb(formatDelta(-0.031, 'fr').text)).toBe('−3,1 %')
    expect(formatDelta(-0.031, 'fr').direction).toBe('down')
    expect(formatDelta(0, 'fr').direction).toBe('flat')
  })
})
```

- [ ] **Step 2: Run** → FAIL (module missing).

- [ ] **Step 3: Implement `format.ts`**

```ts
import { useTranslation } from 'react-i18next'

const nf = (locale: string, opts?: Intl.NumberFormatOptions) => new Intl.NumberFormat(locale, opts)

export function formatNumber(n: number, locale = 'fr'): string { return nf(locale).format(Math.round(n)) }
export function formatCompact(n: number, locale = 'fr'): string {
  if (Math.abs(n) < 10000) return formatNumber(n, locale)
  return nf(locale, { notation: 'compact', maximumFractionDigits: 1 }).format(n)
}
export function formatPercent(ratio: number, locale = 'fr'): string {
  const digits = Math.abs(ratio) < 0.1 && ratio !== 0 ? 1 : 0
  return nf(locale, { style: 'percent', minimumFractionDigits: digits, maximumFractionDigits: digits }).format(ratio)
}
export function formatDuration(seconds: number, locale = 'fr'): string {
  const s = Math.round(seconds)
  if (s < 60) return `${s} s`
  const m = Math.round(s / 60)
  if (m < 60) return `${m} min`
  const h = Math.floor(m / 60), mm = m % 60
  if (h >= 100) return `${formatNumber(h, locale)} h`
  return `${h} h ${String(mm).padStart(2, '0')}`
}
export function formatPosition(seconds: number): string {
  const s = Math.max(0, Math.round(seconds)), h = Math.floor(s / 3600), m = Math.floor((s % 3600) / 60), x = s % 60
  return (h ? `${h}:${String(m).padStart(2, '0')}` : String(m)) + ':' + String(x).padStart(2, '0')
}
export function formatHour(h: number): string { return `${String(h).padStart(2, '0')} h` }
export function formatDay(date: Date, locale = 'fr', now = new Date()): string {
  const sameYear = date.getFullYear() === now.getFullYear()
  return new Intl.DateTimeFormat(locale, { day: 'numeric', month: 'short', ...(sameYear ? {} : { year: 'numeric' }) }).format(date)
}
export function formatRelative(date: Date, locale = 'fr', now = new Date()): string {
  const rtf = new Intl.RelativeTimeFormat(locale, { numeric: 'auto' })
  const diff = (date.getTime() - now.getTime()) / 1000
  const abs = Math.abs(diff)
  if (abs < 60) return rtf.format(Math.round(diff), 'second')
  if (abs < 3600) return rtf.format(Math.round(diff / 60), 'minute')
  if (abs < 86400) return rtf.format(Math.round(diff / 3600), 'hour')
  return rtf.format(Math.round(diff / 86400), 'day')
}
export function formatDelta(ratio: number, locale = 'fr'): { text: string; direction: 'up' | 'down' | 'flat' } {
  if (ratio === 0) return { text: formatPercent(0, locale), direction: 'flat' }
  const body = formatPercent(Math.abs(ratio), locale)
  return ratio > 0 ? { text: `+${body}`, direction: 'up' } : { text: `−${body}`, direction: 'down' }
}
export function weekdayLabels(locale = 'fr'): string[] {
  const monday = new Date(2024, 0, 1) // a Monday
  return Array.from({ length: 7 }, (_, i) => new Intl.DateTimeFormat(locale, { weekday: 'short' }).format(new Date(monday.getTime() + i * 86400000)))
}

export function useFormat() {
  const { i18n } = useTranslation()
  const l = i18n.resolvedLanguage || 'fr'
  return {
    duration: (s: number) => formatDuration(s, l), position: formatPosition, number: (n: number) => formatNumber(n, l),
    compact: (n: number) => formatCompact(n, l), percent: (r: number) => formatPercent(r, l), hour: formatHour,
    day: (d: Date) => formatDay(d, l), relative: (d: Date) => formatRelative(d, l), delta: (r: number) => formatDelta(r, l),
    weekdays: () => weekdayLabels(l), locale: l,
  }
}
```

- [ ] **Step 4: Run** `pnpm test src/lib/format.test.ts` → all pass. If `formatPercent(…, 'fr')` yields `87 %` with U+202F, the `nb()` helper already normalises it.

- [ ] **Step 5: Commit** — `git commit -m "feat: add value formatters"`

---

### Task 4: Period model and URL state

**Files:**
- Create: `frontend/src/lib/period.ts`, `frontend/src/lib/period.test.ts`, `frontend/src/lib/useUrlState.ts`, `frontend/src/lib/useUrlState.test.tsx`

**Interfaces:**
- Produces:
  - `type Preset = 'today'|'7d'|'30d'|'90d'|'mtd'|'ytd'|'all'`
  - `interface Period { preset: Preset | 'custom'; from: string; to: string /* YYYY-MM-DD, local, to exclusive */; compare: boolean }`
  - `parsePeriod(params: URLSearchParams, today: string /* YYYY-MM-DD in server TZ */): Period` — default `30d`; `from`/`to` win over `range` when both valid.
  - `serializePeriod(p: Period): Record<string, string>` — preset → `{ range }`, custom → `{ from, to }`, plus `compare: '1'` when true; `30d` and `compare=false` are omitted (clean URLs).
  - `previousPeriod(p: Period): { from: string; to: string }` — same length, immediately before; `all` → `from = to = p.from`.
  - `todayIn(tz: string, now = new Date()): string`
  - `PERIOD_PARAMS = ['range', 'from', 'to', 'compare'] as const`
  - `usePeriod(): [Period, (next: Partial<Period>) => void]` (reads tz from `useServerConfig()`; until loaded uses the browser TZ)
  - `useSearchParam(key: string, fallback?: string): [string | undefined, (v?: string) => void]` (replace in history for filters, push for tabs via option `{ push: true }`)

- [ ] **Step 1: Write failing tests `period.test.ts`**

```ts
import { parsePeriod, serializePeriod, previousPeriod, todayIn } from './period'
const P = (s: string) => new URLSearchParams(s)
const today = '2026-11-22'

it('defaults to the last 30 days, end exclusive', () => {
  expect(parsePeriod(P(''), today)).toEqual({ preset: '30d', from: '2026-10-24', to: '2026-11-23', compare: false })
})
it('presets', () => {
  expect(parsePeriod(P('range=7d'), today)).toMatchObject({ from: '2026-11-16', to: '2026-11-23' })
  expect(parsePeriod(P('range=today'), today)).toMatchObject({ from: '2026-11-22', to: '2026-11-23' })
  expect(parsePeriod(P('range=mtd'), today)).toMatchObject({ from: '2026-11-01', to: '2026-11-23' })
  expect(parsePeriod(P('range=ytd'), today)).toMatchObject({ from: '2026-01-01', to: '2026-11-23' })
  expect(parsePeriod(P('range=all'), today)).toMatchObject({ preset: 'all', from: '2000-01-01', to: '2026-11-23' })
})
it('custom from/to wins and invalid input falls back', () => {
  expect(parsePeriod(P('range=7d&from=2026-10-01&to=2026-11-01'), today)).toEqual({ preset: 'custom', from: '2026-10-01', to: '2026-11-01', compare: false })
  expect(parsePeriod(P('from=nope&to=2026-11-01'), today).preset).toBe('30d')
  expect(parsePeriod(P('from=2026-11-05&to=2026-11-01'), today).preset).toBe('30d')
})
it('compare flag', () => { expect(parsePeriod(P('compare=1'), today).compare).toBe(true) })
it('serializes to clean params', () => {
  expect(serializePeriod({ preset: '30d', from: '', to: '', compare: false })).toEqual({})
  expect(serializePeriod({ preset: '7d', from: '', to: '', compare: true })).toEqual({ range: '7d', compare: '1' })
  expect(serializePeriod({ preset: 'custom', from: '2026-10-01', to: '2026-11-01', compare: false })).toEqual({ from: '2026-10-01', to: '2026-11-01' })
})
it('previous period has the same length', () => {
  expect(previousPeriod({ preset: '7d', from: '2026-11-16', to: '2026-11-23', compare: true })).toEqual({ from: '2026-11-09', to: '2026-11-16' })
})
it('today in a timezone', () => {
  const d = new Date(Date.UTC(2026, 10, 21, 23, 30)) // 00:30 in Paris on the 22nd
  expect(todayIn('Europe/Paris', d)).toBe('2026-11-22')
  expect(todayIn('UTC', d)).toBe('2026-11-21')
})
```

- [ ] **Step 2: Run** → FAIL.

- [ ] **Step 3: Implement `period.ts`** (date arithmetic on `YYYY-MM-DD` strings in UTC to avoid DST issues)

```ts
import { formatInTimeZone } from 'date-fns-tz'

export type Preset = 'today' | '7d' | '30d' | '90d' | 'mtd' | 'ytd' | 'all'
export interface Period { preset: Preset | 'custom'; from: string; to: string; compare: boolean }
export const PERIOD_PARAMS = ['range', 'from', 'to', 'compare'] as const
const PRESETS: Preset[] = ['today', '7d', '30d', '90d', 'mtd', 'ytd', 'all']
const RE = /^\d{4}-\d{2}-\d{2}$/

const toDate = (s: string) => new Date(`${s}T00:00:00Z`)
const toStr = (d: Date) => d.toISOString().slice(0, 10)
export const addDays = (s: string, n: number) => { const d = toDate(s); d.setUTCDate(d.getUTCDate() + n); return toStr(d) }
const valid = (s: string | null): s is string => !!s && RE.test(s) && !Number.isNaN(toDate(s).getTime())

export function todayIn(tz: string, now = new Date()): string { return formatInTimeZone(now, tz, 'yyyy-MM-dd') }

function presetRange(p: Preset, today: string): { from: string; to: string } {
  const to = addDays(today, 1)
  switch (p) {
    case 'today': return { from: today, to }
    case '7d': return { from: addDays(today, -6), to }
    case '30d': return { from: addDays(today, -29), to }
    case '90d': return { from: addDays(today, -89), to }
    case 'mtd': return { from: `${today.slice(0, 7)}-01`, to }
    case 'ytd': return { from: `${today.slice(0, 4)}-01-01`, to }
    case 'all': return { from: '2000-01-01', to }
  }
}

export function parsePeriod(params: URLSearchParams, today: string): Period {
  const compare = params.get('compare') === '1'
  const from = params.get('from'), to = params.get('to')
  if (valid(from) && valid(to) && from < to) return { preset: 'custom', from, to, compare }
  const r = params.get('range') as Preset | null
  const preset: Preset = r && PRESETS.includes(r) ? r : '30d'
  return { preset, ...presetRange(preset, today), compare }
}

export function serializePeriod(p: Period): Record<string, string> {
  const out: Record<string, string> = {}
  if (p.preset === 'custom') { out.from = p.from; out.to = p.to } else if (p.preset !== '30d') out.range = p.preset
  if (p.compare) out.compare = '1'
  return out
}

export function previousPeriod(p: Period): { from: string; to: string } {
  if (p.preset === 'all') return { from: p.from, to: p.from }
  const days = Math.round((toDate(p.to).getTime() - toDate(p.from).getTime()) / 86400000)
  return { from: addDays(p.from, -days), to: p.from }
}
```

- [ ] **Step 4: Run** → period tests pass.

- [ ] **Step 5: Write failing test `useUrlState.test.tsx`**

```tsx
import { renderHook, act } from '@testing-library/react'
import { MemoryRouter, useLocation } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { usePeriod, useSearchParam } from './useUrlState'

function wrap(initial: string) {
  const qc = new QueryClient()
  qc.setQueryData(['server-config'], { configured: true, jellyfinUrl: '', timezone: 'Europe/Paris' })
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={qc}><MemoryRouter initialEntries={[initial]}>{children}</MemoryRouter></QueryClientProvider>
  )
}

it('reads and writes the period, keeping other params', () => {
  const { result } = renderHook(() => ({ p: usePeriod(), loc: useLocation() }), { wrapper: wrap('/history?user=lea&range=7d') })
  expect(result.current.p[0].preset).toBe('7d')
  act(() => result.current.p[1]({ preset: '90d' }))
  expect(result.current.loc.search).toBe('?user=lea&range=90d')
  act(() => result.current.p[1]({ compare: true }))
  expect(new URLSearchParams(result.current.loc.search).get('compare')).toBe('1')
})

it('single param read/write and removal', () => {
  const { result } = renderHook(() => ({ s: useSearchParam('tab', 'overview'), loc: useLocation() }), { wrapper: wrap('/users/lea') })
  expect(result.current.s[0]).toBe('overview')
  act(() => result.current.s[1]('history'))
  expect(result.current.loc.search).toBe('?tab=history')
  act(() => result.current.s[1](undefined))
  expect(result.current.loc.search).toBe('')
})
```

- [ ] **Step 6: Implement `useUrlState.ts`** (depends on `useServerConfig` from Task 5 — implement Task 5's `serverConfig.ts` first if executing out of order; the test seeds the query cache key `['server-config']`)

```ts
import { useCallback, useMemo } from 'react'
import { useSearchParams } from 'react-router-dom'
import { parsePeriod, serializePeriod, todayIn, PERIOD_PARAMS, type Period } from './period'
import { useServerConfig } from './serverConfig'

export function usePeriod(): [Period, (next: Partial<Period>) => void] {
  const [params, setParams] = useSearchParams()
  const tz = useServerConfig().data?.timezone ?? Intl.DateTimeFormat().resolvedOptions().timeZone
  const today = todayIn(tz)
  const period = useMemo(() => parsePeriod(params, today), [params, today])
  const update = useCallback((next: Partial<Period>) => {
    setParams((prev) => {
      const merged: Period = { ...parsePeriod(prev, today), ...next }
      const out = new URLSearchParams(prev)
      PERIOD_PARAMS.forEach((k) => out.delete(k))
      Object.entries(serializePeriod(merged)).forEach(([k, v]) => out.set(k, v))
      return out
    }, { replace: false })
  }, [setParams, today])
  return [period, update]
}

export function useSearchParam(key: string, fallback?: string, opts: { push?: boolean } = {}): [string | undefined, (v?: string) => void] {
  const [params, setParams] = useSearchParams()
  const value = params.get(key) ?? fallback
  const set = useCallback((v?: string) => {
    setParams((prev) => {
      const out = new URLSearchParams(prev)
      if (v === undefined || v === '' || v === fallback) out.delete(key); else out.set(key, v)
      return out
    }, { replace: !opts.push })
  }, [key, fallback, opts.push, setParams])
  return [value, set]
}
```

- [ ] **Step 7: Run** `pnpm test src/lib` → pass. **Commit** — `git commit -m "feat: add period model and url state hooks"`

---

### Task 5: Server timezone (backend), auth helpers, API client, React Query, server config

**Files:**
- Modify: `backend/internal/handler/frontend_auth.go` (`IsConfigured`), Test: `backend/internal/router/security_test.go` (add one test)
- Create: `frontend/src/lib/auth.ts`, `frontend/src/lib/api.ts`, `frontend/src/lib/api.test.ts`, `frontend/src/lib/queryClient.ts`, `frontend/src/lib/serverConfig.ts`
- Move: `frontend/src/lib/device.ts` → keep file, re-export from `auth.ts`

**Interfaces:**
- Backend produces: `GET /auth/isConfigured` → `{"state":0|1,"jellyfinUrl"?:string,"timezone":"Europe/Paris"}` (from `statsql.Timezone()`), public.
- Frontend produces:
  - `getToken(): string | null`, `setSession(token: string, username: string)`, `clearSession()`, `logout(): Promise<void>` (POST `/auth/logout` then clear, then `location.assign('/login')`), re-export `getDeviceId`, `getDeviceName`.
  - `class ApiError extends Error { status: number; body: unknown }`
  - `apiFetch<T>(path: string, opts?: { method?: string; params?: Record<string, string | number | boolean | undefined>; body?: unknown; signal?: AbortSignal }): Promise<T>` — adds `Authorization: Bearer`, JSON body, skips undefined params; on 401 → `clearSession()` + redirect to `/login` (unless already there); throws `ApiError` on non-2xx.
  - `queryClient` (staleTime 30 s, retry 1, refetchOnWindowFocus false).
  - `useServerConfig(): UseQueryResult<{ configured: boolean; jellyfinUrl?: string; timezone: string }>` — query key `['server-config']`, `staleTime: Infinity`.

- [ ] **Step 1: Backend failing test** — append to `backend/internal/router/security_test.go`:

```go
func TestIsConfigured_ExposesTimezone(t *testing.T) {
	t.Setenv("TZ", "Europe/Paris")
	r, _ := fullRouter(t, true)
	w := do(r, "GET", "/auth/isConfigured", "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"timezone":"Europe/Paris"`) {
		t.Fatalf("isConfigured = %d %s, want timezone", w.Code, w.Body.String())
	}
}
```

Run `cd backend && POSTGRES_PORT=55432 go test ./internal/router/ -run Timezone` → FAIL.

- [ ] **Step 2: Implement** in `frontend_auth.go` `IsConfigured`: compute `tz, _ := statsql.Timezone()` and include `"timezone": tz` in both JSON responses (import `github.com/Jellystics/Jellystics/internal/statsql`). Re-run → PASS.

- [ ] **Step 3: Frontend failing test `api.test.ts`**

```ts
import { apiFetch, ApiError } from './api'
import { setSession, getToken } from './auth'

beforeEach(() => { localStorage.clear(); vi.restoreAllMocks() })

it('sends the bearer token and query params, skipping undefined', async () => {
  setSession('tok', 'lea')
  const spy = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({ ok: 1 }), { status: 200, headers: { 'Content-Type': 'application/json' } }))
  const res = await apiFetch<{ ok: number }>('/api/v2/overview', { params: { from: '2026-11-01', userId: undefined, compare: true } })
  expect(res).toEqual({ ok: 1 })
  const [url, init] = spy.mock.calls[0]
  expect(String(url)).toBe('/api/v2/overview?from=2026-11-01&compare=true')
  expect((init!.headers as Record<string, string>).Authorization).toBe('Bearer tok')
})

it('throws ApiError with status', async () => {
  vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response('{"error":"x"}', { status: 500 }))
  await expect(apiFetch('/x')).rejects.toMatchObject({ status: 500 })
  await expect(apiFetch('/x')).rejects.toBeInstanceOf(ApiError)
})

it('clears the session on 401', async () => {
  setSession('tok', 'lea')
  const assign = vi.fn()
  Object.defineProperty(window, 'location', { value: { ...window.location, pathname: '/users', assign }, writable: true })
  vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response('', { status: 401 }))
  await expect(apiFetch('/x')).rejects.toMatchObject({ status: 401 })
  expect(getToken()).toBeNull()
  expect(assign).toHaveBeenCalledWith('/login')
})
```

- [ ] **Step 4: Implement `auth.ts`**

```ts
export { getDeviceId, getDeviceName } from './device'
const TOKEN = 'jellystics-token', USER = 'jellystics-username'

export function getToken(): string | null { try { return localStorage.getItem(TOKEN) } catch { return null } }
export function getUsername(): string { try { return localStorage.getItem(USER) ?? '' } catch { return '' } }
export function setSession(token: string, username: string) { localStorage.setItem(TOKEN, token); localStorage.setItem(USER, username) }
export function clearSession() { try { localStorage.removeItem(TOKEN); localStorage.removeItem(USER) } catch { /* ignore */ } }
export function getUserId(): string {
  const t = getToken(); if (!t) return ''
  try { return JSON.parse(atob(t.split('.')[1].replace(/-/g, '+').replace(/_/g, '/'))).userId ?? '' } catch { return '' }
}
export async function logout() {
  const t = getToken()
  if (t) await fetch('/auth/logout', { method: 'POST', headers: { Authorization: `Bearer ${t}` } }).catch(() => undefined)
  clearSession()
  window.location.assign('/login')
}
```

- [ ] **Step 5: Implement `api.ts`**

```ts
import { clearSession, getToken } from './auth'

export class ApiError extends Error {
  constructor(public status: number, public body: unknown) { super(`HTTP ${status}`) }
}
type Params = Record<string, string | number | boolean | undefined | null>

export async function apiFetch<T>(path: string, opts: { method?: string; params?: Params; body?: unknown; signal?: AbortSignal } = {}): Promise<T> {
  const qs = new URLSearchParams()
  Object.entries(opts.params ?? {}).forEach(([k, v]) => { if (v !== undefined && v !== null && v !== '') qs.set(k, String(v)) })
  const url = qs.toString() ? `${path}?${qs}` : path
  const headers: Record<string, string> = { Accept: 'application/json' }
  const token = getToken(); if (token) headers.Authorization = `Bearer ${token}`
  let body: BodyInit | undefined
  if (opts.body !== undefined) { headers['Content-Type'] = 'application/json'; body = JSON.stringify(opts.body) }
  const res = await fetch(url, { method: opts.method ?? (body ? 'POST' : 'GET'), headers, body, signal: opts.signal })
  const text = await res.text()
  let data: unknown = undefined
  try { data = text ? JSON.parse(text) : undefined } catch { data = text }
  if (res.status === 401) {
    clearSession()
    if (window.location.pathname !== '/login') window.location.assign('/login')
  }
  if (!res.ok) throw new ApiError(res.status, data)
  return data as T
}
```

- [ ] **Step 6: Implement `queryClient.ts` and `serverConfig.ts`**

```ts
// queryClient.ts
import { QueryClient } from '@tanstack/react-query'
export const queryClient = new QueryClient({ defaultOptions: { queries: { staleTime: 30_000, retry: 1, refetchOnWindowFocus: false } } })
```

```ts
// serverConfig.ts
import { useQuery } from '@tanstack/react-query'
import { apiFetch } from './api'
export interface ServerConfig { configured: boolean; jellyfinUrl?: string; timezone: string }
export function useServerConfig() {
  return useQuery({
    queryKey: ['server-config'],
    staleTime: Infinity,
    queryFn: async (): Promise<ServerConfig> => {
      const r = await apiFetch<{ state: number; jellyfinUrl?: string; timezone?: string }>('/auth/isConfigured')
      return { configured: r.state === 1, jellyfinUrl: r.jellyfinUrl, timezone: r.timezone || 'UTC' }
    },
  })
}
```

- [ ] **Step 7: Run** `pnpm test src/lib` and the backend test → pass. **Commit** — `git commit -m "feat: add api client, auth helpers and server timezone"`

---

### Task 6: UI kit — static components

**Files (each `X.tsx` + `X.module.css`, one test file `src/ui/static.test.tsx`):**
Button, Badge, Chip, Card (Card, CardHeader, CardBody, CardFooter), Skeleton, EmptyState, ErrorState, Avatar, AvatarStack, Art, Meter, SegBar, Breadcrumbs, PageHeader, EntityHeader, TypeKey, `src/ui/index.ts`.

**Interfaces (Produces):**

```ts
type MediaType = 'movie' | 'series' | 'music'
type PlayMethod = 'direct' | 'stream' | 'transcode'
Button: (props: ButtonHTMLAttributes & { variant?: 'primary'|'secondary'|'ghost'|'danger'; size?: 'sm'|'md'|'lg'; icon?: ReactNode; iconOnly?: boolean /* requires aria-label */ ; asChild?: false }) // forwards ref
Badge: ({ tone?: 'neutral'|'accent'|'good'|'warning'|'serious'|'critical'|'new'; icon?: ReactNode; dot?: string /* css color */; children })
MediaTypeBadge: ({ type: MediaType }) ; MethodBadge: ({ method: PlayMethod }) ; ViewBadge: ({ view: boolean }) ; NewBadge: ()
Chip: ({ pressed?: boolean; onRemove?: () => void; onClick?; children })
Card: ({ children, className }) ; CardHeader: ({ title: ReactNode; subtitle?: ReactNode; action?: ReactNode }) ; CardBody: ({ flush?: boolean }) ; CardFooter
Skeleton: ({ width?: number|string; height?: number|string; radius?: 'sm'|'md'|'lg' })
EmptyState: ({ icon?: ReactNode; title: string; description?: string; action?: ReactNode })
ErrorState: ({ title: string; description?: string; onRetry?: () => void })
Avatar: ({ id: string; name: string; size?: 'sm'|'md'|'lg'; src?: string }) // color = stable hash of id over the 8 categorical slots
AvatarStack: ({ users: { id: string; name: string }[]; max?: number })
Art: ({ title: string; kind?: 'poster'|'thumb'|'square'; size?: 'sm'|'md'|'lg'; src?: string; seed?: string }) // image with fallback gradient + initials
Meter: ({ value: number /* 0..1 */; tone?: 'accent'|'serious'|'muted'|'series'; thin?: boolean; label: string /* aria */ })
SegBar: ({ parts: { label: string; value: number; color: string; ink?: string; display?: string }[]; thin?: boolean }) // tall: % labels when share > 9 %, legend below; keyboard focusable parts
Breadcrumbs: ({ items: { label: string; to?: string }[] })
PageHeader: ({ title: string; subtitle?: ReactNode; actions?: ReactNode })
EntityHeader: ({ art: ReactNode; kicker?: ReactNode; title: string; meta?: ReactNode[]; actions?: ReactNode })
TypeKey: ({ type: MediaType; label?: string })
```

- [ ] **Step 1: Write failing tests `src/ui/static.test.tsx`**

```tsx
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { Button, Badge, MethodBadge, ViewBadge, Avatar, Art, Meter, SegBar, Breadcrumbs, EmptyState } from '@/ui'

it('Button variants and icon-only accessibility', () => {
  render(<><Button variant="primary">Sync</Button><Button iconOnly aria-label="Filtrer" icon={<svg />} /></>)
  expect(screen.getByRole('button', { name: 'Sync' })).toHaveClass('primary')
  expect(screen.getByRole('button', { name: 'Filtrer' })).toHaveClass('iconOnly')
})
it('status badges: neutral background, colored icon only', () => {
  const { container } = render(<MethodBadge method="transcode" />)
  const badge = container.firstElementChild as HTMLElement
  expect(badge).toHaveClass('badge', 'serious')
  expect(badge.querySelector('svg')).not.toBeNull()
})
it('ViewBadge labels', () => {
  render(<><ViewBadge view /><ViewBadge view={false} /></>)
  expect(screen.getByText('app.badge.view')).toBeInTheDocument()
  expect(screen.getByText('app.badge.click')).toBeInTheDocument()
})
it('Avatar color is stable per id', () => {
  const a = render(<Avatar id="lea" name="Léa" />).container.firstElementChild as HTMLElement
  const b = render(<Avatar id="lea" name="Léa" />).container.firstElementChild as HTMLElement
  expect(a.style.background).toBe(b.style.background)
  expect(a).toHaveTextContent('L')
})
it('Art falls back to initials', () => {
  render(<Art title="Station Nord" kind="poster" />)
  expect(screen.getByText('SN')).toBeInTheDocument()
})
it('Meter exposes a progressbar', () => {
  render(<Meter value={0.42} label="Complétion" />)
  expect(screen.getByRole('progressbar', { name: 'Complétion' })).toHaveAttribute('aria-valuenow', '42')
})
it('SegBar prints shares above 9 % and a legend', () => {
  render(<SegBar parts={[{ label: 'Direct', value: 68, color: 'var(--m-direct)' }, { label: 'Stream', value: 5, color: 'var(--m-stream)' }]} />)
  expect(screen.getByText(/93/)).toBeInTheDocument() // 68/73 ≈ 93 %
  expect(screen.getAllByText('Stream')).toHaveLength(1) // legend only, no in-bar label under 9 %
})
it('Breadcrumbs: last item is current page, others are links', () => {
  render(<MemoryRouter><Breadcrumbs items={[{ label: 'Séries', to: '/libraries/a' }, { label: 'Station Nord' }]} /></MemoryRouter>)
  expect(screen.getByRole('link', { name: 'Séries' })).toHaveAttribute('href', '/libraries/a')
  expect(screen.getByText('Station Nord')).toHaveAttribute('aria-current', 'page')
})
it('EmptyState', () => {
  render(<EmptyState title="Aucune lecture" description="Élargis la période" />)
  expect(screen.getByText('Aucune lecture')).toBeInTheDocument()
})
```

In `src/test/setup.ts` add an i18n mock so `t(key)` returns the key:

```ts
vi.mock('react-i18next', () => ({
  useTranslation: () => ({ t: (k: string, o?: Record<string, unknown>) => (o && 'count' in o ? `${k}:${o.count}` : k), i18n: { resolvedLanguage: 'fr', language: 'fr', changeLanguage: vi.fn() } }),
  Trans: ({ children }: { children: React.ReactNode }) => children,
  initReactI18next: { type: '3rdParty', init: () => {} },
}))
```

- [ ] **Step 2: Run** → FAIL.

- [ ] **Step 3: Implement the components.** CSS: port the matching rules of `design/jellystics-design.html` into each module (Button ← `/* Button */` lines 172–188; Badge/Chip ← `/* Badges & chips */` 212–238 incl. the status-badge rules "only the icon / dot carries the status color" and `.badge.new`; Card ← 239–250; Art ← 294–304; Avatar ← 305–310; TypeKey ← 311–314; Meter ← 315–319; Breadcrumbs ← 320–325; PageHeader ← 326–331; EntityHeader ← 332–338; EmptyState ← 339–343; Skeleton ← 344–348). In modules, class names are the same as in the design file (e.g. `.btn`, `.primary`, `.sm`) so the reference stays greppable. Icons from `lucide-react` with `size={16}` and `strokeWidth={1.6}` (badges `size={12}`). Badge mappings:

```tsx
// MethodBadge
const M = { direct: { tone: 'good', icon: <Check size={12} strokeWidth={1.6} />, key: 'app.method.direct' }, stream: { tone: 'warning', icon: <Zap size={12} strokeWidth={1.6} />, key: 'app.method.stream' }, transcode: { tone: 'serious', icon: <Cpu size={12} strokeWidth={1.6} />, key: 'app.method.transcode' } } as const
// MediaTypeBadge: dot = var(--t-movie|--t-series|--t-music), label key app.type.movie|series|music
// ViewBadge: view → tone accent + dot var(--accent), key app.badge.view ; click → dot var(--axis), key app.badge.click
```

Avatar color: `const SLOTS = ['#6250d6','#2a78d6','#d55181','#1baf7a','#eb6834','#52514e','#eda100','#e34948']`, index = `[...id].reduce((a, c) => (a * 31 + c.charCodeAt(0)) >>> 0, 7) % 8`. Art initials: first letters of the first two words with ≥ 3 letters or starting with an uppercase letter/digit (same rule as `art()` in the design file); gradient `linear-gradient(160deg, c, color-mix(in srgb, c 55%, #000))` with `c` chosen from the design file `ART` list by hash of `seed ?? title`; when `src` is given render `<img loading="lazy" alt="">` over the fallback and hide it on error.

SegBar: port `segBar()` from the design file script (tall variant: 28 px, 4 px radius, 2 px gap, label inside only when share > 0.09, legend below listing names; thin variant: 8 px fully rounded). Each part is `tabIndex={0}` with `aria-label="{label} : {percent}"`.

- [ ] **Step 4: Run** `pnpm test src/ui` → pass. Add the i18n keys used (`app.badge.view` = « Vue »/« View »/« Vista », `app.badge.click` = « Clic »/« Click »/« Clic », `app.method.direct` = « Direct »…, `app.method.stream` = « Direct stream », `app.method.transcode` = « Transcodage »/« Transcode »/« Transcodificación », `app.type.movie|series|music` = « Film »/« Série »/« Musique » etc., `app.badge.new` = « NEW ») to the three locale files under a new top-level `"app"` object.

- [ ] **Step 5: Commit** — `git commit -m "feat: add static ui components"`

---

### Task 7: UI kit — interactive components (Radix)

**Files:** `src/ui/{Input,SearchInput,Select,Field,SegmentedControl,Tabs,Switch,Checkbox,Menu,Popover,Tooltip,Dialog,ConfirmDialog,Toast,PeriodPicker}.tsx` (+ modules), test `src/ui/interactive.test.tsx`.

**Interfaces (Produces):**

```ts
Input: InputHTMLAttributes ; SearchInput: ({ value, onChange(v: string), placeholder, shortcut?: string })
Select: ({ value: string; onValueChange(v: string); options: { value: string; label: string }[]; 'aria-label': string })
Field: ({ label: string; htmlFor: string; hint?: string; error?: string; children })
SegmentedControl: <T extends string>({ value: T; onValueChange(v: T); options: { value: T; label: ReactNode; ariaLabel?: string }[]; 'aria-label': string })
Tabs: ({ value: string; onValueChange(v: string); items: { value: string; label: string; count?: number }[]; children?: ReactNode }) // Radix Tabs; pages bind value to useSearchParam('tab', default, { push: true })
Switch: ({ checked; onCheckedChange; label: string }) ; Checkbox: ({ checked; onCheckedChange; label: string })
Menu: ({ trigger: ReactNode; items: ({ label: string; icon?: ReactNode; hint?: string; onSelect(): void; danger?: boolean } | 'separator' | { group: string })[] })
Popover: ({ trigger: ReactNode; children; align?: 'start'|'end' })
Tooltip: ({ content: ReactNode; children: ReactElement })
Dialog: ({ open; onOpenChange; title: string; description?: string; children; footer?: ReactNode })
ConfirmDialog: ({ open; onOpenChange; title: string; description: string; confirmLabel: string; danger?: boolean; onConfirm(): void | Promise<void> })
toast: { success(msg: string): void; error(msg: string, action?: { label: string; onClick(): void }): void; info(msg: string): void } ; Toaster: () => JSX // bottom-left, max 5, 5 s, pauses on hover
PeriodPicker: () // reads/writes usePeriod(): presets today/7d/30d/90d/mtd/ytd/all, custom range (two <input type="date">), compare toggle; trigger shows label + "vs {previous range}" when compare
```

- [ ] **Step 1: Write failing tests `interactive.test.tsx`**

```tsx
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, useLocation } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { SegmentedControl, Tabs, ConfirmDialog, PeriodPicker, Toaster, toast } from '@/ui'
import { useState } from 'react'

it('SegmentedControl selects with mouse and keyboard', async () => {
  function T() { const [v, setV] = useState<'a' | 'b'>('a'); return <SegmentedControl aria-label="Métrique" value={v} onValueChange={setV} options={[{ value: 'a', label: 'Temps' }, { value: 'b', label: 'Vues' }]} /> }
  render(<T />)
  await userEvent.click(screen.getByRole('radio', { name: 'Vues' }))
  expect(screen.getByRole('radio', { name: 'Vues' })).toHaveAttribute('data-state', 'on')
})

it('Tabs expose tab roles and selection', async () => {
  const onChange = vi.fn()
  render(<Tabs value="a" onValueChange={onChange} items={[{ value: 'a', label: 'Aperçu' }, { value: 'b', label: 'Historique', count: 3 }]} />)
  expect(screen.getByRole('tab', { name: /Aperçu/ })).toHaveAttribute('aria-selected', 'true')
  await userEvent.click(screen.getByRole('tab', { name: /Historique/ }))
  expect(onChange).toHaveBeenCalledWith('b')
})

it('ConfirmDialog calls onConfirm', async () => {
  const onConfirm = vi.fn()
  render(<ConfirmDialog open onOpenChange={() => {}} title="Supprimer ?" description="Définitif" confirmLabel="Supprimer" danger onConfirm={onConfirm} />)
  await userEvent.click(screen.getByRole('button', { name: 'Supprimer' }))
  expect(onConfirm).toHaveBeenCalled()
})

it('PeriodPicker writes the range to the URL', async () => {
  const qc = new QueryClient(); qc.setQueryData(['server-config'], { configured: true, timezone: 'Europe/Paris' })
  let search = ''
  function Loc() { search = useLocation().search; return null }
  render(<QueryClientProvider client={qc}><MemoryRouter initialEntries={['/']}><PeriodPicker /><Loc /></MemoryRouter></QueryClientProvider>)
  await userEvent.click(screen.getByRole('button', { name: /app.period.30d/ }))
  await userEvent.click(screen.getByRole('menuitemradio', { name: /app.period.7d/ }))
  expect(search).toBe('?range=7d')
})

it('toast renders messages', async () => {
  render(<Toaster />)
  toast.success('Synchronisation terminée')
  expect(await screen.findByText('Synchronisation terminée')).toBeInTheDocument()
})
```

- [ ] **Step 2: Run** → FAIL.

- [ ] **Step 3: Implement.** Map Radix to design classes: `SegmentedControl` = `@radix-ui/react-toggle-group` type single (`.seg`, items get `role="radio"` by Radix), `Tabs` = `@radix-ui/react-tabs` (`.tabs`), `Switch`/`Checkbox` = Radix (`.switch`, `.checkbox`), `Menu` = `@radix-ui/react-dropdown-menu` (`.pop`, `.pop-item`, `.pop-sep`, `.pop-label`), `Popover`, `Tooltip` (delay 300 ms), `Dialog` (`.dialog*`, overlay `rgba(0,0,0,.4)`), `Select` = `@radix-ui/react-select` styled as `.select`. CSS from design file sections `/* Inputs */` 189–199, `/* Segmented control */` 200–204, `/* Tabs */` 205–211, `/* Menu / popover / tooltip */` 349–366, `/* Dialog */` 367–372, `/* Toast */` 373–376, `/* Switch */` 377–384. `toast` is a tiny external store (`useSyncExternalStore`) with an id counter; `Toaster` renders `.toast` items in a fixed bottom-left stack with `role="status"` / `aria-live="polite"`. `PeriodPicker`: Radix DropdownMenu with `DropdownMenu.RadioGroup` (items are `menuitemradio`), a separator, « Période personnalisée… » opening a Popover with two `<input type="date">` and an Apply button (validates from < to, sets `{ preset: 'custom', from, to: addDays(to, 1) }` — the UI shows inclusive end, the URL stores exclusive), and a `DropdownMenu.CheckboxItem` for compare. Trigger label key `app.period.{preset}` or the custom range formatted with `formatDay`. Add i18n keys `app.period.today|7d|30d|90d|mtd|ytd|all|custom|compare|apply|vs`.

- [ ] **Step 4: Run** `pnpm test src/ui` → pass. **Commit** — `git commit -m "feat: add interactive ui components and period picker"`

---

### Task 8: UI kit — data components

**Files:** `src/ui/{StatTile,StatStrip,BarList,DataTable,SessionCard}.tsx` (+ modules), test `src/ui/data.test.tsx`.

**Interfaces (Produces):**

```ts
StatTile: ({ label: string; value: string; unit?: string; delta?: { text: string; direction: 'up'|'down'|'flat'; good: boolean }; comparedTo?: string; foot?: ReactNode; trend?: number[]; selected?: boolean; onSelect?: () => void })
// delta arrow follows direction; color = good ? --good-ink : --critical-ink ; flat = --ink-3. Selectable tiles are buttons (aria-pressed).
StatStrip: ({ children; columns?: number })
BarList: ({ rows: { key: string; label: string; value: number; to?: string; lead?: ReactNode; badge?: ReactNode; fill?: string; tooltip?: [string, string][] }[]; valueLabel: string; labelHeader?: string; format?: (v: number) => string; max?: number })
DataTable: <T>({ columns: ColumnDef<T>[]; data: T[]; rowHref?: (row: T) => string; pagination?: { page: number; pageSize: number; total: number; onPageChange(p: number): void }; sort?: { id: string; desc: boolean; onChange(s: { id: string; desc: boolean }): void }; dense?: boolean; empty?: ReactNode; loading?: boolean })
// rows with rowHref render the first cell's content inside an <a> covering the row (no onClick-only rows); numeric columns use meta { numeric: true } → right aligned tabular-nums
SessionCard: ({ session: { id: string; title: string; subtitle: string; itemHref: string; type: MediaType; user: { id: string; name: string }; device: string; method: PlayMethod; paused: boolean; positionSec: number; runtimeSec: number; artSrc?: string } })
```

- [ ] **Step 1: Write failing tests `data.test.tsx`**

```tsx
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { StatTile, BarList, DataTable } from '@/ui'

it('StatTile delta: arrow from sign, color from goodness', () => {
  const { container } = render(<StatTile label="Transcodages" value="18" unit="%" delta={{ text: '−4 pts', direction: 'down', good: true }} comparedTo="vs préc." />)
  const d = container.querySelector('.delta')!
  expect(d).toHaveClass('good')
  expect(d.querySelector('[data-icon="down"]')).not.toBeNull()
})
it('selectable StatTile is a pressed button', async () => {
  const onSelect = vi.fn()
  render(<StatTile label="Vues" value="3 412" selected onSelect={onSelect} />)
  const b = screen.getByRole('button', { name: /Vues/ })
  expect(b).toHaveAttribute('aria-pressed', 'true')
  await userEvent.click(b); expect(onSelect).toHaveBeenCalled()
})
it('BarList rows are links with values', () => {
  render(<MemoryRouter><BarList valueLabel="Vues" rows={[{ key: '1', label: 'Station Nord', value: 412, to: '/series/1' }, { key: '2', label: 'Orbital', value: 133 }]} /></MemoryRouter>)
  expect(screen.getByRole('link', { name: /Station Nord/ })).toHaveAttribute('href', '/series/1')
  expect(screen.getByText('412')).toBeInTheDocument()
})
it('DataTable renders rows as links and paginates', async () => {
  const onPageChange = vi.fn()
  render(<MemoryRouter><DataTable columns={[{ accessorKey: 'name', header: 'Nom' }, { accessorKey: 'n', header: 'Vues', meta: { numeric: true } }]} data={[{ name: 'Léa', n: 3 }]} rowHref={() => '/users/lea'} pagination={{ page: 1, pageSize: 25, total: 60, onPageChange }} /></MemoryRouter>)
  expect(screen.getByRole('link', { name: 'Léa' })).toHaveAttribute('href', '/users/lea')
  await userEvent.click(screen.getByRole('button', { name: 'app.table.next' }))
  expect(onPageChange).toHaveBeenCalledWith(2)
})
```

- [ ] **Step 2: Run** → FAIL. **Step 3: Implement** (CSS from design file: Stat tile 251–267, Bar list 268–278, Table 279–293, Live session card 385–396). Links use `AppLink` from Task 11 — until Task 11 exists, import `Link` from react-router-dom and replace with `AppLink` in Task 11 Step 4. Pagination footer text key `app.table.range` (`{{from}}–{{to}} sur {{total}}`), buttons `app.table.prev|next` (icon-only with aria-label). **Step 4: Run** → pass. **Step 5: Commit** — `git commit -m "feat: add data ui components"`

---

### Task 9: Chart kit — core

**Files:** `src/charts/{scale.ts,scale.test.ts,ChartTooltip.tsx,ChartFrame.tsx,LineChart.tsx,Columns.tsx,StackedColumns.tsx,Sparkline.tsx,charts.test.tsx,index.ts,charts.module.css}`

**Interfaces (Produces):**

```ts
// scale.ts (pure)
niceMax(v: number): number                        // 1, 2, 2.5, 5, 10 × 10^n
ticks(max: number, count = 5): number[]           // [0, max/count, …, max]
barPath(x, y, w, h, r = 4): string                // rounded value end, square baseline
barPathH(x, y, w, h, r = 4): string
linePath(points: [number, number][]): string
// components (all: role="img", aria-label = title, hover + focus tooltip, `height` in px, width = container via ResizeObserver)
ChartFrame: ({ title: string; subtitle?: string; legend?: { label: string; color: string; line?: boolean }[]; table: { columns: string[]; rows: (string|number)[][] }; csvName: string; action?: ReactNode; children: ReactNode })
// renders CardHeader + chart/table toggle (SegmentedControl) + CSV download; the table view is the accessibility fallback
LineChart: ({ labels: string[]; series: { name: string; color: string; values: number[] }[]; compare?: { name: string; values: number[] }; area?: boolean; height?: number; yFormat?: (v: number) => string; xEvery?: number; onPointClick?: (index: number) => void })
Columns: ({ labels: string[]; values: number[]; name: string; color?: string; emphasis?: number; height?: number; yFormat?; xEvery?: number; ticks?: boolean; onBarClick?: (i: number) => void })
StackedColumns: ({ labels: string[]; series: { name: string; color: string; values: number[] }[]; height?; yFormat?; xEvery? })
Sparkline: ({ values: number[]; width?: number; height?: number; color?: string })
useChartWidth(): [ref: RefObject<HTMLDivElement>, width: number]
```

- [ ] **Step 1: Failing tests `scale.test.ts`**

```ts
import { niceMax, ticks, barPath } from './scale'
it('niceMax', () => { expect([0.7, 1, 1.4, 2.2, 3, 7, 12, 260].map(niceMax)).toEqual([1, 1, 2, 2.5, 5, 10, 20, 500]) })
it('ticks are clean', () => { expect(ticks(10)).toEqual([0, 2, 4, 6, 8, 10]); expect(ticks(2.5)).toEqual([0, 0.5, 1, 1.5, 2, 2.5]) })
it('barPath has a square baseline and rounded top', () => {
  const d = barPath(0, 10, 20, 90)
  expect(d.startsWith('M0,100V14')).toBe(true) // starts at bottom-left, square corner
  expect(d).toContain('Q0,10 4,10')
  expect(barPath(0, 0, 20, 0)).toBe('')
})
```

- [ ] **Step 2: Failing tests `charts.test.tsx`**

```tsx
import { render, screen, fireEvent } from '@testing-library/react'
import { LineChart, Columns, ChartFrame } from '@/charts'

it('LineChart is an accessible image with one hit zone per label', () => {
  const { container } = render(<div style={{ width: 600 }}><LineChart labels={['a', 'b', 'c']} series={[{ name: 'Temps', color: 'var(--accent)', values: [1, 3, 2] }]} /></div>)
  expect(screen.getByRole('img', { name: 'Temps' })).toBeInTheDocument()
  expect(container.querySelectorAll('[data-hit]')).toHaveLength(3)
})
it('Columns emphasis labels the emphasised bar only', () => {
  render(<Columns labels={['00 h', '01 h']} values={[3, 9]} name="Vues" emphasis={1} />)
  expect(screen.getByText('9')).toBeInTheDocument()
  expect(screen.queryByText('3')).toBeNull()
})
it('tooltip appears on hover and on focus', () => {
  const { container } = render(<Columns labels={['lun.']} values={[4]} name="Vues" />)
  fireEvent.mouseMove(container.querySelector('[data-hit]')!, { clientX: 10, clientY: 10 })
  expect(screen.getByRole('tooltip')).toHaveTextContent('lun.')
})
it('ChartFrame toggles to a table view', () => {
  render(<ChartFrame title="Vues" csvName="vues" table={{ columns: ['Jour', 'Vues'], rows: [['lun.', 4]] }}><div /></ChartFrame>)
  fireEvent.click(screen.getByRole('radio', { name: 'app.chart.table' }))
  expect(screen.getByRole('table')).toHaveTextContent('lun.')
})
```

- [ ] **Step 3: Run** → FAIL. **Step 4: Implement** by porting from `design/jellystics-design.html` `<script>`: `niceMax`, `barPath`, `barPathH`, `lineChart` (→ `LineChart`, incl. compare filet in `var(--axis)`, crosshair, hidden hover dots, end dot), `columns` (→ `Columns`, ticks `ticks(ymax, 5)` = the corrected 5-interval axis), `stackedColumns`, `spark` (→ `Sparkline`). Replace the string-building with JSX; replace the document-level `data-tip` handler with a `ChartTooltip` context: each hit element gets `data-hit`, `tabIndex={0}`, `onMouseMove`/`onFocus` → `show({ x, y, title, rows })`, `onMouseLeave`/`onBlur` → `hide()`; the tooltip is a portal with `role="tooltip"` styled as `.tip` (design file `/* Menu / popover / tooltip */`). Use `useChartWidth()` (ResizeObserver; jsdom fallback width 600) so the SVG is drawn at its real pixel width — **never** scale a fixed viewBox (that made axis text 16 px in the mock). Axis text 11 px `fill: var(--ink-3)`; x labels: draw every `xEvery`, and the last label only if `i % xEvery >= xEvery / 2`. Add i18n keys `app.chart.chart|table|csv`. **Step 5: Run** → pass. **Step 6: Commit** — `git commit -m "feat: add core chart components"`

---

### Task 10: Chart kit — specialised charts

**Files:** `src/charts/{WeekHeatmap,CalendarHeatmap,Funnel,RetentionChart,StopHistogram,Matrix,Scatter,StackedArea,SessionTimeline,SmallMultiples}.tsx`, test `src/charts/special.test.tsx`.

**Interfaces (Produces):**

```ts
WeekHeatmap: ({ matrix: number[][] /* 7×24, Monday first */; unit: string })
CalendarHeatmap: ({ days: { date: string /* YYYY-MM-DD */; seconds: number }[] }) // Monday-first columns, month labels, 6-step seq scale
Funnel: ({ stages: { label: string; value: number }[] }) // ordinal ramp --seq-6..--seq-3, % vs first stage
RetentionChart: ({ seasons: number[][] /* viewers per episode */; onEpisodeClick?: (season: number, episode: number) => void }) // series color, season separators, retention label top-right
StopHistogram: ({ bins: number[]; runtimeMinutes: number })
Matrix: ({ rows: string[]; cols: string[]; values: (number|null)[][]; unit: string; labelWidth?: number }) // cell text class lo/hi follows theme (see design .mx-t rules)
Scatter: ({ points: { x: number; y: number; label: string; color?: string }[]; xLabel: string; yLabel: string; xFormat?; yFormat?; note?: string })
StackedArea: ({ labels: string[]; series: { name: string; color: string; values: number[] }[]; yFormat? })
SessionTimeline: ({ fromHour: number; toHour: number; rows: { user: string; spans: { start: number; end: number /* hours, decimal */; type: MediaType; label: string }[] }[] })
SmallMultiples: ({ items: { name: string; color: string; values: number[] /* 24 hours */ }[] }) // shared y scale
```

- [ ] **Step 1: Failing tests `special.test.tsx`** — for each component assert: `role="img"` with a name; correct number of cells/marks (WeekHeatmap 168 hit cells; CalendarHeatmap one per day; Funnel one bar per stage and a `%` label for stages 2+; RetentionChart one hit zone per episode and text containing `%`; Matrix renders `null` cells without a value label; SessionTimeline one rect per span; SmallMultiples one chart per item sharing the same max (assert the tallest bar of the smaller series is shorter than the tallest of the larger)); tooltip on hover for WeekHeatmap. Example:

```tsx
import { render, screen, fireEvent } from '@testing-library/react'
import { WeekHeatmap, Funnel, Matrix, SessionTimeline } from '@/charts'

it('WeekHeatmap: 7×24 focusable cells with tooltip', () => {
  const m = Array.from({ length: 7 }, () => Array.from({ length: 24 }, (_, h) => h))
  const { container } = render(<WeekHeatmap matrix={m} unit="vues" />)
  const cells = container.querySelectorAll('[data-hit]')
  expect(cells).toHaveLength(168)
  fireEvent.mouseMove(cells[23], { clientX: 1, clientY: 1 })
  expect(screen.getByRole('tooltip')).toHaveTextContent('23 h')
})
it('Funnel shows the share vs the first stage', () => {
  render(<Funnel stages={[{ label: 'Clics', value: 60 }, { label: 'Vues', value: 48 }]} />)
  expect(screen.getByText('80 %')).toBeInTheDocument()
})
it('Matrix leaves null cells empty', () => {
  const { container } = render(<Matrix rows={['HEVC']} cols={['H.264', 'HEVC']} values={[[41, null]]} unit="Lectures" />)
  expect(container.querySelectorAll('text.mx-t')).toHaveLength(1)
})
it('SessionTimeline draws one bar per span', () => {
  const { container } = render(<SessionTimeline fromHour={18} toHour={24} rows={[{ user: 'Léa', spans: [{ start: 20, end: 21, type: 'series', label: 'S2É4' }, { start: 21, end: 22, type: 'series', label: 'S2É5' }] }]} />)
  expect(container.querySelectorAll('[data-hit]')).toHaveLength(2)
})
```

- [ ] **Step 2: Run** → FAIL. **Step 3: Implement** by porting `weekHeatmap`, `calendarHeatmap` (cell width = available width / weeks, no 13 px cap), `funnel`, `dropoff` (→ RetentionChart, label at top-right), `stopHistogram`, and from the « Nouveautés » script `matrix` (with `max-width` = natural width, `.mx-t lo/hi` classes), `scatter`, `stackedArea`, `sessionTimeline`, `smallMultiples`/`columnsScaled`. Same tooltip/width rules as Task 9. Add the `.mx-t` CSS rules (light + both dark scopes) from the design file to `charts.module.css` as global selectors (`:global(.mx-t)`). **Step 4: Run** → pass. **Step 5: Commit** — `git commit -m "feat: add specialised chart components"`

---

### Task 11: App shell, AppLink, legacy wrapper, socket notifications

**Files:**
- Create: `src/app/AppLink.tsx`, `src/app/shell/{AppShell,Sidebar,TopBar,UserMenu,MobileNav,CommandPalette}.tsx` (+ modules), `src/app/shell/crumbs.ts`, `src/app/legacy/LegacyPage.tsx`, `src/lib/socket.ts` (moved from old `lib/socket.ts`), test `src/app/shell/shell.test.tsx`
- Modify: `src/lib/i18n.ts` (sync `document.documentElement.lang` on `languageChanged`)

**Interfaces:**
- Consumes: ui kit, `usePeriod`, `logout`, `getUsername`, `getUserId`, `useThemeMode`.
- Produces:
  - `AppLink: (props: LinkProps & { keepPeriod?: boolean /* default true */ })` — appends current `range/from/to/compare` to `to` when the target has none of them.
  - `type Crumb = { label: string; to?: string }` ; route `handle: { crumb?: (match: UIMatch) => Crumb | null; nav?: NavKey }` ; `useCrumbs(): Crumb[]` (from `useMatches()`; pages may override the last label via `useSetCrumbLabel(label: string)` for entity names loaded asynchronously).
  - `type NavKey = 'overview'|'live'|'history'|'libraries'|'users'|'insights'|'discover'|'playback'|'settings'`
  - `AppShell` layout route element (renders `<Outlet />` inside `.app-body`), `CommandPalette` opened with ⌘K / Ctrl+K, phase 1 commands = navigation to every nav entry + « Synchroniser » + « Basculer le thème » (search results come in phase 2).
  - `LegacyPage: ({ children })` — wraps with legacy `ThemeProvider(buildTheme(mode))`, `CssBaseline` **scoped** (`<ScopedCssBaseline>`), `LocalizationProvider(AdapterDateFns)`, `SnackbarProvider`, `PaletteProvider`, `DateRangeProvider` from the old `src/lib/*` (imported only here).
  - `useSocketEvent(event: string, handler: (data: unknown) => void)`; the shell subscribes to the task events of the old `SocketNotifier` (`PlaybackSyncTask`, `PartialSyncTask`, `FullSyncTask`, `BackupTask`, `TaskError`, `GeneralAlert`) and shows `toast.success|error|info`.

- [ ] **Step 1: Failing tests `shell.test.tsx`**

```tsx
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AppShell } from './AppShell'
import { AppLink } from '../AppLink'

function renderAt(path: string) {
  const qc = new QueryClient(); qc.setQueryData(['server-config'], { configured: true, timezone: 'Europe/Paris' })
  const router = createMemoryRouter([{ element: <AppShell />, children: [
    { path: '/', handle: { nav: 'overview' }, element: <AppLink to="/users">Utilisateurs</AppLink> },
    { path: '/users', handle: { nav: 'users', crumb: () => ({ label: 'app.nav.users' }) }, element: <p>users</p> },
  ] }], { initialEntries: [path] })
  render(<QueryClientProvider client={qc}><RouterProvider router={router} /></QueryClientProvider>)
  return router
}

it('marks the active nav entry with aria-current', () => {
  renderAt('/users')
  expect(screen.getByRole('link', { name: 'app.nav.users' })).toHaveAttribute('aria-current', 'page')
})
it('AppLink keeps the period', () => {
  renderAt('/?range=7d')
  expect(screen.getByRole('link', { name: 'Utilisateurs' })).toHaveAttribute('href', '/users?range=7d')
})
it('sidebar nav links keep the period too', () => {
  renderAt('/?range=90d')
  expect(screen.getByRole('link', { name: 'app.nav.history' })).toHaveAttribute('href', '/history?range=90d')
})
it('opens the command palette with Ctrl+K', async () => {
  renderAt('/')
  await userEvent.keyboard('{Control>}k{/Control}')
  expect(screen.getByRole('dialog', { name: 'app.command.title' })).toBeInTheDocument()
})
it('renders breadcrumbs from route handles', () => {
  renderAt('/users')
  expect(screen.getByText('app.nav.users', { selector: '[aria-current="page"]:not(a)' })).toBeInTheDocument()
})
```

- [ ] **Step 2: Run** → FAIL. **Step 3: Implement** with CSS from the design file `APP FRAME` section (lines 397–420) adapted to full-viewport layout: grid `var(--sidebar-w) 1fr`, sidebar sticky full height, top bar sticky 52 px with breadcrumbs + `PeriodPicker`, content `max-width: 1440px` padding `var(--s-5)`. Nav groups and icons (lucide): Observer (`Home` overview `/`, `Radio` live `/live` with live dot when `sessions` socket payload has ≥ 1 playing session, `History` history `/history`), Explorer (`Library` `/libraries`, `Users` `/users`), Comprendre (`BarChart3` `/insights`, `Compass` `/discover`, `Gauge` `/playback`); footer: avatar + username, `UserMenu` (Menu: theme toggle, language fr/en/es via `i18n.changeLanguage`, settings `/settings/config`, logout). Under 900 px the sidebar is hidden and `MobileNav` (Radix Dialog drawer from the left, trigger `Menu` icon in the top bar) renders the same nav. i18n keys `app.nav.*`, `app.navGroup.observe|explore|understand`, `app.command.*`, `app.user.*`. Replace `Link` imports in Task 8 components with `AppLink`. **Step 4: Run** → pass. **Step 5: Commit** — `git commit -m "feat: add app shell, navigation and command palette"`

---

### Task 12: Routes, redirects, auth guard, login, 404, entry point

**Files:**
- Create: `src/app/routes.tsx`, `src/app/App.tsx`, `src/app/RequireAuth.tsx`, `src/app/pages/LoginPage.tsx` (+ module), `src/app/pages/NotFoundPage.tsx`, `src/app/pages/ItemRedirect.tsx`, `src/app/pages/ComingSoon.tsx`, test `src/app/routes.test.tsx`
- Modify: `src/main.tsx` (render `src/app/App`, import `styles/tokens.css`, `styles/base.css`, `lib/i18n`)

**Interfaces:**
- Produces route tree (all under `RequireAuth` + `AppShell` except `/login`):

| Path | Element in phase 1 | handle.nav |
|---|---|---|
| `/` | `<LegacyPage><DashboardPage/></LegacyPage>` | overview |
| `/live` | `<ComingSoon nav="live"/>` | live |
| `/history` | `<LegacyPage><ActivityPage/></LegacyPage>`, or `TimelinePage` when `?view=calendar` | history |
| `/libraries`, `/libraries/:libraryId` | legacy `LibrariesPage`, `LibraryDetailPage` | libraries |
| `/movies/:id`, `/series/:id`, `/series/:id/episodes/:episodeId` | legacy `ItemDetailPage` via `LegacyAt` (see below) mapped to `/items/:itemId` with `itemId = episodeId ?? id` | libraries |
| `/artists/:id` | `ComingSoon` | libraries |
| `/albums/:id` | legacy `AlbumDetailPage` through the same `<Routes location>` adapter mapping to `/libraries/_/albums/:albumId` | libraries |
| `/users`, `/users/:id` | legacy `UsersPage`, `UserDetailPage` (param name `id` unchanged) | users |
| `/users/:id/recap/:period`, `/insights`, `/discover`, `/playback` | `ComingSoon` (except `/insights` → legacy `StatisticsPage` for now) | users / insights / discover / playback |
| `/settings/:section` | legacy `SettingsPage` with `?view=` set from `:section` via the `<Routes location>` adapter (`/settings?view=section`) ; `/settings` → redirect `/settings/config` ; `/settings/about` → legacy `AboutPage` | settings |
| `/login` | new `LoginPage` | — |
| `*` | `NotFoundPage` | — |

`LegacyAt` adapter (in `src/app/legacy/LegacyAt.tsx`): legacy pages read their own param names (`itemId`, `albumId`, `?view=`). Wrap them as `<LegacyAt path="/items/:itemId" location={{ pathname: `/items/${id}`, search: "" }}><ItemDetailPage /></LegacyAt>`, implemented as `<Routes location={location}><Route path={path} element={<LegacyPage>{children}</LegacyPage>} /></Routes>` so `useParams()`/`useSearchParams()` inside the legacy page see the legacy shape while the real URL stays the new one.

Redirects (`<Navigate replace>` preserving search): `/statistics → /insights`, `/activity → /history`, `/activity/timeline → /history?view=calendar`, `/libraries/:lib/albums/:id → /albums/:id`, `/about → /settings/about`, `/setup → /login`, `/settings?view=x → /settings/x`, `/items/:id` and `/libraries/:lib/items/:id` → `ItemRedirect` which calls `GET /api/items/:id` (existing endpoint returning the item; use its `Type`: `Movie → /movies/:id`, `Series → /series/:id`, `Episode → /series/:SeriesId/episodes/:id`, `MusicAlbum → /albums/:id`, `Audio → /albums/:AlbumId?track=:id`, otherwise `/movies/:id`) and renders a Skeleton while loading, `NotFoundPage` on 404.

`RequireAuth`: if `useServerConfig()` says not configured → `/login`; if no token → `/login?next={current path+search}`; else render children. `LoginPage`: port the existing `features/auth/LoginPage.tsx` behaviour (server check via `/auth/check-server`, setup wizard posting `/auth/configSetup` when unconfigured, sign-in posting `/auth/jellyfin-login` with `deviceId`/`deviceName`, 401 → invalid credentials, 403 → admin required) in the new DA: centred `Card` 400 px wide, logo, server name, two `Field`s, primary `Button` full width, `LanguageSwitcher` (Select). On success `setSession()` then navigate to `next` or `/`.

- [ ] **Step 1: Failing tests `routes.test.tsx`** (mock legacy pages with `vi.mock('@/features/dashboard/DashboardPage', () => ({ default: () => <p>legacy-dashboard</p> }))` etc. and `LegacyPage` with a passthrough):

```tsx
it.each([
  ['/statistics', '/insights'], ['/activity', '/history'], ['/activity/timeline', '/history?view=calendar'],
  ['/about', '/settings/about'], ['/setup', '/login'], ['/settings', '/settings/config'], ['/settings?view=backup', '/settings/backup'],
  ['/libraries/l1/albums/a1', '/albums/a1'],
])('redirects %s → %s', async (from, to) => {
  const router = renderApp(from) // helper: token set, server-config seeded, createMemoryRouter(routes)
  await waitFor(() => expect(router.state.location.pathname + router.state.location.search).toBe(to))
})
it('item redirect resolves the type', async () => {
  mockFetch({ '/api/items/e1': { Id: 'e1', Type: 'Episode', SeriesId: 's1' } })
  const router = renderApp('/items/e1')
  await waitFor(() => expect(router.state.location.pathname).toBe('/series/s1/episodes/e1'))
})
it('unknown path renders the 404 page', async () => {
  renderApp('/nope'); expect(await screen.findByText('app.notFound.title')).toBeInTheDocument()
})
it('unauthenticated access goes to login with next', async () => {
  const router = renderApp('/users?range=7d', { token: null })
  await waitFor(() => expect(router.state.location.pathname).toBe('/login'))
  expect(router.state.location.search).toBe('?next=%2Fusers%3Frange%3D7d')
})
it('legacy dashboard is mounted at /', async () => { renderApp('/'); expect(await screen.findByText('legacy-dashboard')).toBeInTheDocument() })
```

- [ ] **Step 2: Run** → FAIL. **Step 3: Implement** routes, `App.tsx` (`QueryClientProvider`, `Toaster`, `RouterProvider`; theme hook mounted once), `main.tsx`. i18n keys `app.login.*`, `app.notFound.*`, `app.comingSoon.*`. **Step 4: Run** `pnpm test` → all pass; `pnpm exec tsc --noEmit`; `pnpm build` → succeeds. **Step 5: Commit** — `git commit -m "feat: add routes, redirects, auth guard and login page"`

---

### Task 13: Playwright smoke tests, cleanup of replaced files

**Files:**
- Create: `frontend/playwright.config.ts`, `frontend/e2e/mockApi.ts`, `frontend/e2e/smoke.spec.ts`
- Delete: `frontend/src/App.tsx`, `frontend/src/lib/router.tsx`, `frontend/src/features/auth/` (all), `frontend/src/shared/components/Layout/` (all), `frontend/src/lib/axios.ts` **only if** no legacy page imports it (check with `grep -rn "lib/axios\|lib/publicApi" src/features`; if used, keep them for the legacy pages)

**Interfaces:** `pnpm test:e2e` runs against `vite preview` with all `/api`, `/auth`, `/stats`, `/sessions`, `/socket.io`, `/proxy` requests answered by `mockApi.ts` fixtures (empty-but-valid payloads: `[]` / `{}` / `{state:1,timezone:'Europe/Paris'}`, `/auth/jellyfin-login` → `{token: <fake jwt with userId>}`).

- [ ] **Step 1: `playwright.config.ts`**

```ts
import { defineConfig, devices } from '@playwright/test'
export default defineConfig({
  testDir: 'e2e',
  webServer: { command: 'pnpm build && pnpm preview --port 4173 --strictPort', port: 4173, reuseExistingServer: true, timeout: 180_000 },
  use: { baseURL: 'http://localhost:4173' },
  projects: [
    { name: 'desktop-light', use: { ...devices['Desktop Chrome'], viewport: { width: 1360, height: 900 }, colorScheme: 'light' } },
    { name: 'desktop-dark', use: { ...devices['Desktop Chrome'], viewport: { width: 1360, height: 900 }, colorScheme: 'dark' } },
    { name: 'mobile', use: { ...devices['Pixel 7'], colorScheme: 'light' } },
  ],
})
```

Note: `vite build` outputs to `../dist` (see `vite.config.ts`); `vite preview` serves it. If the Playwright browser is missing, run `pnpm exec playwright install chromium`.

- [ ] **Step 2: `e2e/smoke.spec.ts`**

```ts
import { test, expect } from '@playwright/test'
import { mockApi, login } from './mockApi'

const ROUTES = ['/', '/live', '/history', '/history?view=calendar', '/libraries', '/users', '/insights', '/discover', '/playback', '/settings/config', '/settings/about', '/nope']

test.beforeEach(async ({ page }) => { await mockApi(page) })

test('login page renders without errors', async ({ page }) => {
  const errors: string[] = []; page.on('pageerror', (e) => errors.push(e.message))
  await page.goto('/login')
  await expect(page.getByRole('button', { name: /se connecter|sign in|iniciar/i })).toBeVisible()
  expect(errors).toEqual([])
})

for (const route of ROUTES) {
  test(`route ${route}: shell, no console error, no horizontal overflow`, async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', (e) => errors.push(e.message))
    page.on('console', (m) => { if (m.type() === 'error') errors.push(m.text()) })
    await login(page)
    await page.goto(route)
    await expect(page.locator('nav[aria-label]').first()).toBeAttached()
    const overflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth)
    expect(overflow).toBeLessThanOrEqual(1)
    expect(errors).toEqual([])
  })
}
```

`login(page)` sets `localStorage['jellystics-token']` to a fake JWT (`header.base64({userId:'u1',userName:'Céline',isAdmin:true,sid:'s'}).sig`) via `page.addInitScript`.

- [ ] **Step 3: Run** `pnpm test:e2e` → all projects pass. Fix any page error or overflow found (legacy pages inside the new shell may need `min-width: 0` on `.app-body` children — fix in `AppShell.module.css`, not in legacy pages).

- [ ] **Step 4: Delete the replaced files**, run `pnpm exec tsc --noEmit`, `pnpm lint`, `pnpm test`, `pnpm test:e2e`, `cd ../backend && POSTGRES_PORT=55432 go test ./...` → all green.

- [ ] **Step 5: Commit** — `git commit -m "test: add playwright smoke tests and remove replaced shell"`

---

## Phases 2–7 (separate plans, written when each phase starts)

Each later phase gets its own plan file `docs/superpowers/plans/2026-11-22-frontend-phaseN-*.md`, written with the same level of detail once the previous phase is merged into the branch, so that it can rely on the real interfaces produced here:

2. **Backend v2 socle** — `/api/v2` common params middleware (`from`, `to`, `compare`, `userId`, `libraryId`, `type` → typed struct), `overview`, `history` (+ CSV, calendar), `items/:id/route`, `search`, entity endpoints (`movies`, `series`, `episodes`, `artists`, `albums`, `tracks`, `users`). Go tests per endpoint.
3. **Pages cœur** — Vue d'ensemble, En direct, Historique, Utilisateurs, Utilisateur (replace `LegacyPage` mounts).
4. **Pages d'entité** — Bibliothèques, Bibliothèque, Film, Série, Épisode, Artiste, Album.
5. **Analyses** — Insights, Découverte, Lecture & transcodage, Récap + their v2 endpoints.
6. **Capture tracker** — migration 011 and the views depending on it.
7. **Réglages + nettoyage** — settings pages in the new DA, removal of MUI and all legacy code/deps, full i18n pass, full Playwright pass.
