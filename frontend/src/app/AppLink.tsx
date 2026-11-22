import { forwardRef } from 'react'
import { Link, useLocation, type LinkProps, type To } from 'react-router-dom'
import { PERIOD_PARAMS } from '@/lib/period'

export type AppLinkProps = LinkProps & { keepPeriod?: boolean }

export function withPeriod(to: To, current: URLSearchParams): To {
  const carried = PERIOD_PARAMS.filter((k) => current.has(k))
  if (carried.length === 0) return to
  const obj = typeof to === 'string' ? splitPath(to) : { ...to }
  const params = new URLSearchParams(obj.search ?? '')
  if (PERIOD_PARAMS.some((k) => params.has(k))) return to
  carried.forEach((k) => params.set(k, current.get(k) as string))
  const search = params.toString()
  return { ...obj, search: search ? `?${search}` : '' }
}

function splitPath(to: string) {
  const hashAt = to.indexOf('#')
  const hash = hashAt >= 0 ? to.slice(hashAt) : ''
  const rest = hashAt >= 0 ? to.slice(0, hashAt) : to
  const q = rest.indexOf('?')
  return { pathname: q >= 0 ? rest.slice(0, q) : rest, search: q >= 0 ? rest.slice(q) : '', hash }
}

/** Router Link that carries the global period params (range/from/to/compare) to the target. */
export const AppLink = forwardRef<HTMLAnchorElement, AppLinkProps>(function AppLink({ to, keepPeriod = true, ...rest }, ref) {
  const { search } = useLocation()
  const target = keepPeriod && !/^[a-z][a-z0-9+.-]*:/i.test(typeof to === 'string' ? to : '') ? withPeriod(to, new URLSearchParams(search)) : to
  return <Link ref={ref} to={target} {...rest} />
})
