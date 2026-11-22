import { useEffect, useState, type FormEvent } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { ApiError, apiFetch } from '@/lib/api'
import { getDeviceId, getDeviceName, setSession } from '@/lib/auth'
import { useServerConfig } from '@/lib/serverConfig'
import { Button, Card, CardBody, Field, Input, Select, Skeleton } from '@/ui'
import styles from './LoginPage.module.css'

const LANGS = [
  { value: 'en', label: 'English' },
  { value: 'fr', label: 'Français' },
  { value: 'es', label: 'Español' },
]

type Step = 'server' | 'credentials'

function safeNext(next: string | null): string {
  return next && next.startsWith('/') && !next.startsWith('//') ? next : '/'
}

export default function LoginPage() {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const [params] = useSearchParams()
  const config = useServerConfig()
  const [step, setStep] = useState<Step | null>(null)
  const [serverName, setServerName] = useState('')
  const [jellyfinUrl, setJellyfinUrl] = useState('')
  const [apiKey, setApiKey] = useState('')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const configured = config.data?.configured
  const configuredUrl = config.data?.jellyfinUrl
  useEffect(() => {
    if (configured === undefined) return
    if (!configured) { setStep('server'); return }
    let alive = true
    const url = configuredUrl ?? ''
    apiFetch<{ serverName: string }>('/auth/check-server', { body: { url } })
      .then((info) => { if (alive) setServerName(info.serverName) })
      .catch(() => { if (alive) setServerName(url) })
      .finally(() => { if (alive) setStep('credentials') })
    return () => { alive = false }
  }, [configured, configuredUrl])

  async function connect(e: FormEvent) {
    e.preventDefault()
    if (!jellyfinUrl.trim() || !apiKey.trim()) return
    setError(null); setLoading(true)
    try {
      await apiFetch('/auth/configSetup', { body: { JF_HOST: jellyfinUrl.trim(), JF_API_KEY: apiKey.trim() } })
      const info = await apiFetch<{ serverName: string }>('/auth/check-server', { body: { url: jellyfinUrl.trim() } })
      setServerName(info.serverName)
      await qc.invalidateQueries({ queryKey: ['server-config'] })
      setStep('credentials')
    } catch (err) {
      const msg = err instanceof ApiError ? (err.body as { errorMessage?: string } | undefined)?.errorMessage : undefined
      setError(msg ?? t('app.login.configError'))
    } finally { setLoading(false) }
  }

  async function signIn(e: FormEvent) {
    e.preventDefault()
    if (!username.trim() || !password) return
    setError(null); setLoading(true)
    try {
      const res = await apiFetch<{ token: string }>('/auth/jellyfin-login', {
        body: { username, password, deviceId: getDeviceId(), deviceName: getDeviceName() },
      })
      setSession(res.token, username)
      navigate(safeNext(params.get('next')), { replace: true })
    } catch (err) {
      const status = err instanceof ApiError ? err.status : 0
      if (status === 403) setError(t('app.login.adminRequired'))
      else if (status === 503) setError(t('app.login.unreachable'))
      else setError(t('app.login.invalidCredentials'))
      setLoading(false)
    }
  }

  return (
    <main className={styles.page}>
      <div className={styles.wrap}>
        <Card>
          <CardBody>
            <div className={styles.head}>
              <img src="/logo.svg" alt="" width={32} height={32} />
              <div className={styles.lang}>
                <Select
                  aria-label={t('app.login.language')}
                  value={LANGS.some((l) => l.value === i18n.resolvedLanguage) ? (i18n.resolvedLanguage as string) : 'en'}
                  onValueChange={(v) => void i18n.changeLanguage(v)}
                  options={LANGS}
                />
              </div>
            </div>
            {step === null && <Skeleton height={120} />}
            {step === 'server' && (
              <form className={styles.form} onSubmit={connect}>
                <h1 className={styles.title}>{t('app.login.serverTitle')}</h1>
                <p className={styles.sub}>{t('app.login.serverDescription')}</p>
                {error && <p className={styles.error} role="alert">{error}</p>}
                <Field label={t('app.login.jellyfinUrl')} htmlFor="jf-url" hint={t('app.login.jellyfinUrlHint')}>
                  <Input id="jf-url" value={jellyfinUrl} onChange={(e) => setJellyfinUrl(e.target.value)} disabled={loading} autoFocus />
                </Field>
                <Field label={t('app.login.apiKey')} htmlFor="jf-key" hint={t('app.login.apiKeyHint')}>
                  <Input id="jf-key" value={apiKey} onChange={(e) => setApiKey(e.target.value)} disabled={loading} />
                </Field>
                <Button type="submit" variant="primary" className={styles.full} disabled={loading || !jellyfinUrl.trim() || !apiKey.trim()}>{t('app.login.connect')}</Button>
              </form>
            )}
            {step === 'credentials' && (
              <form className={styles.form} onSubmit={signIn}>
                <h1 className={styles.title}>{t('app.login.signIn')}</h1>
                <p className={styles.sub}>{t('app.login.connectedTo', { name: '' })}<strong>{serverName}</strong></p>
                {error && <p className={styles.error} role="alert">{error}</p>}
                <Field label={t('app.login.username')} htmlFor="jf-user">
                  <Input id="jf-user" value={username} onChange={(e) => setUsername(e.target.value)} disabled={loading} autoComplete="username" autoFocus />
                </Field>
                <Field label={t('app.login.password')} htmlFor="jf-pass">
                  <Input id="jf-pass" type="password" value={password} onChange={(e) => setPassword(e.target.value)} disabled={loading} autoComplete="current-password" />
                </Field>
                <Button type="submit" variant="primary" className={styles.full} disabled={loading || !username.trim() || !password}>{t('app.login.signIn')}</Button>
              </form>
            )}
          </CardBody>
        </Card>
        <p className={styles.powered}>{t('app.login.poweredBy')} <a href="https://github.com/Jellystics/jellystics" target="_blank" rel="noopener noreferrer">Jellystics</a></p>
      </div>
    </main>
  )
}
