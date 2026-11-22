import type { LibrariesDaily } from '@/api'

/** Categorical slots (tokens.css --cat-1..8); a 9th series folds into "other". */
export const MAX_SERIES = 8
export const OTHER = '__other__'

/**
 * Series = libraries. The MAX_SERIES biggest (by views) keep their own colour,
 * slots assigned by name so a library keeps its colour while the set is the
 * same; the rest fold into "other" (neutral). Points of folded libraries are
 * summed per date.
 */
export function librarySeries(data: LibrariesDaily, otherLabel: string) {
  const libs = data.libraries ?? []
  const fold = libs.length > MAX_SERIES
  const kept = (fold ? libs.slice(0, MAX_SERIES) : libs).slice().sort((a, b) => a.name.localeCompare(b.name) || a.id.localeCompare(b.id))
  const series = kept.map((lib, i) => ({ id: lib.id, name: lib.name, color: `var(--cat-${i + 1})` }))
  if (fold) series.push({ id: OTHER, name: otherLabel, color: 'var(--axis)' })
  const keptIds = new Set(kept.map((x) => x.id))
  const byKey = new Map<string, { date: string; seriesId: string; views: number }>()
  for (const p of data.points ?? []) {
    const sid = keptIds.has(p.libraryId) ? p.libraryId : OTHER
    if (sid === OTHER && !fold) continue
    const k = `${p.date}|${sid}`
    const cur = byKey.get(k)
    if (cur) cur.views += p.views
    else byKey.set(k, { date: p.date, seriesId: sid, views: p.views })
  }
  return { series, points: [...byKey.values()] }
}
