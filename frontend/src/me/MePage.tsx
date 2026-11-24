import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ArrowLeft, LogIn, PlugZap, RotateCw } from 'lucide-react'
import { Avatar } from '@/ui/Avatar'
import { Button, buttonClass } from '@/ui/Button'
import { Card } from '@/ui/Card'
import { EmptyState } from '@/ui/EmptyState'
import { SegmentedControl } from '@/ui/SegmentedControl'
import { Skeleton } from '@/ui/Skeleton'
import { useFormat } from '@/lib/format'
import { dateOf } from '@/app/pages/history/links'
import { MeApiError, useMeProfile, type UnavailableReason } from './api'
import { isPreset, meRange, ME_PRESETS, todayIn, type MePreset } from './period'
import { jellyfinHome, userImageUrl, type Session } from './session'
import { DailyCard, DevicesCards, HabitsCard, HistoryCard, KpiStrip, PlaysCard, TopCard, TypesCard } from './sections'
import l from '@/app/pages/overview/layout.module.css'
import s from './me.module.css'

function BackLink({ basePath }: { basePath: string }) {
  const { t } = useTranslation()
  return <a className={s.back} href={jellyfinHome(basePath)}><ArrowLeft size={14} aria-hidden="true" />{t('app.me.back')}</a>
}

function TopBar({ basePath, user }: { basePath: string; user?: { id: string; name: string; tag: string | null } }) {
  const { t } = useTranslation()
  return (
    <header className={s.top}>
      <BackLink basePath={basePath} />
      <h1 className={s.title}>{t('app.me.title')}</h1>
      {user && (
        <span className={s.who}>
          <Avatar id={user.id} name={user.name} src={user.tag ? userImageUrl(basePath, user.id, 64, user.tag) : null} />
          <span>{user.name}</span>
        </span>
      )}
    </header>
  )
}

/** Whole-page state when the user is not signed in to Jellyfin in this browser. */
export function SignedOut({ basePath }: { basePath: string }) {
  const { t } = useTranslation()
  return (
    <>
      <TopBar basePath={basePath} />
      <div className={s.center}>
        <Card>
          <EmptyState
            role="alert"
            icon={<LogIn size={20} strokeWidth={1.6} />}
            title={t('app.me.signedOut.title')}
            description={t('app.me.signedOut.description')}
            action={<a className={buttonClass({ variant: 'primary' })} href={jellyfinHome(basePath)}>{t('app.me.signedOut.action')}</a>}
          />
        </Card>
      </div>
    </>
  )
}

/** Whole-page state when the statistics cannot be served (plugin not linked, Jellystics down, unknown user). */
export function Unavailable({ basePath, reason, onRetry }: { basePath: string; reason: UnavailableReason; onRetry: () => void }) {
  const { t } = useTranslation()
  if (reason === 'signedOut') return <SignedOut basePath={basePath} />
  return (
    <>
      <TopBar basePath={basePath} />
      <div className={s.center}>
        <Card>
          <EmptyState
            role="alert"
            icon={<PlugZap size={20} strokeWidth={1.6} />}
            title={t('app.me.unavailable.title')}
            description={t(`app.me.unavailable.${reason}`)}
            action={
              <div className={s.actions}>
                <Button size="sm" icon={<RotateCw size={16} strokeWidth={1.6} />} onClick={onRetry}>{t('app.action.retry')}</Button>
                <a className={buttonClass({ size: 'sm', variant: 'ghost' })} href={jellyfinHome(basePath)}>{t('app.me.back')}</a>
              </div>
            }
          />
        </Card>
      </div>
    </>
  )
}

const readPreset = (): MePreset => {
  try { const v = new URLSearchParams(window.location.search).get('range'); return isPreset(v) ? v : '30d' } catch { return '30d' }
}
const writePreset = (p: MePreset) => {
  try {
    const u = new URL(window.location.href)
    if (p === '30d') u.searchParams.delete('range'); else u.searchParams.set('range', p)
    window.history.replaceState(window.history.state, '', u)
  } catch { /* ignore */ }
}

/** « Mes statistiques »: the signed-in Jellyfin user's own statistics. */
export function MePage({ session, basePath }: { session: Session; basePath: string }) {
  const { t } = useTranslation()
  const f = useFormat()
  const [preset, setPreset] = useState<MePreset>(readPreset)
  const [now] = useState(() => new Date())
  const profile = useMeProfile(session)
  const p = profile.data
  // Sections render once the profile is known, so the range is always in the server display zone.
  const range = meRange(preset, todayIn(p?.timezone, now))

  const err = profile.error
  if (!p && err instanceof MeApiError && err.reason !== 'error') return <Unavailable basePath={basePath} reason={err.reason} onRetry={() => profile.refetch()} />
  if (!p && profile.isError) return <Unavailable basePath={basePath} reason="error" onRetry={() => profile.refetch()} />

  const choose = (v: MePreset) => { setPreset(v); writePreset(v) }

  return (
    <>
      <TopBar basePath={basePath} user={p ? { id: p.id, name: p.name, tag: p.primaryImageTag } : undefined} />
      <main className={s.main}>
        <div className={s.bar}>
          <div>
            <h2>{p ? p.name : <Skeleton width={160} height={22} />}</h2>
            {p?.memberSince && <div className={s.meta}>{t('app.me.memberSince', { date: f.day(dateOf(p.memberSince)) })}</div>}
          </div>
          <div className={s.periods}>
            <SegmentedControl<MePreset>
              aria-label={t('app.me.period.label')}
              value={preset}
              onValueChange={choose}
              options={ME_PRESETS.map((v) => ({ value: v, label: t(`app.me.period.${v}`) }))}
            />
          </div>
        </div>
        {!p ? (
          <div className={l.page} aria-busy="true">
            <Skeleton height={96} radius="lg" />
            <Skeleton height={260} radius="lg" />
            <div className={l.g2}><Skeleton height={240} radius="lg" /><Skeleton height={240} radius="lg" /></div>
          </div>
        ) : (
          <>
            <KpiStrip session={session} range={range} profile={p} />
            <DailyCard session={session} range={range} />
            <div className={l.wide}>
              <TopCard session={session} range={range} basePath={basePath} />
              <TypesCard session={session} range={range} />
            </div>
            <div className={l.g2}>
              <HabitsCard session={session} range={range} />
              <PlaysCard session={session} range={range} />
            </div>
            <DevicesCards session={session} range={range} />
            <HistoryCard key={range.preset} session={session} range={range} basePath={basePath} />
          </>
        )}
      </main>
    </>
  )
}
