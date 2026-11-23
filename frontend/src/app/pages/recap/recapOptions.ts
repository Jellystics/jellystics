/** Current year and month (1–12) in `timeZone` (the server's display zone). */
function yearMonthIn(now: Date, timeZone?: string): { year: number; month: number } {
  try {
    const parts = new Intl.DateTimeFormat('en-US', { timeZone, year: 'numeric', month: 'numeric' }).formatToParts(now)
    const get = (type: string) => Number(parts.find((p) => p.type === type)?.value)
    const year = get('year'), month = get('month')
    if (year && month) return { year, month }
  } catch {
    // Unknown zone: fall back to the browser clock.
  }
  return { year: now.getFullYear(), month: now.getMonth() + 1 }
}

/**
 * Period switch options: the year, then up to three months of it (latest
 * first, never in the future of the server time zone).
 */
export function recapOptions(period: string, now = new Date(), timeZone?: string): string[] {
  const year = Number(period.slice(0, 4))
  const cur = yearMonthIn(now, timeZone)
  const lastMonth = year === cur.year ? cur.month : year < cur.year ? 12 : 0
  const months = Array.from({ length: Math.min(3, lastMonth) }, (_, i) => `${year}-${String(lastMonth - i).padStart(2, '0')}`)
  if (period.length === 7 && !months.includes(period)) months.push(period)
  return [String(year), ...months]
}
