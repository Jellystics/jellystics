import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Pencil, Plus, Send, Trash2, Webhook as WebhookIcon } from 'lucide-react'
import { Badge, Button, Card, CardBody, CardHeader, Checkbox, ConfirmDialog, Dialog, EmptyState, ErrorState, Field, Input, Skeleton, Switch, toast } from '@/ui'
import { apiFetch } from '@/lib/api'
import { asArray, errorMessage, formatDateTime } from '../net'
import styles from '../SettingsPage.module.css'

interface Hook {
  id: number; name: string; url: string; bot_username?: string; bot_avatar_url?: string
  discord_events?: string[]; enabled: boolean; last_triggered?: string | null
}
interface Form { name: string; url: string; bot_username: string; bot_avatar_url: string; discord_events: string[] }

const EVENTS = ['task_start', 'task_complete', 'task_failed', 'backup_complete', 'api_key_created', 'api_key_deleted'] as const
const EMPTY: Form = { name: '', url: '', bot_username: 'jellystics_bot', bot_avatar_url: '', discord_events: [] }
const DISCORD_RE = /^https:\/\/(?:[\w-]+\.)?(?:discord|discordapp)\.com\/api\/webhooks\/\S+$/i

type Errors = Partial<Record<keyof Form, string>>

export function WebhooksSection() {
  const { t, i18n } = useTranslation()
  const qc = useQueryClient()
  const q = useQuery({ queryKey: ['settings', 'webhooks'], queryFn: () => apiFetch<Hook[]>('/webhooks/') })
  const [editing, setEditing] = useState<{ id: number | null } | null>(null)
  const [form, setForm] = useState<Form>(EMPTY)
  const [errors, setErrors] = useState<Errors>({})
  const [delTarget, setDelTarget] = useState<number | null>(null)
  const refresh = () => qc.invalidateQueries({ queryKey: ['settings', 'webhooks'] })
  const hooks = asArray<Hook>(q.data)

  const validate = (f: Form): Errors => {
    const e: Errors = {}
    if (!f.name.trim()) e.name = t('app.settings.webhooks.nameRequired')
    if (!DISCORD_RE.test(f.url.trim())) e.url = t('app.settings.webhooks.urlInvalid')
    if (!f.bot_username.trim()) e.bot_username = t('app.settings.webhooks.botRequired')
    if (f.discord_events.length === 0) e.discord_events = t('app.settings.webhooks.eventsRequired')
    return e
  }

  const open = (h?: Hook) => {
    setErrors({})
    setForm(h ? { name: h.name, url: h.url, bot_username: h.bot_username || 'jellystics_bot', bot_avatar_url: h.bot_avatar_url || '', discord_events: h.discord_events ?? [] } : EMPTY)
    setEditing({ id: h?.id ?? null })
  }

  const save = useMutation({
    mutationFn: () => {
      const body = { name: form.name.trim(), url: form.url.trim(), bot_username: form.bot_username.trim() || 'jellystics_bot', bot_avatar_url: form.bot_avatar_url.trim(), discord_events: form.discord_events, trigger_type: 'discord', enabled: true, method: 'POST' }
      return editing?.id != null ? apiFetch(`/webhooks/${editing.id}`, { method: 'PUT', body }) : apiFetch('/webhooks/', { method: 'POST', body })
    },
    onSuccess: () => { toast.success(t(editing?.id != null ? 'app.settings.webhooks.updated' : 'app.settings.webhooks.added')); setEditing(null); refresh() },
    onError: (e) => toast.error(errorMessage(e, t('app.settings.saveError'))),
  })
  const del = useMutation({
    mutationFn: (id: number) => apiFetch(`/webhooks/${id}`, { method: 'DELETE' }),
    onSuccess: () => { toast.success(t('app.settings.webhooks.deleted')); refresh() },
    onError: (e) => toast.error(errorMessage(e, t('app.settings.saveError'))),
  })
  const toggle = useMutation({
    mutationFn: (h: Hook) => apiFetch(`/webhooks/${h.id}`, { method: 'PUT', body: { ...h, enabled: !h.enabled } }),
    onSuccess: refresh,
    onError: (e) => toast.error(errorMessage(e, t('app.settings.saveError'))),
  })
  const test = useMutation({
    mutationFn: (h: Hook) => apiFetch(`/webhooks/${h.id}/test`, { method: 'POST', body: {} }),
    onSuccess: () => toast.success(t('app.settings.webhooks.testOk')),
    onError: () => toast.error(t('app.settings.webhooks.testFail')),
  })

  const submit = (e: FormEvent) => {
    e.preventDefault()
    const errs = validate(form); setErrors(errs)
    if (Object.keys(errs).length === 0) save.mutate()
  }
  const set = <K extends keyof Form>(k: K, v: Form[K]) => setForm((f) => ({ ...f, [k]: v }))
  const tglEvent = (ev: string, on: boolean) => set('discord_events', on ? [...form.discord_events, ev] : form.discord_events.filter((x) => x !== ev))
  const host = (u: string) => { try { return `${new URL(u).hostname}/api/webhooks/…` } catch { return '…' } }

  return (
    <>
      <Card>
        <CardHeader
          title={t('app.settings.webhooks.title')} subtitle={t('app.settings.webhooks.subtitle')}
          action={<Button variant="primary" icon={<Plus size={16} strokeWidth={1.6} />} onClick={() => open()}>{t('app.settings.webhooks.add')}</Button>}
        />
        <CardBody>
          {q.isError ? <ErrorState title={t('app.error.generic')} onRetry={() => q.refetch()} />
            : q.isLoading ? <Skeleton height={88} radius="md" />
            : hooks.length === 0 ? <EmptyState icon={<WebhookIcon size={20} strokeWidth={1.6} />} title={t('app.settings.webhooks.empty')} description={t('app.settings.webhooks.emptyHint')} />
            : <ul className={styles.list} style={{ listStyle: 'none', margin: 0, padding: 0 }}>
              {hooks.map((h) => (
                <li key={h.id} className={styles.item}>
                  <div className={styles.itemHead}>
                    <div className={styles.grow}>
                      <div className={styles.name}>{h.name} <span className={styles.meta}>· {h.bot_username || 'jellystics_bot'}</span></div>
                      <div className={`${styles.meta} ${styles.mono}`}>{host(h.url)}</div>
                    </div>
                    <Switch label={t('app.settings.webhooks.enabled')} checked={h.enabled} onCheckedChange={() => toggle.mutate(h)} />
                    <Button size="sm" variant="ghost" iconOnly aria-label={t('app.settings.webhooks.test', { name: h.name })} icon={<Send size={16} strokeWidth={1.6} />} onClick={() => test.mutate(h)} />
                    <Button size="sm" variant="ghost" iconOnly aria-label={t('app.settings.webhooks.edit', { name: h.name })} icon={<Pencil size={16} strokeWidth={1.6} />} onClick={() => open(h)} />
                    <Button size="sm" variant="ghost" iconOnly aria-label={t('app.settings.webhooks.delete', { name: h.name })} icon={<Trash2 size={16} strokeWidth={1.6} />} onClick={() => setDelTarget(h.id)} />
                  </div>
                  <div className={styles.chips}>{(h.discord_events ?? []).map((ev) => <Badge key={ev}>{t(`app.settings.webhooks.events.${ev}`)}</Badge>)}</div>
                  {h.last_triggered && <div className={styles.meta}>{t('app.settings.webhooks.lastTriggered', { date: formatDateTime(h.last_triggered, i18n.resolvedLanguage || 'fr') })}</div>}
                </li>
              ))}
            </ul>}
        </CardBody>
      </Card>

      <Dialog
        open={editing !== null} onOpenChange={(o) => { if (!o) setEditing(null) }}
        title={t(editing?.id != null ? 'app.settings.webhooks.editTitle' : 'app.settings.webhooks.addTitle')}
        footer={<>
          <Button onClick={() => setEditing(null)}>{t('app.action.cancel')}</Button>
          <Button variant="primary" type="submit" form="webhook-form" disabled={save.isPending}>{t(editing?.id != null ? 'app.settings.action.save' : 'app.settings.action.create')}</Button>
        </>}
      >
        <form id="webhook-form" className={styles.form} noValidate onSubmit={submit}>
          <Field label={t('app.settings.webhooks.name')} htmlFor="wh-name" error={errors.name}>
            <Input id="wh-name" autoFocus value={form.name} onChange={(e) => set('name', e.target.value)} />
          </Field>
          <Field label={t('app.settings.webhooks.url')} htmlFor="wh-url" error={errors.url} hint="https://discord.com/api/webhooks/…">
            <Input id="wh-url" value={form.url} onChange={(e) => set('url', e.target.value)} />
          </Field>
          <Field label={t('app.settings.webhooks.bot')} htmlFor="wh-bot" error={errors.bot_username}>
            <Input id="wh-bot" value={form.bot_username} onChange={(e) => set('bot_username', e.target.value)} />
          </Field>
          <Field label={t('app.settings.webhooks.avatar')} htmlFor="wh-avatar" hint={t('app.settings.webhooks.avatarHint')}>
            <Input id="wh-avatar" value={form.bot_avatar_url} onChange={(e) => set('bot_avatar_url', e.target.value)} />
          </Field>
          <fieldset style={{ border: 0, padding: 0, margin: 0 }}>
            <legend className={styles.name}>{t('app.settings.webhooks.notifications')}</legend>
            <div className={styles.grid2}>
              {EVENTS.map((ev) => <Checkbox key={ev} label={t(`app.settings.webhooks.events.${ev}`)} checked={form.discord_events.includes(ev)} onCheckedChange={(c) => tglEvent(ev, c)} />)}
            </div>
            {errors.discord_events && <span role="alert" className={styles.meta}>{errors.discord_events}</span>}
          </fieldset>
        </form>
      </Dialog>

      <ConfirmDialog
        open={delTarget !== null} onOpenChange={(o) => { if (!o) setDelTarget(null) }} danger
        title={t('app.settings.webhooks.deleteTitle')} description={t('app.settings.webhooks.deleteConfirm')}
        confirmLabel={t('app.settings.action.delete')} onConfirm={async () => { await del.mutateAsync(delTarget!) }}
      />
    </>
  )
}
