import { useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Upload } from 'lucide-react'
import { Badge, Button, Card, CardBody, CardHeader, EmptyState, ErrorState, Field, Input, Skeleton, Switch, toast } from '@/ui'
import { apiFetch } from '@/lib/api'
import { useSocketEvent } from '@/lib/socket'
import { asArray, enc, errorMessage, formatDateTime, uploadFile } from '../net'
import styles from '../SettingsPage.module.css'

interface Task { name: string; displayName: string; running: boolean; lastRun?: string | null }
interface TaskCfg { cronExpression?: string; enabled?: boolean }

/** Full sync always runs; the others need an explicit toggle. */
const ALWAYS_ENABLED = 'Full Jellyfin Sync'
const DEFAULT_CRON: Record<string, string> = {
  'Full Jellyfin Sync': '0 0 * * *',
  'Recently Added Sync': '0 * * * *',
  'Backup': '0 3 * * *',
  'Jellyfin Playback Reporting Plugin Sync': '0 4 * * *',
}
const CRON_RE = /^\s*\S+(\s+\S+){4}\s*$/

function TaskRow({ task, cfg, onRun }: { task: Task; cfg?: TaskCfg; onRun: () => void }) {
  const { t, i18n } = useTranslation()
  const qc = useQueryClient()
  const always = task.name === ALWAYS_ENABLED
  const [enabled, setEnabled] = useState(always || !!cfg?.enabled)
  const [cron, setCron] = useState(cfg?.cronExpression || DEFAULT_CRON[task.name] || '0 * * * *')
  const [touched, setTouched] = useState(false)
  const cronError = touched && !CRON_RE.test(cron) ? t('app.settings.tasks.cronInvalid') : undefined
  const post = (body: Record<string, unknown>) => apiFetch('/api/setTaskSettings', { method: 'POST', body: { taskname: task.name, ...body } })
  const saveCron = useMutation({
    mutationFn: () => post({ cronExpression: cron.trim() }),
    onSuccess: () => { toast.success(t('app.settings.tasks.cronSaved')); qc.invalidateQueries({ queryKey: ['settings', 'taskSettings'] }) },
    onError: (e) => toast.error(errorMessage(e, t('app.settings.saveError'))),
  })
  const toggle = useMutation({
    mutationFn: (v: boolean) => post({ enabled: v }),
    onMutate: (v) => setEnabled(v),
    onError: (e, v) => { setEnabled(!v); toast.error(errorMessage(e, t('app.settings.saveError'))) },
  })
  const id = `cron-${task.name.replace(/\W+/g, '-')}`
  return (
    <li className={styles.item}>
      <div className={styles.itemHead}>
        <div className={styles.grow}>
          <div className={styles.name}>{task.displayName}</div>
          <div className={styles.meta}>{task.lastRun ? t('app.settings.tasks.lastRun', { date: formatDateTime(task.lastRun, i18n.resolvedLanguage || 'fr') }) : t('app.settings.tasks.neverRun')}</div>
        </div>
        {!always && <Switch label={t('app.settings.tasks.schedule')} checked={enabled} onCheckedChange={(v) => toggle.mutate(v)} />}
        {task.running
          ? <Badge tone="accent">{t('app.settings.tasks.running')}</Badge>
          : <Button size="sm" onClick={onRun}>{t('app.settings.tasks.run')}</Button>}
      </div>
      {enabled && (
        <form className={styles.row} onSubmit={(e) => { e.preventDefault(); setTouched(true); if (CRON_RE.test(cron)) saveCron.mutate() }}>
          <div className={styles.grow}>
            <Field label={t('app.settings.tasks.cron')} htmlFor={id} error={cronError} hint={t('app.settings.tasks.cronHint')}>
              <Input id={id} className={styles.mono} value={cron} onChange={(e) => setCron(e.target.value)} />
            </Field>
          </div>
          <Button type="submit" size="sm" variant="primary" disabled={saveCron.isPending}>{t('app.settings.action.save')}</Button>
        </form>
      )}
    </li>
  )
}

function ImportCard() {
  const { t } = useTranslation()
  const ref = useRef<HTMLInputElement>(null)
  const [file, setFile] = useState<File | null>(null)
  const imp = useMutation({
    mutationFn: () => uploadFile<{ imported?: number }>('/sync/importPlaybackBackup', file!),
    onSuccess: (r) => { toast.success(t('app.settings.tasks.importOk', { count: r?.imported ?? 0 })); setFile(null) },
    onError: (e) => toast.error(errorMessage(e, t('app.settings.tasks.importError'))),
  })
  return (
    <Card>
      <CardHeader title={t('app.settings.tasks.importTitle')} subtitle={t('app.settings.tasks.importHint')} />
      <CardBody>
        <input ref={ref} type="file" accept=".tsv,.txt,text/tab-separated-values,text/plain" className={styles.file} data-testid="import-file" aria-label={t('app.settings.tasks.importTitle')}
          onChange={(e) => { setFile(e.target.files?.[0] ?? null); e.target.value = '' }} />
        <div className={styles.row}>
          <Button onClick={() => ref.current?.click()} disabled={imp.isPending}>{t('app.settings.action.chooseFile')}</Button>
          {file && <span className={styles.meta}>{file.name} ({(file.size / 1024).toFixed(1)} KB)</span>}
          <Button variant="primary" icon={<Upload size={16} strokeWidth={1.6} />} disabled={!file || imp.isPending} onClick={() => imp.mutate()}>{t('app.settings.tasks.import')}</Button>
        </div>
      </CardBody>
    </Card>
  )
}

export function TasksSection() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const tasks = useQuery({ queryKey: ['settings', 'tasks'], queryFn: () => apiFetch<Task[]>('/api/getTasks'), refetchInterval: 10_000 })
  const cfgs = useQuery({ queryKey: ['settings', 'taskSettings'], queryFn: () => apiFetch<Record<string, TaskCfg>>('/api/getTaskSettings') })
  const [logs, setLogs] = useState<string[]>([])
  const term = useRef<HTMLDivElement>(null)
  useSocketEvent('TaskLog', (msg) => {
    setLogs((p) => [...p.slice(-500), String(msg)])
    requestAnimationFrame(() => { if (term.current) term.current.scrollTop = term.current.scrollHeight })
  })
  const run = useMutation({
    mutationFn: (name: string) => apiFetch(`/api/runTask/${enc(name)}`, { method: 'POST' }),
    onSuccess: () => { toast.info(t('app.settings.tasks.started')); qc.invalidateQueries({ queryKey: ['settings', 'tasks'] }) },
    onError: (e) => toast.error(errorMessage(e, t('app.settings.saveError'))),
  })
  const list = asArray<Task>(tasks.data)
  const settings = cfgs.data && !Array.isArray(cfgs.data) ? cfgs.data : {}

  return (
    <div className={styles.stack}>
      <Card>
        <CardHeader title={t('app.settings.tasks.title')} subtitle={t('app.settings.tasks.subtitle')} />
        <CardBody>
          {tasks.isError ? <ErrorState title={t('app.error.generic')} onRetry={() => tasks.refetch()} />
            : tasks.isLoading || cfgs.isLoading ? <Skeleton height={120} radius="md" />
            : list.length === 0 ? <EmptyState title={t('app.settings.tasks.empty')} />
            : <ul className={styles.list} style={{ listStyle: 'none', margin: 0, padding: 0 }}>
              {list.map((task) => <TaskRow key={task.name} task={task} cfg={settings[task.name]} onRun={() => run.mutate(task.name)} />)}
            </ul>}
        </CardBody>
      </Card>
      <ImportCard />
      <Card>
        <CardHeader title={t('app.settings.tasks.output')} />
        <CardBody>
          <div ref={term} className={styles.terminal} role="log" aria-label={t('app.settings.tasks.output')}>
            {logs.length === 0 ? <span className={styles.meta}>{t('app.settings.tasks.noOutput')}</span> : logs.map((l, i) => <div key={i}>{l}</div>)}
          </div>
        </CardBody>
      </Card>
    </div>
  )
}
