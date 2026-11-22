import '@testing-library/jest-dom/vitest'
import { createElement } from 'react'

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

vi.mock('react-i18next', () => ({
  useTranslation: () => ({ t: (k: string, o?: Record<string, unknown>) => (o && 'count' in o ? `${k}:${o.count}` : k), i18n: { resolvedLanguage: 'fr', language: 'fr', changeLanguage: vi.fn() } }),
  Trans: ({ i18nKey, values, children }: { i18nKey?: string; values?: Record<string, string>; children?: React.ReactNode }) =>
    children ?? createElement('span', { 'data-i18n': i18nKey }, ...Object.values(values ?? {}).map((v) => createElement('strong', { key: v }, v))),
  initReactI18next: { type: '3rdParty', init: () => {} },
}))

// jsdom's AbortSignal is rejected by Node's undici Request, which React Router builds on every
// navigation. Drop the signal in tests (nothing here relies on request cancellation).
const NativeRequest = globalThis.Request
globalThis.Request = class extends NativeRequest {
  constructor(input: RequestInfo | URL, init?: RequestInit) {
    super(input, init ? { ...init, signal: undefined } : init)
  }
} as typeof Request
