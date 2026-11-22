export function niceMax(v: number): number {
  if (!(v > 0)) return 1
  const p = 10 ** Math.floor(Math.log10(v))
  const n = v / p
  return (n <= 1 ? 1 : n <= 2 ? 2 : n <= 2.5 ? 2.5 : n <= 5 ? 5 : 10) * p
}

export function ticks(max: number, count = 5): number[] {
  return Array.from({ length: count + 1 }, (_, k) => (max * k) / count)
}

/** Rounded value end (top), square baseline. */
export function barPath(x: number, y: number, w: number, h: number, r = 4): string {
  if (h <= 0) return ''
  r = Math.min(r, h, w / 2)
  return `M${x},${y + h}V${y + r}Q${x},${y} ${x + r},${y}H${x + w - r}Q${x + w},${y} ${x + w},${y + r}V${y + h}Z`
}

/** Horizontal bar: rounded right end, square baseline on the left. */
export function barPathH(x: number, y: number, w: number, h: number, r = 4): string {
  if (w <= 0) return ''
  r = Math.min(r, w, h / 2)
  return `M${x},${y}H${x + w - r}Q${x + w},${y} ${x + w},${y + r}V${y + h - r}Q${x + w},${y + h} ${x + w - r},${y + h}H${x}Z`
}

export function linePath(points: [number, number][]): string {
  return points.map(([x, y], i) => `${i ? 'L' : 'M'}${+x.toFixed(1)},${+y.toFixed(1)}`).join('')
}

const formatters = new Map<string, Intl.NumberFormat>()
export function defaultFormat(v: number, locale = 'fr'): string {
  let f = formatters.get(locale)
  if (!f) { f = new Intl.NumberFormat(locale, { notation: 'compact', maximumFractionDigits: 1 }); formatters.set(locale, f) }
  return f.format(v)
}

export const showXLabel = (i: number, n: number, every: number): boolean =>
  i % every === 0 || (i === n - 1 && i % every >= every / 2)

/** Clamp a 0..1 ratio so an out-of-range value (e.g. a corrupt negative) never draws outside the plot. */
export const clamp01 = (r: number): number => (Number.isFinite(r) ? Math.min(1, Math.max(0, r)) : 0)
