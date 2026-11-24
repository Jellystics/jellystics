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

/** Local calendar date of the browser (the server display zone is not exposed to the page). */
export function localToday(now = new Date()): string {
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`
}
