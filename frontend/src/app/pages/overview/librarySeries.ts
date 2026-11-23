import type { LibrariesDaily, MediaFamily } from '@/api'

export const OTHER = '__other__'
/** Slots after the three media-type colours (tokens.css --cat-4..8). */
const EXTRA_SLOTS = [4, 5, 6, 7, 8]
const TYPES: MediaFamily[] = ['movie', 'series', 'music']

/**
 * Series = libraries, coloured like the rest of the app by their type: the
 * first library (by name) of each type takes that type's token (--t-movie /
 * --t-series / --t-music = categorical slots 1-3). Every other library (a
 * second library of a type, or an "other" one) takes the next free slot
 * --cat-4..8 in fixed order, never cycled; when there are more of them than
 * slots, the most viewed keep a slot and the rest fold into "other" (neutral),
 * summed per date. Slots go by name so a library keeps its colour while the
 * set is the same.
 */
export function librarySeries(data: LibrariesDaily, otherLabel: string) {
  const libs = data.libraries ?? []
  const byName = libs.slice().sort((a, b) => a.name.localeCompare(b.name) || a.id.localeCompare(b.id))
  const colour = new Map<string, string>()
  for (const t of TYPES) {
    const first = byName.find((l) => l.type === t)
    if (first) colour.set(first.id, `var(--t-${t})`)
  }
  // Remaining libraries: the most viewed keep a slot (input is by views desc).
  const extras = libs.filter((l) => !colour.has(l.id))
  const keptExtras = new Set(extras.slice(0, EXTRA_SLOTS.length).map((l) => l.id))
  const fold = extras.length > EXTRA_SLOTS.length
  let slot = 0
  for (const l of byName) if (keptExtras.has(l.id)) colour.set(l.id, `var(--cat-${EXTRA_SLOTS[slot++]})`)

  const series = byName.filter((l) => colour.has(l.id)).map((l) => ({ id: l.id, name: l.name, color: colour.get(l.id)! }))
  if (fold) series.push({ id: OTHER, name: otherLabel, color: 'var(--axis)' })
  const byKey = new Map<string, { date: string; seriesId: string; views: number }>()
  for (const p of data.points ?? []) {
    const sid = colour.has(p.libraryId) ? p.libraryId : OTHER
    if (sid === OTHER && !fold) continue
    const k = `${p.date}|${sid}`
    const cur = byKey.get(k)
    if (cur) cur.views += p.views
    else byKey.set(k, { date: p.date, seriesId: sid, views: p.views })
  }
  return { series, points: [...byKey.values()] }
}
