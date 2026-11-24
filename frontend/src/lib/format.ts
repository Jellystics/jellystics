import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

const nf = (locale: string, opts?: Intl.NumberFormatOptions) => new Intl.NumberFormat(locale, opts)

export function formatNumber(n: number, locale = 'fr'): string { return nf(locale).format(Math.round(n)) }
export function formatCompact(n: number, locale = 'fr'): string {
  if (Math.abs(n) < 10000) return formatNumber(n, locale)
  return nf(locale, { notation: 'compact', maximumFractionDigits: 1 }).format(n)
}
export function formatPercent(ratio: number, locale = 'fr'): string {
  // Decide on the rounded value so 0.0999 reads "10 %", not "10,0 %".
  const digits = ratio !== 0 && Math.abs(Math.round(ratio * 100)) < 10 ? 1 : 0
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
/**
 * Hours on watch-time axes and tooltips: "30 min", "1 h 30", "2 h", "12,5 h". Small values switch to minutes
 * (seconds under 1 min, one decimal under 10 min) so adjacent axis ticks (0.5 h, 0.2 h… steps) never read the same.
 */
export function formatHoursAxis(hours: number, locale = 'fr'): string {
  if (!(hours > 0)) return '0 h'
  const min = hours * 60
  if (min < 1) return `${Math.round(min * 60)} s`
  if (min < 10 && Math.abs(min - Math.round(min)) > 1e-6) return `${nf(locale, { maximumFractionDigits: 1 }).format(min)} min`
  const m = Math.round(min)
  if (m < 60) return `${m} min`
  if (hours < 10) {
    const h = Math.floor(m / 60), mm = m % 60
    return mm ? `${h} h ${String(mm).padStart(2, '0')}` : `${h} h`
  }
  return `${nf(locale, { maximumFractionDigits: hours < 100 ? 1 : 0 }).format(hours)} h`
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
  // Anything below 0.05 % would display as "0,0 %": treat it as flat.
  if (Math.abs(ratio) < 0.0005) return { text: formatPercent(0, locale), direction: 'flat' }
  const body = formatPercent(Math.abs(ratio), locale)
  return ratio > 0 ? { text: `+${body}`, direction: 'up' } : { text: `−${body}`, direction: 'down' }
}
/** File size: "412 Go" / "18,2 Go" / "950 Mo" (decimal units like file managers, one decimal under 100). */
export function formatBytes(bytes: number, locale = 'fr'): string {
  const units = locale.startsWith('fr') ? ['o', 'Ko', 'Mo', 'Go', 'To'] : ['B', 'KB', 'MB', 'GB', 'TB']
  let v = Math.max(0, bytes), i = 0
  while (v >= 1000 && i < units.length - 1) { v /= 1000; i++ }
  const digits = v >= 100 || i === 0 ? 0 : 1
  return `${nf(locale, { maximumFractionDigits: digits }).format(v)} ${units[i]}`
}
export function weekdayLabels(locale = 'fr'): string[] {
  const monday = new Date(2024, 0, 1) // a Monday
  return Array.from({ length: 7 }, (_, i) => new Intl.DateTimeFormat(locale, { weekday: 'short' }).format(new Date(monday.getTime() + i * 86400000)))
}

export function useFormat() {
  const { i18n } = useTranslation()
  const l = i18n.resolvedLanguage || 'fr'
  return useMemo(() => ({
    duration: (s: number) => formatDuration(s, l), position: formatPosition, number: (n: number) => formatNumber(n, l),
    hoursAxis: (h: number) => formatHoursAxis(h, l),
    compact: (n: number) => formatCompact(n, l), percent: (r: number) => formatPercent(r, l), hour: formatHour,
    day: (d: Date) => formatDay(d, l), relative: (d: Date) => formatRelative(d, l), delta: (r: number) => formatDelta(r, l), bytes: (b: number) => formatBytes(b, l),
    weekdays: () => weekdayLabels(l), locale: l,
  }), [l])
}
