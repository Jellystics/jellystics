import { useCallback, useMemo } from 'react'
import { useSearchParams } from 'react-router-dom'
import { parsePeriod, serializePeriod, todayIn, PERIOD_PARAMS, type Period } from './period'
import { useServerConfig } from './serverConfig'

export function usePeriod(): [Period, (next: Partial<Period>) => void] {
  const [params, setParams] = useSearchParams()
  const tz = useServerConfig().data?.timezone ?? Intl.DateTimeFormat().resolvedOptions().timeZone
  const today = todayIn(tz)
  const period = useMemo(() => parsePeriod(params, today), [params, today])
  const update = useCallback((next: Partial<Period>) => {
    setParams((prev) => {
      const merged: Period = { ...parsePeriod(prev, today), ...next }
      const out = new URLSearchParams(prev)
      PERIOD_PARAMS.forEach((k) => out.delete(k))
      Object.entries(serializePeriod(merged)).forEach(([k, v]) => out.set(k, v))
      return out
    }, { replace: false })
  }, [setParams, today])
  return [period, update]
}

export function useSearchParam(key: string, fallback?: string, opts: { push?: boolean } = {}): [string | undefined, (v?: string) => void] {
  const [params, setParams] = useSearchParams()
  const value = params.get(key) ?? fallback
  const set = useCallback((v?: string) => {
    setParams((prev) => {
      const out = new URLSearchParams(prev)
      if (v === undefined || v === '' || v === fallback) out.delete(key); else out.set(key, v)
      return out
    }, { replace: !opts.push })
  }, [key, fallback, opts.push, setParams])
  return [value, set]
}
