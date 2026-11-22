import type { UserDetail } from '@/api'

/** Compares dotted versions numerically (10.10.0 > 10.9.3). */
export function compareVersions(a: string, b: string): number {
  const pa = a.split(/[.\-+]/).map((x) => parseInt(x, 10) || 0), pb = b.split(/[.\-+]/).map((x) => parseInt(x, 10) || 0)
  for (let i = 0; i < Math.max(pa.length, pb.length); i++) {
    const d = (pa[i] ?? 0) - (pb[i] ?? 0)
    if (d) return d
  }
  return 0
}

/** Devices whose app version is older than the newest one seen for the same client. */
export function outdated(devices: UserDetail['devices']): Set<number> {
  const latest = new Map<string, string>()
  devices.forEach((d) => { if (d.version && (!latest.has(d.client) || compareVersions(d.version, latest.get(d.client)!) > 0)) latest.set(d.client, d.version) })
  const out = new Set<number>()
  devices.forEach((d, i) => { if (d.version && compareVersions(d.version, latest.get(d.client) ?? d.version) < 0) out.add(i) })
  return out
}

/** Active days and day count of [from, to) from the 365-day calendar (days after the last calendar day are ignored). */
export function activeDays(calendar: UserDetail['calendar'], from: string, to: string): { active: number; days: number } {
  const inRange = calendar.filter((d) => d.date >= from && d.date < to)
  return { active: inRange.filter((d) => d.watchSeconds > 0).length, days: inRange.length }
}

/** Busiest day of the calendar. */
export function recordDay(calendar: UserDetail['calendar']): { date: string; watchSeconds: number } | null {
  return calendar.reduce<{ date: string; watchSeconds: number } | null>((best, d) => (d.watchSeconds > (best?.watchSeconds ?? 0) ? d : best), null)
}

/** Peak index of a series (-1 when all zero). */
export function peakIndex(values: number[]): number {
  let best = -1
  values.forEach((v, i) => { if (v > 0 && (best < 0 || v > values[best])) best = i })
  return best
}
