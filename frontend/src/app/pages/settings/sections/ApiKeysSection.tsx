import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import type { ColumnDef } from '@tanstack/react-table'
import { Copy, MoreVertical, Plus, Trash2 } from 'lucide-react'
import { Button, Card, CardBody, CardHeader, ConfirmDialog, DataTable, Dialog, EmptyState, ErrorState, Field, Input, Menu, toast } from '@/ui'
import { apiFetch } from '@/lib/api'
import { asArray, errorMessage } from '../net'
import styles from '../SettingsPage.module.css'

interface ApiKey { name: string; key: string; createdAt?: string }

async function copy(text: string, ok: string, fail: string) {
  try { await navigator.clipboard.writeText(text); toast.success(ok) } catch { toast.error(fail) }
}

export function ApiKeysSection() {
  const { t, i18n } = useTranslation()
  const qc = useQueryClient()
  const q = useQuery({ queryKey: ['settings', 'apikeys'], queryFn: () => apiFetch<unknown>('/api/keys') })
  const [creating, setCreating] = useState(false)
  const [name, setName] = useState('')
  const [touched, setTouched] = useState(false)
  const [created, setCreated] = useState<ApiKey | null>(null)
  const [delTarget, setDelTarget] = useState<string | null>(null)
  const refresh = () => qc.invalidateQueries({ queryKey: ['settings', 'apikeys'] })

  const create = useMutation({
    mutationFn: () => apiFetch<ApiKey>('/api/keys', { method: 'POST', body: { name: name.trim() } }),
    onSuccess: (k) => { setCreating(false); setName(''); setTouched(false); setCreated(k); refresh() },
    onError: (e) => toast.error(errorMessage(e, t('app.settings.saveError'))),
  })
  const del = useMutation({
    mutationFn: (key: string) => apiFetch('/api/keys', { method: 'DELETE', body: { key } }),
    onSuccess: () => { toast.success(t('app.settings.apikeys.deleted')); refresh() },
    onError: (e) => toast.error(errorMessage(e, t('app.settings.saveError'))),
  })

  const rows = asArray<ApiKey>(q.data).filter((k) => typeof k?.name === 'string' && typeof k?.key === 'string')
  const fmt = new Intl.DateTimeFormat(i18n.resolvedLanguage || 'fr', { dateStyle: 'medium', timeStyle: 'short' })
  const date = (s?: string) => { if (!s) return '—'; const d = new Date(s); return Number.isNaN(d.getTime()) ? s : fmt.format(d) }
  const nameError = touched && !name.trim() ? t('app.settings.apikeys.nameRequired') : undefined

  const columns: ColumnDef<ApiKey>[] = [
    { id: 'name', header: t('app.settings.apikeys.name'), cell: ({ row }) => <b>{row.original.name}</b> },
    { id: 'date', header: t('app.settings.apikeys.created'), cell: ({ row }) => date(row.original.createdAt) },
    { id: 'key', header: t('app.settings.apikeys.key'), cell: ({ row }) => <span className={styles.mono}>{row.original.key.slice(0, 8)}••••••••</span> },
    {
      id: 'actions', header: t('app.settings.action.actions'), meta: { numeric: true },
      cell: ({ row }) => (
        <Menu
          trigger={<Button variant="ghost" size="sm" iconOnly aria-label={t('app.settings.apikeys.actionsFor', { name: row.original.name })} icon={<MoreVertical size={16} strokeWidth={1.6} />} />}
          items={[
            { label: t('app.settings.apikeys.copy'), icon: <Copy size={16} strokeWidth={1.6} />, onSelect: () => copy(row.original.key, t('app.settings.apikeys.copied'), t('app.settings.saveError')) },
            'separator',
            { label: t('app.settings.action.delete'), icon: <Trash2 size={16} strokeWidth={1.6} />, danger: true, onSelect: () => setDelTarget(row.original.key) },
          ]}
        />
      ),
    },
  ]

  return (
    <>
      <Card>
        <CardHeader
          title={t('app.settings.apikeys.title')} subtitle={t('app.settings.apikeys.subtitle')}
          action={<Button variant="primary" icon={<Plus size={16} strokeWidth={1.6} />} onClick={() => setCreating(true)}>{t('app.settings.apikeys.create')}</Button>}
        />
        <CardBody flush>
          {q.isError ? <ErrorState title={t('app.error.generic')} onRetry={() => q.refetch()} /> : (
            <DataTable columns={columns} data={rows} loading={q.isLoading} empty={<EmptyState title={t('app.settings.apikeys.empty')} />} />
          )}
        </CardBody>
      </Card>

      <Dialog
        open={creating} onOpenChange={(o) => { setCreating(o); if (!o) { setName(''); setTouched(false) } }} title={t('app.settings.apikeys.create')}
        footer={<>
          <Button onClick={() => setCreating(false)}>{t('app.action.cancel')}</Button>
          <Button variant="primary" type="submit" form="apikey-form" disabled={create.isPending}>{t('app.settings.action.create')}</Button>
        </>}
      >
        <form id="apikey-form" noValidate onSubmit={(e) => { e.preventDefault(); setTouched(true); if (name.trim()) create.mutate() }}>
          <Field label={t('app.settings.apikeys.name')} htmlFor="apikey-name" error={nameError}>
            <Input id="apikey-name" autoFocus value={name} onChange={(e) => setName(e.target.value)} />
          </Field>
        </form>
      </Dialog>

      <Dialog
        open={created !== null} onOpenChange={(o) => { if (!o) setCreated(null) }} title={t('app.settings.apikeys.createdTitle')} description={t('app.settings.apikeys.createdHint')}
        footer={<>
          <Button icon={<Copy size={16} strokeWidth={1.6} />} onClick={() => created && copy(created.key, t('app.settings.apikeys.copied'), t('app.settings.saveError'))}>{t('app.settings.apikeys.copy')}</Button>
          <Button variant="primary" onClick={() => setCreated(null)}>{t('app.settings.action.done')}</Button>
        </>}
      >
        <div className={styles.secret} data-testid="new-key">{created?.key}</div>
      </Dialog>

      <ConfirmDialog
        open={delTarget !== null} onOpenChange={(o) => { if (!o) setDelTarget(null) }} danger
        title={t('app.settings.apikeys.deleteTitle')} description={t('app.settings.apikeys.deleteConfirm')}
        confirmLabel={t('app.settings.action.delete')} onConfirm={async () => { await del.mutateAsync(delTarget!) }}
      />
    </>
  )
}
