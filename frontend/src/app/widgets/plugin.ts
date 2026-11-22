// Visibility rules of the plugin-fed widgets (PluginWidgets.tsx).
import type { PlayedToEnd } from '@/api'
import type { ItemMarks } from '@/api/plugin'
import { usePeriod } from '@/lib/useUrlState'

/** True when the plugin data starts after the start of the URL period. */
export function usePluginPartial(since: string | null | undefined): boolean {
  const [period] = usePeriod()
  return !!since && since > period.from
}

/** True when the period has plays measured by the plugin. */
export const hasPlayedToEnd = (d: PlayedToEnd | undefined): d is PlayedToEnd => !!d && d.measured > 0

/** True when the plugin recorded played / unplayed events for the item. */
export const hasMarks = (d: ItemMarks | undefined): d is ItemMarks => !!d?.collectedSince && d.manual + d.automatic + d.other + d.unplayed > 0
