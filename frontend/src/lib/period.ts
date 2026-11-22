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
