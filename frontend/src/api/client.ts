import { apiFetch } from '@/lib/api'
import type { QueryParams } from './keys'

/** Base path of the v2 analytics API. */
export const V2 = '/api/v2'
/** Encodes one URL path segment (ids are opaque strings). */
export const seg = encodeURIComponent
/** React Query `queryFn` for a GET of `path` with `params` (aborted with the query). */
export const get = <T,>(path: string, params: QueryParams) => ({ signal }: { signal: AbortSignal }) => apiFetch<T>(path, { params, signal })
