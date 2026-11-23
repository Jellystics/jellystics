// Formatting helpers of the analysis pages.

/** Bit/s → "8,5" (Mb/s, unit added by the caller's translation). */
export function formatMbps(bps: number, locale = 'fr'): string {
  const mb = bps / 1_000_000
  return new Intl.NumberFormat(locale, { maximumFractionDigits: mb < 10 ? 1 : 0 }).format(mb)
}

/** Bit/s → "8,5 Mb/s". */
export function formatBitrate(bps: number, locale = 'fr'): string {
  return `${formatMbps(bps, locale)} Mb/s`
}

/** Hours with one decimal under 10 (y axes of watch-time charts). */
export function formatHours(seconds: number, locale = 'fr'): string {
  const h = seconds / 3600
  return `${new Intl.NumberFormat(locale, { maximumFractionDigits: h < 10 ? 1 : 0 }).format(h)} h`
}

/** "2026-11" → "nov." (+ year when it is not the current one). */
export function formatMonth(month: string, locale = 'fr', withYear = false): string {
  const [y, m] = month.split('-').map(Number)
  const d = new Date(y, m - 1, 1)
  return new Intl.DateTimeFormat(locale, { month: 'short', ...(withYear || y !== new Date().getFullYear() ? { year: '2-digit' } : {}) }).format(d)
}

