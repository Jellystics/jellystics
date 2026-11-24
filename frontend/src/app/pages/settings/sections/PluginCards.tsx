import { useEffect, useRef, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { Copy, RefreshCw, Upload } from 'lucide-react'
import { Badge, Button, Card, CardBody, CardHeader, ConfirmDialog, EmptyState, ErrorState, Meter, PeriodPicker, SegBar, Skeleton, toast } from '@/ui'
import { useFormat } from '@/lib/format'
import {
  DATA_SOURCES, PLUGIN_MANIFEST_URL, invalidatePluginData, useDataQuality, usePluginStatus, usePushStatus, useResync, useStartPush,
  meLinkState, type DataQuality, type DataSource, type MeLinkState, type PluginStatus, type PushStatus,
} from '@/api/plugin'
import { errorMessage } from '../net'
import styles from './PluginCards.module.css'

/**
 * More precise = stronger contrast with the card surface, in both themes (the
 * sequential scale is inverted in dark mode); sources of unknown precision use
 * neutral inks.
 */
const SOURCE_COLOR: Record<DataSource, string> = {
  plugin: 'var(--seq-6)', tracker: 'var(--seq-4)', playbackReporting: 'var(--seq-3)',
  otherImports: 'var(--seq-2)', restore: 'var(--ink-2)', other: 'var(--ink-3)',
}

const toLocalDate = (s: string) => { const [y, m, d] = s.split('-').map(Number); return new Date(y, m - 1, d) }

function StateBadge({ st }: { st: PluginStatus }) {
  const { t } = useTranslation()
  if (!st.detected) return <Badge>{t('app.settings.plugin.state.absent')}</Badge>
  if (st.healthy) return <Badge tone="good" dot="var(--good)">{t('app.settings.plugin.state.healthy')}</Badge>
  return <Badge tone="warning" dot="var(--warning)">{t('app.settings.plugin.state.unhealthy')}</Badge>
}

function PushBlock() {
  const { t } = useTranslation()
  const f = useFormat()
  const q = usePushStatus()
  const start = useStartPush()
  const qc = useQueryClient()
  const st: PushStatus | undefined = q.data
  const running = st?.state === 'running'
  const noCounts = !!st && st.imported + st.skipped + st.rejected === 0

  // Completion toast when a push this page watched finishes.
  const prev = useRef(st?.state)
  useEffect(() => {
    if (prev.current === 'running' && st?.state === 'done') { toast.success(t('app.settings.plugin.push.doneToast')); invalidatePluginData(qc) }
    if (prev.current === 'running' && st?.state === 'error') toast.error(t('app.settings.plugin.push.failed', { error: st.error ?? '' }))
    prev.current = st?.state
  }, [st?.state, st?.error, t, qc])

  const run = () => start.mutate(undefined, {
    onSuccess: () => toast.success(t('app.settings.plugin.push.started')),
    onError: (e) => toast.error(errorMessage(e, t('app.settings.plugin.push.error'))),
  })

  return (
    <div className={styles.block}>
      <span className={styles.blockTitle}>{t('app.settings.plugin.push.title')}</span>
      <p className={styles.text}>{t('app.settings.plugin.push.hint')}</p>
      {st && running && (
        <div className={styles.detail} role="status">
          <div className={styles.detailHead}>
            <span>{t('app.settings.plugin.push.running')}</span>
            <span>{t('app.settings.plugin.push.progress', { sent: f.number(st.sent), total: f.number(st.total) })}</span>
          </div>
          <Meter value={st.total > 0 ? st.sent / st.total : 0} label={t('app.settings.plugin.push.title')} />
        </div>
      )}
      {st?.state === 'done' && (
        <span className={styles.text} role="status">
          {noCounts
            ? t('app.settings.plugin.push.doneNoCounts', { sent: f.number(st.sent), total: f.number(st.total) })
            : t('app.settings.plugin.push.done', { sent: f.number(st.sent), total: f.number(st.total), imported: f.number(st.imported), skipped: f.number(st.skipped), rejected: f.number(st.rejected) })}
        </span>
      )}
      {st?.state === 'error' && <span className={styles.error} role="alert">{t('app.settings.plugin.push.failed', { error: st.error ?? '' })}</span>}
      {st?.state === 'interrupted' && <span className={styles.error} role="alert">{t('app.settings.plugin.push.interrupted')}</span>}
      <div>
        <Button icon={<Upload size={16} strokeWidth={1.6} />} disabled={running || start.isPending || q.isLoading} onClick={run}>
          {st?.resumable ? t('app.settings.plugin.push.resume') : t('app.settings.plugin.push.start')}
        </Button>
      </div>
    </div>
  )
}

function ResyncBlock() {
  const { t } = useTranslation()
  const resync = useResync()
  const [open, setOpen] = useState(false)
  return (
    <div className={styles.block}>
      <span className={styles.blockTitle}>{t('app.settings.plugin.resync.title')}</span>
      <p className={styles.text}>{t('app.settings.plugin.resync.hint')}</p>
      <div>
        <Button icon={<RefreshCw size={16} strokeWidth={1.6} />} onClick={() => setOpen(true)}>{t('app.settings.plugin.resync.title')}</Button>
      </div>
      <ConfirmDialog
        open={open}
        onOpenChange={setOpen}
        title={t('app.settings.plugin.resync.confirmTitle')}
        description={t('app.settings.plugin.resync.confirmText')}
        confirmLabel={t('app.settings.plugin.resync.confirm')}
        onConfirm={async () => {
          try { await resync.mutateAsync(); toast.success(t('app.settings.plugin.resync.started')) } catch (e) { toast.error(errorMessage(e, t('app.settings.saveError'))); throw e }
        }}
      />
    </div>
  )
}

const ME_TONE: Record<MeLinkState, { tone: 'good' | 'warning' | undefined; dot?: string }> = {
  active: { tone: 'good', dot: 'var(--good)' },
  unreachable: { tone: 'warning', dot: 'var(--warning)' },
  relinking: { tone: 'warning', dot: 'var(--warning)' },
  notLinked: { tone: undefined },
}

/** « Mes statistiques » link status (read only: the address override lives in the plugin settings). */
function MeLinkBlock({ st }: { st: PluginStatus }) {
  const { t } = useTranslation()
  const state = meLinkState(st)
  const { tone, dot } = ME_TONE[state]
  return (
    <div className={styles.block} data-testid="me-link">
      <span className={styles.blockTitle}>{t('app.settings.plugin.me.title')}</span>
      <div><Badge tone={tone} dot={dot}>{t(`app.settings.plugin.me.${state}`)}</Badge></div>
      <p className={styles.text}>{t(`app.settings.plugin.me.${state}Hint`)}</p>
      <dl className={styles.facts}>
        <div>
          <dt>{t('app.settings.plugin.me.url')}</dt>
          <dd>
            {st.meUrl ? <code className={styles.url}>{st.meUrl}</code> : t('app.settings.plugin.me.urlUnknown')}
            {st.meUrl && ` (${t(st.meUrlSource === 'env' ? 'app.settings.plugin.me.sourceEnv' : 'app.settings.plugin.me.sourceAuto')})`}
          </dd>
        </div>
      </dl>
      {st.meLinkError && <span className={styles.error} role="alert">{t('app.settings.plugin.me.error', { error: st.meLinkError })}</span>}
    </div>
  )
}

function InstallHelp() {
  const { t } = useTranslation()
  const copy = async () => {
    try { await navigator.clipboard.writeText(PLUGIN_MANIFEST_URL); toast.success(t('app.settings.plugin.absent.copied')) } catch { toast.error(t('app.settings.saveError')) }
  }
  return (
    <div className={styles.details}>
      <p className={styles.text}>{t('app.settings.plugin.absent.lead')}</p>
      <ul className={styles.benefits}>
        <li>{t('app.settings.plugin.absent.b1')}</li>
        <li>{t('app.settings.plugin.absent.b2')}</li>
        <li>{t('app.settings.plugin.absent.b3')}</li>
      </ul>
      <p className={styles.text}>{t('app.settings.plugin.absent.steps')}</p>
      <div className={styles.manifest}>
        <code className={styles.url} aria-label={t('app.settings.plugin.absent.manifest')}>{PLUGIN_MANIFEST_URL}</code>
        <Button icon={<Copy size={16} strokeWidth={1.6} />} onClick={copy}>{t('app.settings.plugin.absent.copy')}</Button>
      </div>
    </div>
  )
}

export function PluginCard() {
  const { t } = useTranslation()
  const f = useFormat()
  const q = usePluginStatus()
  const st = q.data
  if (q.isError) return <ErrorState title={t('app.error.generic')} onRetry={() => q.refetch()} />
  return (
    <Card>
      <CardHeader title={t('app.settings.plugin.title')} subtitle={t('app.settings.plugin.hint')} action={st && <StateBadge st={st} />} />
      <CardBody>
        {!st ? <Skeleton height={96} radius="md" /> : !st.detected ? (
          <>
            {st.lastError && <span className={styles.error} role="alert">{t('app.settings.plugin.lastErrorValue', { error: st.lastError })}</span>}
            <InstallHelp />
          </>
        ) : (
          <>
            <dl className={styles.facts}>
              <div><dt>{t('app.settings.plugin.version')}</dt><dd>{st.pluginVersion ?? '—'}</dd></div>
              <div>
                <dt>{t('app.settings.plugin.schema')}</dt>
                <dd>{t('app.settings.plugin.schemaValue', { version: st.schemaVersion ?? '—', supported: st.supportedSchemaVersion })}</dd>
              </div>
              <div><dt>{t('app.settings.plugin.lastSync')}</dt><dd>{st.lastSuccessAt ? f.relative(new Date(st.lastSuccessAt)) : t('app.settings.plugin.never')}</dd></div>
              <div><dt>{t('app.settings.plugin.playsSynced')}</dt><dd>{f.number(st.playsSynced)}</dd></div>
            </dl>
            {!st.healthy && <p className={styles.text}>{t('app.settings.plugin.unhealthyHint')}</p>}
            {st.lastError && <span className={styles.error} role="alert">{t('app.settings.plugin.lastErrorValue', { error: st.lastError })}</span>}
            <MeLinkBlock st={st} />
            <PushBlock />
            <ResyncBlock />
          </>
        )}
      </CardBody>
    </Card>
  )
}

function DetailMeter({ label, n, total }: { label: string; n: number; total: number }) {
  const f = useFormat()
  return (
    <div className={styles.detail}>
      <div className={styles.detailHead}>
        <span>{label}</span>
        <span>{f.number(n)} / {f.number(total)}{total > 0 ? ` · ${f.percent(n / total)}` : ''}</span>
      </div>
      <Meter thin value={total > 0 ? n / total : 0} label={label} />
    </div>
  )
}

/** Plugin start date, repaired durations (period) and undated plays (all time). */
function Notes({ d }: { d: DataQuality }) {
  const { t } = useTranslation()
  const f = useFormat()
  return (
    <ul className={styles.notes}>
      <li>{d.pluginCollectedSince
        ? t('app.settings.quality.pluginSince', { date: f.day(toLocalDate(d.pluginCollectedSince)) })
        : t('app.settings.quality.noPlugin')}</li>
      {(d.totals.repaired ?? 0) > 0 && <li>{t('app.settings.quality.repairedAtLeast', { count: d.totals.repaired })}</li>}
      {d.undated > 0 && <li>{t('app.settings.quality.undated', { count: d.undated })}</li>}
    </ul>
  )
}

export function DataQualityCard() {
  const { t } = useTranslation()
  const f = useFormat()
  const q = useDataQuality()
  const d = q.data
  // Most precise first, whatever order the server sends.
  const visible = d ? DATA_SOURCES.flatMap((k) => d.sources.filter((s) => s.key === k && s.clicks > 0)) : []
  return (
    <Card>
      <CardHeader title={t('app.settings.quality.title')} subtitle={t('app.settings.quality.hint')} action={<PeriodPicker />} />
      <CardBody>
        {q.isError ? <ErrorState title={t('app.error.generic')} onRetry={() => q.refetch()} /> : !d ? <Skeleton height={160} radius="md" /> : d.totals.clicks === 0 ? (
          <div className={styles.details}>
            <EmptyState title={t('app.settings.quality.empty')} description={t('app.analytics.noDataHint')} />
            <Notes d={d} />
          </div>
        ) : (
          <div className={styles.details}>
            <SegBar thin parts={visible.map((s) => ({
              label: t(`app.settings.quality.sources.${s.key}.label`), value: s.watchSeconds, color: SOURCE_COLOR[s.key], display: f.percent(s.watchShare),
            }))} />
            <ul className={styles.sources}>
              {visible.map((s) => (
                <li key={s.key} className={styles.source}>
                  <span className={styles.sourceHead}>
                    <span className={styles.dot} style={{ background: SOURCE_COLOR[s.key] }} aria-hidden="true" />
                    {t(`app.settings.quality.sources.${s.key}.label`)}
                  </span>
                  <span className={styles.figures}>
                    <strong>{t('app.settings.quality.views', { count: s.views })}</strong>
                    <span>{t('app.settings.quality.watchShare', { share: f.percent(s.watchShare), time: f.duration(s.watchSeconds) })}</span>
                    <span>{t('app.settings.quality.clicks', { count: s.clicks })}</span>
                  </span>
                  <span className={styles.precision}>{t(`app.settings.quality.sources.${s.key}.precision`)}</span>
                </li>
              ))}
            </ul>
            <div className={styles.block}>
              <span className={styles.blockTitle}>{t('app.settings.quality.details.title')}</span>
              <DetailMeter label={t('app.settings.quality.details.positions')} n={d.totals.positions} total={d.totals.clicks} />
              <DetailMeter label={t('app.settings.quality.details.streams')} n={d.totals.streams} total={d.totals.clicks} />
              <DetailMeter label={t('app.settings.quality.details.captured')} n={d.totals.captured} total={d.totals.clicks} />
              {d.totals.transcodes > 0 && <DetailMeter label={t('app.settings.quality.details.transcode')} n={d.totals.transcodeDetailed} total={d.totals.transcodes} />}
            </div>
            <Notes d={d} />
          </div>
        )}
      </CardBody>
    </Card>
  )
}
