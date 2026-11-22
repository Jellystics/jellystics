import { useTranslation } from 'react-i18next'
import { Flame } from 'lucide-react'
import type { UserDetail } from '@/api'
import { useFormat } from '@/lib/format'
import { Card, CardBody, CardHeader } from '@/ui'
import { dateOf } from '../history/links'
import { recordDay } from './stats'
import s from './user.module.css'

/** « Série en cours & records »: current / best streak, last 14 days, busiest day. */
export function StreakCard({ user }: { user: UserDetail }) {
  const { t } = useTranslation()
  const f = useFormat()
  const last14 = (user.calendar ?? []).slice(-14)
  const record = recordDay(user.calendar ?? [])
  return (
    <Card>
      <CardHeader title={t('app.userPage.streak.title')} subtitle={t('app.userPage.streak.subtitle')} />
      <CardBody>
        <div className={s.streak}>
          <span className={s.flame} aria-hidden="true"><Flame size={18} strokeWidth={1.6} /></span>
          <div>
            <div className={s.streakValue}>{t('app.userPage.streak.days', { count: user.streak.current })}</div>
            <div className="muted">{t('app.userPage.streak.best', { count: user.streak.best })}</div>
          </div>
          <div className={s.streakDays} role="img" aria-label={t('app.userPage.streak.last14', { count: last14.filter((d) => d.watchSeconds > 0).length })}>
            {last14.map((d) => <span key={d.date} className={d.watchSeconds > 0 ? s.on : undefined} title={`${f.day(dateOf(d.date))} · ${f.duration(d.watchSeconds)}`} />)}
          </div>
        </div>
        <div className={s.records}>
          <div className={s.record}><b>{t('app.userPage.streak.days', { count: user.streak.best })}</b><span>{t('app.userPage.streak.bestLabel')}</span></div>
          <div className={s.record}>
            <b>{record ? f.duration(record.watchSeconds) : '—'}</b>
            <span>{record ? t('app.userPage.streak.recordDay', { date: f.day(dateOf(record.date)) }) : t('app.userPage.streak.noRecord')}</span>
          </div>
        </div>
      </CardBody>
    </Card>
  )
}
