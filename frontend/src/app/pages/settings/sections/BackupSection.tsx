import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import type { ColumnDef } from '@tanstack/react-table'
import { Download, MoreVertical, RotateCcw, Trash2, Upload } from 'lucide-react'
import { Button, Card, CardBody, CardHeader, ConfirmDialog, DataTable, EmptyState, ErrorState, Field, Input, Menu, toast } from '@/ui'
import { apiFetch } from '@/lib/api'
import { asArray, downloadAuthed, enc, errorMessage, uploadFile } from '../net'
import styles from '../SettingsPage.module.css'

interface BackupRaw { name: string; size: number; datecreated: string }

function KeepLogsCard() {
  const { t } = useTranslation()
  const cfg = useQuery({ queryKey: ['settings', 'config'], queryFn: () => apiFetch<{ settings?: { KeepLogsForDays?: number } }>('/api/getconfig') })
  const [days, setDays] = useState('30')
  useEffect(() => { const d = cfg.data?.settings?.KeepLogsForDays; if (d) setDays(String(d)) }, [cfg.data])
  const n = Number(days)
  const valid = Number.isInteger(n) && n >= 1
  const save = useMutation({
    mutationFn: () => apiFetch('/api/setconfig', { method: 'POST', body: { KeepLogsForDays: n } }),
    onSuccess: () => toast.success(t('app.settings.saved')),
    onError: (e) => toast.error(errorMessage(e, t('app.settings.saveError'))),
  })
  return (
    <Card>
      <CardHeader title={t('app.settings.backup.keepTitle')} subtitle={t('app.settings.backup.keepHint')} />
      <CardBody>
        <form className={styles.row} onSubmit={(e) => { e.preventDefault(); if (valid) save.mutate() }}>
          <Field label={t('app.settings.backup.keepDays')} htmlFor="keep-days" error={valid ? undefined : t('app.settings.backup.keepInvalid')}>
            <Input id="keep-days" type="number" min={1} value={days} onChange={(e) => setDays(e.target.value)} style={{ width: 120 }} />
          </Field>
          <Button type="submit" variant="primary" disabled={!valid || save.isPending}>{t('app.settings.action.save')}</Button>
        </form>
      </CardBody>
    </Card>
  )
}

export function BackupSection() {
  const { t, i18n } = useTranslation()
  const qc = useQueryClient()
  const fileRef = useRef<HTMLInputElement>(null)
  const [delTarget, setDelTarget] = useState<string | null>(null)
  const [restoreTarget, setRestoreTarget] = useState<string | null>(null)
  const files = useQuery({ queryKey: ['settings', 'backups'], queryFn: () => apiFetch<BackupRaw[]>('/backup/files') })
  const refresh = () => qc.invalidateQueries({ queryKey: ['settings', 'backups'] })

  const create = useMutation({
    mutationFn: () => apiFetch('/backup/beginBackup'),
    onSuccess: () => { toast.success(t('app.settings.backup.created')); refresh() },
    onError: (e) => toast.error(errorMessage(e, t('app.settings.saveError'))),
  })
  const upload = useMutation({
    mutationFn: (f: File) => uploadFile('/backup/upload', f),
    onSuccess: () => { toast.success(t('app.settings.backup.uploaded')); refresh() },
    onError: (e) => toast.error(errorMessage(e, t('app.settings.backup.uploadError'))),
  })
  const del = useMutation({
    mutationFn: (name: string) => apiFetch(`/backup/files/${enc(name)}`, { method: 'DELETE' }),
    onSuccess: () => { toast.success(t('app.settings.backup.deleted')); refresh() },
    onError: (e) => toast.error(errorMessage(e, t('app.settings.saveError'))),
  })
  const restore = useMutation({
    mutationFn: (name: string) => apiFetch<{ restored?: Record<string, number> }>(`/backup/restore/${enc(name)}`),
    onSuccess: (r) => {
      const total = Object.values(r?.restored ?? {}).reduce((s, v) => s + Number(v || 0), 0)
      toast.success(t('app.settings.backup.restored', { count: total }))
      qc.invalidateQueries()
    },
    onError: (e) => toast.error(errorMessage(e, t('app.settings.backup.restoreError'))),
  })
  const download = async (name: string) => {
    try { await downloadAuthed(`/backup/files/${enc(name)}`, name) } catch { toast.error(t('app.settings.saveError')) }
  }

  const rows = asArray<BackupRaw>(files.data)
  const fmt = new Intl.DateTimeFormat(i18n.resolvedLanguage || 'fr', { dateStyle: 'medium', timeStyle: 'short' })
  const date = (s: string) => { const d = new Date(s); return Number.isNaN(d.getTime()) ? s : fmt.format(d) }
  const size = (b: number) => b >= 1048576 ? `${(b / 1048576).toFixed(1)} ${t('app.settings.backup.mb')}` : `${(b / 1024).toFixed(1)} ${t('app.settings.backup.kb')}`

  const columns: ColumnDef<BackupRaw>[] = [
    { id: 'name', header: t('app.settings.backup.name'), cell: ({ row }) => <span className={styles.mono}>{row.original.name}</span> },
    { id: 'date', header: t('app.settings.backup.date'), cell: ({ row }) => date(row.original.datecreated) },
    { id: 'size', header: t('app.settings.backup.size'), meta: { numeric: true }, cell: ({ row }) => size(row.original.size) },
    {
      id: 'actions', header: t('app.settings.action.actions'), meta: { numeric: true },
      cell: ({ row }) => (
        <Menu
          trigger={<Button variant="ghost" size="sm" iconOnly aria-label={t('app.settings.backup.actionsFor', { name: row.original.name })} icon={<MoreVertical size={16} strokeWidth={1.6} />} />}
          items={[
            { label: t('app.settings.action.download'), icon: <Download size={16} strokeWidth={1.6} />, onSelect: () => download(row.original.name) },
            { label: t('app.settings.backup.restore'), icon: <RotateCcw size={16} strokeWidth={1.6} />, onSelect: () => setRestoreTarget(row.original.name) },
            'separator',
            { label: t('app.settings.action.delete'), icon: <Trash2 size={16} strokeWidth={1.6} />, danger: true, onSelect: () => setDelTarget(row.original.name) },
          ]}
        />
      ),
    },
  ]

  return (
    <div className={styles.stack}>
      <Card>
        <CardHeader
          title={t('app.settings.backup.title')}
          subtitle={t('app.settings.backup.subtitle')}
          action={
            <div className={styles.actions}>
              <input ref={fileRef} type="file" accept=".json,application/json" className={styles.file} data-testid="backup-file" aria-label={t('app.settings.backup.upload')}
                onChange={(e) => { const f = e.target.files?.[0]; e.target.value = ''; if (f) upload.mutate(f) }} />
              <Button icon={<Upload size={16} strokeWidth={1.6} />} disabled={upload.isPending} onClick={() => fileRef.current?.click()}>{t('app.settings.backup.upload')}</Button>
              <Button variant="primary" disabled={create.isPending} onClick={() => create.mutate()}>{create.isPending ? t('app.settings.backup.creating') : t('app.settings.backup.create')}</Button>
            </div>
          }
        />
        <CardBody flush>
          {files.isError ? <ErrorState title={t('app.error.generic')} onRetry={() => files.refetch()} /> : (
            <DataTable columns={columns} data={rows} loading={files.isLoading} empty={<EmptyState title={t('app.settings.backup.empty')} />} />
          )}
        </CardBody>
      </Card>
      <KeepLogsCard />
      <ConfirmDialog
        open={delTarget !== null} onOpenChange={(o) => { if (!o) setDelTarget(null) }} danger
        title={t('app.settings.backup.deleteTitle')} description={t('app.settings.backup.deleteConfirm', { name: delTarget ?? '' })}
        confirmLabel={t('app.settings.action.delete')} onConfirm={async () => { await del.mutateAsync(delTarget!) }}
      />
      <ConfirmDialog
        open={restoreTarget !== null} onOpenChange={(o) => { if (!o) setRestoreTarget(null) }} danger
        title={t('app.settings.backup.restoreTitle')} description={t('app.settings.backup.restoreConfirm', { name: restoreTarget ?? '' })}
        confirmLabel={t('app.settings.backup.restore')} onConfirm={async () => { await restore.mutateAsync(restoreTarget!) }}
      />
    </div>
  )
}
