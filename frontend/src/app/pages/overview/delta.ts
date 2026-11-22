import type { TFunction } from 'i18next'
import type { StatDelta } from '@/ui'
import type { useFormat } from '@/lib/format'

type Fmt = ReturnType<typeof useFormat>

/** Relative change vs the previous period (undefined without a usable baseline). */
export function ratioDelta(cur: number, prev: number | null | undefined, f: Fmt, higherIsGood = true): StatDelta | undefined {
  if (prev === undefined || prev === null || prev === 0) return undefined
  const d = f.delta((cur - prev) / prev)
  return { ...d, good: d.direction === 'flat' ? true : (d.direction === 'up') === higherIsGood }
}

/** Change of a ratio in percentage points. */
export function pointsDelta(cur: number | null | undefined, prev: number | null | undefined, t: TFunction): StatDelta | undefined {
  if (cur === null || cur === undefined || prev === null || prev === undefined) return undefined
  const pts = Math.round((cur - prev) * 100)
  const direction = pts > 0 ? 'up' : pts < 0 ? 'down' : 'flat'
  const text = `${pts > 0 ? '+' : pts < 0 ? '−' : ''}${t('app.overview.points', { count: Math.abs(pts) })}`
  return { text, direction, good: direction !== 'down' }
}
