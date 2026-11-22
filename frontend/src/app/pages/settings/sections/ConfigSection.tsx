import { useEffect, useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Card, CardBody, CardFooter, CardHeader, Button, ErrorState, Field, Input, Skeleton, Switch, toast } from '@/ui'
import { apiFetch } from '@/lib/api'
import { useServerConfig } from '@/lib/serverConfig'
import { asArray, errorMessage } from '../net'
import styles from '../SettingsPage.module.css'
import { DataQualityCard, PluginCard } from './PluginCards'

interface ConfigResp { JF_HOST?: string; app_url?: string; settings?: Record<string, unknown> }
interface Validation { isValid: boolean; serverName?: string; errorMessage?: string }
interface TrackedLibrary { Id: string; Name?: string | null; Tracked: boolean }
interface UserLite { Id: string; Name?: string | null }

const URL_RE = /^https?:\/\/\S+$/i

function ConnectionCard() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const q = useQuery({ queryKey: ['settings', 'config'], queryFn: () => apiFetch<ConfigResp>('/api/getconfig') })
  const [host, setHost] = useState('')
  const [apiKey, setApiKey] = useState('')
  const [appUrl, setAppUrl] = useState('')
  const [touched, setTouched] = useState(false)
  const [check, setCheck] = useState<Validation | null>(null)
  const data = q.data

  useEffect(() => {
    if (!data || Array.isArray(data)) return
    setHost(data.JF_HOST ?? ''); setAppUrl(data.app_url ?? '')
  }, [data])

  const hostError = touched && !URL_RE.test(host.trim()) ? t('app.settings.config.urlInvalid') : undefined

  const save = useMutation({
    mutationFn: () => apiFetch('/api/setconfig', { method: 'POST', body: { JF_HOST: host.trim(), JF_API_KEY: apiKey, app_url: appUrl.trim() } }),
    onSuccess: () => { toast.success(t('app.settings.config.saved')); setApiKey(''); qc.invalidateQueries({ queryKey: ['settings', 'config'] }) },
    onError: (e) => toast.error(errorMessage(e, t('app.settings.saveError'))),
  })
  const test = useMutation({
    mutationFn: () => apiFetch<Validation>('/proxy/validateSettings', { method: 'POST', body: { url: host.trim(), apikey: apiKey } }),
    onSuccess: setCheck,
    onError: (e) => setCheck({ isValid: false, errorMessage: errorMessage(e, t('app.settings.saveError')) }),
  })

  const submit = (e: FormEvent) => {
    e.preventDefault(); setTouched(true)
    if (!URL_RE.test(host.trim())) return
    save.mutate()
  }

  if (q.isError) return <ErrorState title={t('app.error.generic')} onRetry={() => q.refetch()} />
  return (
    <Card>
      <CardHeader title={t('app.settings.config.jellyfin')} subtitle={t('app.settings.config.jellyfinHint')} />
      <form onSubmit={submit} noValidate>
        <CardBody>
          {q.isLoading ? <Skeleton height={96} radius="md" /> : (
            <div className={styles.form}>
              <Field label={t('app.settings.config.url')} htmlFor="cfg-url" error={hostError} hint="http://192.168.1.10:8096">
                <Input id="cfg-url" value={host} onChange={(e) => { setHost(e.target.value); setCheck(null) }} onBlur={() => setTouched(true)} />
              </Field>
              <Field label={t('app.settings.config.apiKey')} htmlFor="cfg-key" hint={t('app.settings.config.apiKeyHint')}>
                <Input id="cfg-key" type="password" autoComplete="off" value={apiKey} onChange={(e) => { setApiKey(e.target.value); setCheck(null) }} />
              </Field>
              <Field label={t('app.settings.config.appUrl')} htmlFor="cfg-app" hint={t('app.settings.config.appUrlHint')}>
                <Input id="cfg-app" value={appUrl} placeholder="https://jellystics.example.com" onChange={(e) => setAppUrl(e.target.value)} />
              </Field>
              {check && (
                <span className={styles.status} role="status">
                  {check.isValid ? t('app.settings.config.testOk', { name: check.serverName ?? '' }) : t('app.settings.config.testFail', { message: check.errorMessage ?? '' })}
                </span>
              )}
            </div>
          )}
        </CardBody>
        <CardFooter>
          <Button type="button" disabled={!URL_RE.test(host.trim()) || !apiKey || test.isPending} onClick={() => test.mutate()}>{t('app.settings.config.test')}</Button>
          <Button type="submit" variant="primary" disabled={save.isPending}>{t('app.settings.action.save')}</Button>
        </CardFooter>
      </form>
    </Card>
  )
}

function TrackingCard() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const libs = useQuery({ queryKey: ['settings', 'libraries'], queryFn: () => apiFetch<TrackedLibrary[]>('/api/TrackedLibraries') })
  const users = useQuery({ queryKey: ['settings', 'users'], queryFn: () => apiFetch<UserLite[]>('/api/users') })
  const untracked = useQuery({ queryKey: ['settings', 'untracked'], queryFn: () => apiFetch<string[]>('/api/UntrackedUsers') })
  const toggleLib = useMutation({
    mutationFn: (id: string) => apiFetch('/api/setExcludedLibraries', { method: 'POST', body: { libraryID: id } }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['settings', 'libraries'] }),
    onError: (e) => toast.error(errorMessage(e, t('app.settings.saveError'))),
  })
  const toggleUser = useMutation({
    mutationFn: (id: string) => apiFetch('/api/setUntrackedUsers', { method: 'POST', body: { userId: id } }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['settings', 'untracked'] }),
    onError: (e) => toast.error(errorMessage(e, t('app.settings.saveError'))),
  })
  const libList = asArray<TrackedLibrary>(libs.data)
  const userList = asArray<UserLite>(users.data)
  const excluded = new Set(asArray<string>(untracked.data))

  return (
    <Card>
      <CardHeader title={t('app.settings.config.tracking')} subtitle={t('app.settings.config.trackingHint')} />
      <CardBody>
        {libs.isLoading || users.isLoading ? <Skeleton height={80} radius="md" /> : (
          <div className={styles.stack}>
            <div>
              <div className={styles.name}>{t('app.settings.config.libraries')}</div>
              {libList.length === 0 ? <span className={styles.meta}>{t('app.settings.config.noLibraries')}</span> : (
                <div className={styles.grid2}>
                  {libList.map((l) => (
                    <Switch key={l.Id} label={l.Name ?? l.Id} checked={l.Tracked} onCheckedChange={() => toggleLib.mutate(l.Id)} />
                  ))}
                </div>
              )}
            </div>
            <div>
              <div className={styles.name}>{t('app.settings.config.users')}</div>
              {userList.length === 0 ? <span className={styles.meta}>{t('app.settings.config.noUsers')}</span> : (
                <div className={styles.grid2}>
                  {userList.map((u) => (
                    <Switch key={u.Id} label={u.Name ?? u.Id} checked={!excluded.has(u.Id)} onCheckedChange={() => toggleUser.mutate(u.Id)} />
                  ))}
                </div>
              )}
            </div>
          </div>
        )}
      </CardBody>
    </Card>
  )
}

export function ConfigSection() {
  const { t } = useTranslation()
  const cfg = useServerConfig()
  return (
    <div className={styles.stack}>
      <ConnectionCard />
      <PluginCard />
      <DataQualityCard />
      <TrackingCard />
      <Card>
        <CardHeader title={t('app.settings.config.timezone')} subtitle={t('app.settings.config.timezoneHint')} />
        <CardBody>{cfg.isPending ? <Skeleton height={20} width={160} radius="md" /> : <span className={styles.mono}>{cfg.data?.timezone || '—'}</span>}</CardBody>
      </Card>
    </div>
  )
}
