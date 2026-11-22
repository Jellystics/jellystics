import { useCallback, useEffect, useMemo, useState } from 'react'
import { Outlet } from 'react-router-dom'
import { toast } from '@/ui'
import socket, { useSocketEvent } from '@/lib/socket'
import { CrumbLabelContext, type CrumbLabels, type CrumbOverride } from './crumbs'
import { Sidebar } from './Sidebar'
import { TopBar } from './TopBar'
import { CommandPalette } from './CommandPalette'
import styles from './AppShell.module.css'

const TASK_EVENTS = ['PlaybackSyncTask', 'PartialSyncTask', 'FullSyncTask', 'BackupTask', 'TaskError', 'GeneralAlert']

function TaskToast({ event }: { event: string }) {
  useSocketEvent(event, (msg) => {
    const m = msg as { type?: string; message?: string } | string | null
    const text = typeof m === 'string' ? m : m?.message ?? String(m)
    const type = typeof m === 'object' && m !== null ? m.type : undefined
    if (type === 'Success') toast.success(text)
    else if (type === 'Error') toast.error(text)
    else toast.info(text)
  })
  return null
}

export function AppShell() {
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
      <div className={styles.app}>
        <Sidebar liveCount={liveCount} onOpenCommand={openPalette} />
        <div className={styles.main}>
          <TopBar liveCount={liveCount} onOpenCommand={openPalette} />
          <main className={styles.body}><Outlet /></main>
        </div>
      </div>
      <CommandPalette open={paletteOpen} onOpenChange={setPaletteOpen} />
      {TASK_EVENTS.map((e) => <TaskToast key={e} event={e} />)}
    </CrumbLabelContext.Provider>
  )
}
