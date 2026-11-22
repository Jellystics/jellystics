import { addMinutes } from 'date-fns'
import { fromZonedTime, formatInTimeZone } from 'date-fns-tz'
import type { ConcurrencyDay } from '@/api/usersExtra'
import type { JellyfinSession } from '@/api/live'
import type { MediaType } from '@/ui'
import { wallDate, wallHours } from '../history/links'
import { sessionFamily } from './toLiveSession'

/** Wall-clock label (HH:mm) of each slot of a local day; handles 92/100-slot DST days. */
export function slotLabels(date: string, count: number, minutes: number, tz: string): string[] {
  const start = fromZonedTime(`${date}T00:00:00`, tz)
  return Array.from({ length: count }, (_, i) => formatInTimeZone(addMinutes(start, i * minutes), tz, 'HH:mm'))
}

type Span = { start: number; end: number; type: MediaType; label: string }

/** Timeline rows of the day: finished sessions from /concurrency plus what is playing right now. */
export function timelineRows(day: ConcurrencyDay | undefined, live: JellyfinSession[], nowHour: number) {
  const rows = new Map<string, { user: string; spans: Span[] }>()
  const seen: { user: string; item: string; start: number; end: number }[] = []
  const add = (id: string, user: string, span: Span) => {
    const r = rows.get(id) ?? { user, spans: [] }
    r.spans.push(span)
    rows.set(id, r)
  }
  for (const s of day?.sessions ?? []) {
    const end = wallDate(s.end) !== day!.date ? 24 : wallHours(s.end)
    const span = { start: wallHours(s.start), end: Math.max(end, wallHours(s.start) + 1 / 60) }
    seen.push({ user: s.user.id, item: s.itemId, ...span })
    add(s.user.id, s.user.name || s.user.id, { start: span.start, end: span.end, type: s.kind, label: s.parentName ? `${s.parentName} · ${s.name}` : s.name })
  }
  for (const s of live) {
    const pos = (s.PlayState?.PositionTicks ?? 0) / 10_000_000 / 3600
    const it = s.NowPlayingItem!
    const start = Math.max(0, nowHour - pos), end = Math.max(nowHour, nowHour - pos + 1 / 60)
    // Already reported by /concurrency (same user + item, overlapping): skip the live duplicate.
    if (seen.some((d) => d.user === s.UserId && d.item === it.Id && d.start < end && start < d.end)) continue
    add(s.UserId, s.UserName, { start, end, type: sessionFamily(s), label: it.SeriesName ? `${it.SeriesName} · ${it.Name}` : it.Name })
  }
  const list = [...rows.values()].sort((a, b) => Math.min(...a.spans.map((s) => s.start)) - Math.min(...b.spans.map((s) => s.start)))
  const starts = list.flatMap((r) => r.spans.map((s) => s.start)), ends = list.flatMap((r) => r.spans.map((s) => s.end))
  const toHour = Math.min(24, Math.max(1, Math.ceil(ends.length ? Math.max(...ends) : nowHour)))
  const fromHour = Math.max(0, Math.min(starts.length ? Math.floor(Math.min(...starts)) : toHour - 6, toHour - 4))
  return { rows: list, fromHour, toHour }
}
