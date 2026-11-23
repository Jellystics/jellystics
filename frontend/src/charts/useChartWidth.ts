import { useLayoutEffect, useRef, useState, type RefObject } from 'react'

export function useChartWidth(): [RefObject<HTMLDivElement | null>, number] {
  const ref = useRef<HTMLDivElement>(null)
  const [width, setWidth] = useState(600)
  useLayoutEffect(() => {
    const el = ref.current
    if (!el) return
    const measure = () => { const w = Math.round(el.getBoundingClientRect().width); if (w > 0) setWidth(w) }
    measure()
    const ro = new ResizeObserver(measure)
    ro.observe(el)
    return () => ro.disconnect()
  }, [])
  return [ref, width]
}
