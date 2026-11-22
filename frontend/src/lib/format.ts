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
