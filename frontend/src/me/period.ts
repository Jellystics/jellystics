import { addDays } from '@/lib/period'

export const ME_PRESETS = ['7d', '30d', '90d', '1y', 'all'] as const
export type MePreset = (typeof ME_PRESETS)[number]
export interface MeRange { preset: MePreset; from: string; to: string; compare: boolean }

const DAYS: Record<Exclude<MePreset, 'all'>, number> = { '7d': 7, '30d': 30, '90d': 90, '1y': 365 }

export const isPreset = (v: string | null | undefined): v is MePreset => (ME_PRESETS as readonly string[]).includes(v ?? '')

/** [from, to) of a preset ending today (local date YYYY-MM-DD); "all" starts in 2000 and is clamped server side. */
export function meRange(preset: MePreset, today: string): MeRange {
  const to = addDays(today, 1)
  if (preset === 'all') return { preset, from: '2000-01-01', to, compare: false }
  return { preset, from: addDays(today, 1 - DAYS[preset]), to, compare: true }
}

/** Local calendar date of the browser. */
export function localToday(now = new Date()): string {
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`
}

/**
 * Calendar date in the server display zone (IANA name from /api/me/profile, the zone of the admin pages), so the
 * presets cover the same days as the server; the browser date when the zone is missing or unknown.
 */
export function todayIn(timeZone: string | null | undefined, now = new Date()): string {
  if (timeZone) {
    try {
      const parts = new Intl.DateTimeFormat('en-US', { timeZone, year: 'numeric', month: '2-digit', day: '2-digit' }).formatToParts(now)
      const get = (type: string) => parts.find((x) => x.type === type)?.value
      const y = get('year'), m = get('month'), d = get('day')
      if (y && m && d) return `${y}-${m}-${d}`
    } catch { /* unknown zone: browser date */ }
  }
  return localToday(now)
}
