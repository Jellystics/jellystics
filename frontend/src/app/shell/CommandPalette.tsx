import { useEffect, useMemo, useState, type KeyboardEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { useLocation, useNavigate } from 'react-router-dom'
import { Clapperboard, Disc3, MicVocal, MoonStar, Music, RefreshCw, Tv, User, type LucideIcon } from 'lucide-react'
import { AppLink, Dialog, SearchInput, toast, withPeriod } from '@/ui'
import { apiFetch } from '@/lib/api'
import { useThemeMode } from '@/lib/themeMode'
import { PERIOD_PARAMS } from '@/lib/period'
import { useSearch, type SearchKind } from '@/api'
import { searchItemPath } from './searchRoutes'
import { NAV_ENTRIES } from './nav'
import styles from './CommandPalette.module.css'

type Entry = { id: string; label: string; sub?: string; icon: LucideIcon; to?: string; run?: () => void }

const SEARCH_DEBOUNCE_MS = 200

const KIND_ICON: Record<SearchKind, LucideIcon> = {
  movie: Clapperboard, series: Tv, episode: Tv, album: Disc3, artist: MicVocal, track: Music,
}

function useDebounced<T>(value: T, ms: number): T {
  const [v, setV] = useState(value)
  useEffect(() => { const h = setTimeout(() => setV(value), ms); return () => clearTimeout(h) }, [value, ms])
  return v
}

export function CommandPalette({ open, onOpenChange }: { open: boolean; onOpenChange: (o: boolean) => void }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { search } = useLocation()
  const { toggle } = useThemeMode()
  const [query, setQuery] = useState('')
  const [active, setActive] = useState(0)
  const term = useDebounced(query.trim(), SEARCH_DEBOUNCE_MS)
  const found = useSearch(open ? term : '')

  useEffect(() => { if (open) { setQuery(''); setActive(0) } }, [open])

  const commands = useMemo<Entry[]>(() => {
    const cur = new URLSearchParams(search)
    const keep = new URLSearchParams()
    PERIOD_PARAMS.forEach((k) => { const v = cur.get(k); if (v !== null) keep.set(k, v) })
    const qs = keep.toString() ? `?${keep}` : ''
    return [
      ...NAV_ENTRIES.map((e): Entry => ({ id: `nav-${e.key}`, label: t(`app.nav.${e.key}`), icon: e.icon, run: () => navigate(`${e.to}${qs}`) })),
      {
        id: 'sync', label: t('app.command.sync'), icon: RefreshCw,
        run: () => { apiFetch(`/api/runTask/${encodeURIComponent('Full Jellyfin Sync')}`, { method: 'POST' }).then(() => toast.info(t('app.command.syncStarted'))).catch(() => toast.error(t('app.command.syncFailed'))) },
      },
      { id: 'theme', label: t('app.command.toggleTheme'), icon: MoonStar, run: toggle },
    ]
  }, [t, navigate, search, toggle])

  const results = useMemo<Entry[]>(() => {
    // Only show results that match what is typed (not a stale previous term).
    if (!term || term !== query.trim() || !found.data) return []
    const items = found.data.items.flatMap((it): Entry[] => {
      const to = searchItemPath(it)
      if (!to) return []
      const sub = [t(`app.command.kind.${it.kind}`), it.parentName, it.year].filter(Boolean).join(' · ')
      return [{ id: `item-${it.kind}-${it.id}`, label: it.name, sub, icon: KIND_ICON[it.kind], to }]
    })
    const users = found.data.users.map((u): Entry => ({
      id: `user-${u.id}`, label: u.name, sub: t('app.command.kind.user'), icon: User, to: `/users/${encodeURIComponent(u.id)}`,
    }))
    return [...items, ...users]
  }, [found.data, term, query, t])

  const q = query.trim().toLowerCase()
  const shown = [...results, ...(q ? commands.filter((c) => c.label.toLowerCase().includes(q)) : commands)]
  const idx = Math.min(active, Math.max(shown.length - 1, 0))

  const close = () => onOpenChange(false)
  const exec = (e: Entry) => {
    close()
    if (e.to) navigate(withPeriod(e.to, new URLSearchParams(search)))
    else e.run?.()
  }
  const onKeyDown = (e: KeyboardEvent) => {
    if (e.key === 'ArrowDown') { e.preventDefault(); setActive((idx + 1) % Math.max(shown.length, 1)) }
    else if (e.key === 'ArrowUp') { e.preventDefault(); setActive((idx - 1 + shown.length) % Math.max(shown.length, 1)) }
    else if (e.key === 'Enter' && shown[idx]) { e.preventDefault(); exec(shown[idx]) }
  }
  const searching = !!q && (query.trim() !== term || found.isFetching) && results.length === 0

  return (
    <Dialog open={open} onOpenChange={onOpenChange} title={t('app.command.title')}>
      <div onKeyDown={onKeyDown} className={styles.wrap}>
        <SearchInput value={query} onChange={(v) => { setQuery(v); setActive(0) }} placeholder={t('app.command.placeholder')} />
        <ul className={styles.list} role="listbox" aria-label={t('app.command.title')} aria-busy={searching || undefined}>
          {shown.map((c, i) => {
            const cls = [styles.cmd, i === idx && styles.on].filter(Boolean).join(' ')
            const body = <><c.icon size={16} strokeWidth={1.6} aria-hidden /><span className={styles.label}>{c.label}</span>{c.sub && <span className={styles.sub}>{c.sub}</span>}</>
            return (
              <li key={c.id} role="option" aria-selected={i === idx}>
                {c.to
                  ? <AppLink to={c.to} tabIndex={-1} className={cls} onMouseEnter={() => setActive(i)} onClick={close}>{body}</AppLink>
                  : <button type="button" tabIndex={-1} className={cls} onMouseEnter={() => setActive(i)} onClick={() => exec(c)}>{body}</button>}
              </li>
            )
          })}
          {shown.length === 0 && <li className={styles.empty}>{t(searching ? 'app.command.searching' : 'app.command.empty')}</li>}
        </ul>
      </div>
    </Dialog>
  )
}
