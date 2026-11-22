import type { ConcurrencyDay } from '@/api/usersExtra'
import type { JellyfinSession } from '@/api/live'
import type { MediaType } from '@/ui'
import { wallDate, wallHours } from '../history/links'
import { sessionFamily } from './toLiveSession'

type Span = { start: number; end: number; type: MediaType; label: string }

/** Timeline rows of the day: finished sessions from /concurrency plus what is playing right now. */
export function timelineRows(day: ConcurrencyDay | undefined, live: JellyfinSession[], nowHour: number) {
  const rows = new Map<string, { user: string; spans: Span[] }>()
  const add = (id: string, user: string, span: Span) => {
    const r = rows.get(id) ?? { user, spans: [] }
    r.spans.push(span)
    rows.set(id, r)
  }
  for (const s of day?.sessions ?? []) {
    const end = wallDate(s.end) !== day!.date ? 24 : wallHours(s.end)
    add(s.user.id, s.user.name || s.user.id, { start: wallHours(s.start), end: Math.max(end, wallHours(s.start) + 1 / 60), type: s.kind, label: s.parentName ? `${s.parentName} · ${s.name}` : s.name })
  }
  for (const s of live) {
    const pos = (s.PlayState?.PositionTicks ?? 0) / 10_000_000 / 3600
    const it = s.NowPlayingItem!
    add(s.UserId, s.UserName, { start: Math.max(0, nowHour - pos), end: Math.max(nowHour, nowHour - pos + 1 / 60), type: sessionFamily(s), label: it.SeriesName ? `${it.SeriesName} · ${it.Name}` : it.Name })
  }
  const list = [...rows.values()].sort((a, b) => Math.min(...a.spans.map((s) => s.start)) - Math.min(...b.spans.map((s) => s.start)))
  const starts = list.flatMap((r) => r.spans.map((s) => s.start)), ends = list.flatMap((r) => r.spans.map((s) => s.end))
  const toHour = Math.min(24, Math.max(1, Math.ceil(ends.length ? Math.max(...ends) : nowHour)))
  const fromHour = Math.max(0, Math.min(starts.length ? Math.floor(Math.min(...starts)) : toHour - 6, toHour - 4))
  return { rows: list, fromHour, toHour }
}
