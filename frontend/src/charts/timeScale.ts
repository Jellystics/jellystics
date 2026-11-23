// Time axis helpers: ticks aligned on wall-clock boundaries (hours, days,
// Mondays, months, years) of a given IANA time zone (the server's display zone).

const HOUR = 3_600_000
const DAY = 24 * HOUR

const partsFmt = new Map<string, Intl.DateTimeFormat>()
function parts(ms: number, timeZone: string | undefined) {
  const key = timeZone ?? ''
  let f = partsFmt.get(key)
  if (!f) {
    f = new Intl.DateTimeFormat('en-US', { timeZone, hourCycle: 'h23', year: 'numeric', month: 'numeric', day: 'numeric', hour: 'numeric', minute: 'numeric', second: 'numeric' })
    partsFmt.set(key, f)
  }
  const o: Record<string, number> = {}
  for (const p of f.formatToParts(ms)) if (p.type !== 'literal') o[p.type] = Number(p.value)
  return o
}

/** Offset (ms) of `timeZone` at instant `ms`: wall clock − UTC. */
export function tzOffset(ms: number, timeZone?: string): number {
  const p = parts(ms, timeZone)
  const wall = Date.UTC(p.year, p.month - 1, p.day, p.hour % 24, p.minute, p.second)
  return wall - Math.floor(ms / 1000) * 1000
}

/** Wall clock of `ms` in `timeZone`, encoded as a UTC timestamp. */
const toWall = (ms: number, tz?: string) => ms + tzOffset(ms, tz)
/** Instant of a wall-clock time (encoded as UTC) in `timeZone`. */
const fromWall = (w: number, tz?: string) => w - tzOffset(w - tzOffset(w, tz), tz)

/** Instant of local midnight of a YYYY-MM-DD date in `timeZone`. */
export function zonedDate(date: string, timeZone?: string): number {
  const [y, m, d] = date.slice(0, 10).split('-').map(Number)
  return fromWall(Date.UTC(y, (m || 1) - 1, d || 1), timeZone)
}

type Unit = 'hour' | 'day' | 'week' | 'month' | 'year'
const STEPS: { unit: Unit; n: number; ms: number }[] = [
  { unit: 'hour', n: 1, ms: HOUR }, { unit: 'hour', n: 3, ms: 3 * HOUR }, { unit: 'hour', n: 6, ms: 6 * HOUR }, { unit: 'hour', n: 12, ms: 12 * HOUR },
  { unit: 'day', n: 1, ms: DAY }, { unit: 'day', n: 2, ms: 2 * DAY },
  { unit: 'week', n: 1, ms: 7 * DAY }, { unit: 'week', n: 2, ms: 14 * DAY },
  { unit: 'month', n: 1, ms: 30 * DAY }, { unit: 'month', n: 3, ms: 91 * DAY }, { unit: 'month', n: 6, ms: 182 * DAY },
  { unit: 'year', n: 1, ms: 365 * DAY }, { unit: 'year', n: 2, ms: 730 * DAY }, { unit: 'year', n: 5, ms: 1826 * DAY }, { unit: 'year', n: 10, ms: 3652 * DAY },
]

/**
 * At most `max` ticks within [min, max] on wall-clock boundaries of
 * `timeZone`, with a label formatter suited to the step.
 */
export function timeTicks(min: number, maxMs: number, maxCount: number, locale: string, timeZone?: string): { ticks: number[]; format: (ms: number) => string } {
  const span = Math.max(1, maxMs - min)
  const step = STEPS.find((s) => span / s.ms <= Math.max(1, maxCount)) ?? STEPS[STEPS.length - 1]
  const w0 = toWall(min, timeZone), w1 = toWall(maxMs, timeZone)
  const out: number[] = []
  const push = (w: number) => { const t = fromWall(w, timeZone); if (t >= min && t <= maxMs) out.push(t) }
  if (step.unit === 'hour' || step.unit === 'day') {
    for (let w = Math.ceil(w0 / step.ms) * step.ms; w <= w1; w += step.ms) push(w)
  } else if (step.unit === 'week') {
    const d = new Date(Math.ceil(w0 / DAY) * DAY)
    let w = d.getTime() + ((8 - d.getUTCDay()) % 7) * DAY // next Monday (or today if Monday)
    for (; w <= w1; w += step.ms) push(w)
  } else {
    const months = step.unit === 'month' ? step.n : step.n * 12
    const d = new Date(w0)
    let y = d.getUTCFullYear(), m = d.getUTCMonth()
    if (Date.UTC(y, m, 1) < w0) m += 1
    const idx = y * 12 + m
    let k = Math.ceil(idx / months) * months
    for (; ; k += months) {
      y = Math.floor(k / 12); m = k % 12
      const w = Date.UTC(y, m, 1)
      if (w > w1) break
      push(w)
    }
  }
  const opts: Intl.DateTimeFormatOptions =
    step.unit === 'hour' ? { hour: '2-digit', minute: '2-digit' }
      : step.unit === 'day' || step.unit === 'week' ? { day: 'numeric', month: 'short' }
        : step.unit === 'month' ? { month: 'short', year: '2-digit' }
          : { year: 'numeric' }
  const fmt = new Intl.DateTimeFormat(locale, { ...opts, timeZone })
  const dayFmt = new Intl.DateTimeFormat(locale, { day: 'numeric', month: 'short', timeZone })
  const format = (ms: number) => {
    // Hour steps: midnight ticks name the day instead.
    if (step.unit === 'hour' && toWall(ms, timeZone) % DAY === 0) return dayFmt.format(ms)
    return fmt.format(ms)
  }
  return { ticks: out, format }
}

/** Date + time of `ms` in `timeZone` (tooltips). */
export function formatDateTime(ms: number, locale: string, timeZone?: string): string {
  return new Intl.DateTimeFormat(locale, { day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit', timeZone }).format(ms)
}
