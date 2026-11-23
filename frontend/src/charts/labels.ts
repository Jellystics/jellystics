/** Full weekday names, Monday first. */
export function weekdayLong(locale = 'fr'): string[] {
  return Array.from({ length: 7 }, (_, i) => new Intl.DateTimeFormat(locale, { weekday: 'long' }).format(new Date(2024, 0, 1 + i)))
}

/** Index 0..5 (or 0..steps-1) of a sequential step for v within [0, max]; 0 means "none". */
export function seqIndex(v: number, max: number, steps: number): number {
  if (!(v > 0) || !(max > 0)) return 0
  return Math.min(steps - 1, 1 + Math.floor((v / max) * (steps - 1 - 0.001)))
}

export const SEQ = [0, 1, 2, 3, 4, 5, 6].map((i) => `var(--seq-${i})`)
