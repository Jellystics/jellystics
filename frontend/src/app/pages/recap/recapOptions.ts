/** Period switch options: the year, then up to three months of it (latest first, never in the future). */
export function recapOptions(period: string, now = new Date()): string[] {
  const year = Number(period.slice(0, 4))
  const lastMonth = year === now.getFullYear() ? now.getMonth() + 1 : year < now.getFullYear() ? 12 : 0
  const months = Array.from({ length: Math.min(3, lastMonth) }, (_, i) => `${year}-${String(lastMonth - i).padStart(2, '0')}`)
  if (period.length === 7 && !months.includes(period)) months.push(period)
  return [String(year), ...months]
}
