/** Wall-clock hour (fractional) of an ISO time with offset, as written by the server (its time zone). */
export function wallHours(iso: string): number {
  const m = /T(\d{2}):(\d{2})/.exec(iso)
  return m ? Number(m[1]) + Number(m[2]) / 60 : 0
}
/** Wall-clock HH:MM of an ISO time with offset. */
export function wallClock(iso: string): string {
  return /T(\d{2}:\d{2})/.exec(iso)?.[1] ?? ''
}
/** Local calendar date (YYYY-MM-DD) of an ISO time with offset. */
export const wallDate = (iso: string) => iso.slice(0, 10)
/** Parses YYYY-MM-DD as a local Date (for display formatting only). */
export function dateOf(s: string): Date {
  const [y, m, d] = s.slice(0, 10).split('-').map(Number)
  return new Date(y, (m || 1) - 1, d || 1)
}

/** YYYY-MM shifted by n months. */
export function shiftMonth(month: string, n: number): string {
  const [y, m] = month.split('-').map(Number)
  const d = new Date(Date.UTC(y, m - 1 + n, 1))
  return `${d.getUTCFullYear()}-${String(d.getUTCMonth() + 1).padStart(2, '0')}`
}
