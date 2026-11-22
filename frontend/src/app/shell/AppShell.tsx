import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Outlet } from 'react-router-dom'
import { toast } from '@/ui'
import socket, { useSocketEvent } from '@/lib/socket'
import { CrumbLabelContext, type CrumbLabels, type CrumbOverride } from './crumbs'
import { Sidebar } from './Sidebar'
import { TopBar } from './TopBar'
import { CommandPalette } from './CommandPalette'
import styles from './AppShell.module.css'

const TASK_EVENTS = ['PlaybackSyncTask', 'PartialSyncTask', 'FullSyncTask', 'BackupTask', 'TaskError', 'GeneralAlert']

/** One instance for all task events: subscribes each event on the shared socket. */
function TaskToast() {
  const handler = useCallback((msg: unknown) => {
    const m = msg as { type?: string; message?: string } | string | null
    const text = typeof m === 'string' ? m : m?.message ?? String(m)
    const type = typeof m === 'object' && m !== null ? m.type : undefined
    if (type === 'Success') toast.success(text)
    else if (type === 'Error') toast.error(text)
    else toast.info(text)
  }, [])
  useSocketEvent(TASK_EVENTS, handler)
  return null
}

export function AppShell() {
  const { t } = useTranslation()
  const mainRef = useRef<HTMLElement>(null)
  const [paletteOpen, setPaletteOpen] = useState(false)
  const [liveCount, setLiveCount] = useState(0)
  const [label, setLabel] = useState<CrumbOverride>(null)
  const [labels, setLabels] = useState<CrumbLabels>({})
  const crumbCtx = useMemo(() => ({ label, set: setLabel, labels, setLabels }), [label, labels])

  useEffect(() => {
    socket.connect()
    return () => { socket.disconnect() }
  }, [])

  useSocketEvent('sessions', useCallback((data: unknown) => {
    if (Array.isArray(data)) setLiveCount(data.filter((s) => (s as { NowPlayingItem?: unknown })?.NowPlayingItem).length)
  }, []))

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') { e.preventDefault(); setPaletteOpen((o) => !o) }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [])

  const openPalette = useCallback(() => setPaletteOpen(true), [])

  return (
    <CrumbLabelContext.Provider value={crumbCtx}>
      <a href="#main" className={styles.skip} onClick={(e) => { e.preventDefault(); mainRef.current?.focus() }}>{t('app.nav.skipToContent')}</a>
      <div className={styles.app}>
        <Sidebar liveCount={liveCount} onOpenCommand={openPalette} />
        <div className={styles.main}>
          <TopBar liveCount={liveCount} onOpenCommand={openPalette} />
          <main id="main" tabIndex={-1} ref={mainRef} className={styles.body}><Outlet /></main>
        </div>
      </div>
      <CommandPalette open={paletteOpen} onOpenChange={setPaletteOpen} />
      <TaskToast />
    </CrumbLabelContext.Provider>
  )
}
