import { useEffect, useState } from 'react'
import { todayIn } from './period'

/** Today's date in `tz`, re-evaluated when the day rolls over (and when the tab becomes visible again). */
export function useToday(tz: string): string {
  const [today, setToday] = useState(() => todayIn(tz))
  useEffect(() => {
    const refresh = () => setToday(todayIn(tz))
    refresh()
    const id = window.setInterval(refresh, 60_000)
    const onVisible = () => { if (document.visibilityState === 'visible') refresh() }
    document.addEventListener('visibilitychange', onVisible)
    return () => { window.clearInterval(id); document.removeEventListener('visibilitychange', onVisible) }
  }, [tz])
  return today
}
