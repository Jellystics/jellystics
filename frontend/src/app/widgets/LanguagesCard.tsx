import { useTranslation } from 'react-i18next'
import { useUserLanguages } from '@/api/analytics'
import { BarList, Card, CardBody, CardHeader, SegBar } from '@/ui'
import { useFormat } from '@/lib/format'
import { usePeriod } from '@/lib/useUrlState'
import { CaptureEmpty, QueryCard } from '@/app/pages/_shared/shared'
import styles from '@/app/pages/_shared/analytics.module.css'

/** ISO 639-2 (as stored by Jellyfin) → BCP 47, for Intl.DisplayNames. */
const ISO2: Record<string, string> = {
  eng: 'en', fre: 'fr', fra: 'fr', spa: 'es', ger: 'de', deu: 'de', ita: 'it', jpn: 'ja', por: 'pt', rus: 'ru',
  chi: 'zh', zho: 'zh', kor: 'ko', dut: 'nl', nld: 'nl', swe: 'sv', nor: 'no', dan: 'da', fin: 'fi', pol: 'pl',
  ara: 'ar', hin: 'hi', tur: 'tr', heb: 'he', gre: 'el', ell: 'el', cze: 'cs', ces: 'cs', hun: 'hu',
}

function languageName(code: string, locale: string): string {
  const tag = ISO2[code] ?? code
  try {
    return new Intl.DisplayNames(locale, { type: 'language' }).of(tag) ?? code
  } catch {
    return code
  }
}

/**
 * VO / VF and subtitles of one user over the URL period (/users/:id/languages).
 * "Original" = the first audio track of the file was played.
 */
export function LanguagesCard({ userId }: { userId: string }) {
  const { t } = useTranslation()
  const f = useFormat()
  const q = useUserLanguages(userId)
  const [period] = usePeriod()
  const name = (code: string) => (code ? languageName(code, f.locale) : t('app.languages.unknown'))
  return (
    <QueryCard
      title={t('app.languages.title')} subtitle={t('app.languages.subtitle')} query={q}
      isEmpty={(d) => d.captured === 0} empty={q.data ? <CaptureEmpty since={q.data.collectedSince} from={period.from} /> : undefined}
    >
      {(d) => {
        const known = d.split.originalSubtitled + d.split.original + d.split.dubbed
        return (
          <Card>
            <CardHeader
              title={t('app.languages.title')} subtitle={t('app.languages.subtitle')}
              action={known > 0 ? <span className={styles.muted}>{t('app.languages.original', { value: f.percent((d.split.originalSubtitled + d.split.original) / known) })}</span> : undefined}
            />
            <CardBody>
              {known > 0 && (
                <SegBar parts={[
                  { label: t('app.languages.originalSubtitled'), value: d.split.originalSubtitled, color: 'var(--m-direct)', ink: 'var(--m-direct-ink)' },
                  { label: t('app.languages.originalOnly'), value: d.split.original, color: 'var(--m-stream)', ink: 'var(--m-stream-ink)' },
                  { label: t('app.languages.dubbed'), value: d.split.dubbed, color: 'var(--m-transcode)', ink: 'var(--m-transcode-ink)' },
                ]} />
              )}
              <div className={styles.gap} />
              <BarList labelHeader={t('app.languages.audio')} valueLabel={t('app.chart.views')}
                rows={d.audio.slice(0, 5).map((a) => ({ key: a.label || '_', label: name(a.label), value: a.views }))} />
              <p className={styles.note}>
                {t('app.languages.subtitles', { value: f.percent(d.captured ? (d.captured - d.subtitleOff) / d.captured : 0) })}
                {d.subtitles.length > 0 && ` · ${d.subtitles.slice(0, 3).map((s) => name(s.label)).join(', ')}`}
              </p>
            </CardBody>
          </Card>
        )
      }}
    </QueryCard>
  )
}
