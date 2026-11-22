import { useTranslation } from 'react-i18next'
import { Cpu } from 'lucide-react'
import { usePlaybackClients, usePlaybackMethods, usePlaybackNetwork, usePlaybackQuality } from '@/api/analytics'
import type { MethodSplit } from '@/api'
import { ChartFrame, Columns, Matrix, StackedColumns } from '@/charts'
import { Badge, BarList, Card, CardBody, CardHeader, ErrorState, PageHeader, SegBar, Skeleton, StatStrip, StatTile, type StatDelta } from '@/ui'
import { useFormat } from '@/lib/format'
import { usePeriod } from '@/lib/useUrlState'
import { CollectedEmpty, QueryCard } from '../insights/shared'
import { formatBitrate } from '../insights/analyticsFormat'
import styles from '../insights/analytics.module.css'

/** Jellyfin TranscodeReason values with a translated label (others show raw). */
const KNOWN_REASONS = new Set([
  'ContainerNotSupported', 'VideoCodecNotSupported', 'AudioCodecNotSupported', 'SubtitleCodecNotSupported',
  'AudioIsExternal', 'SecondaryAudioNotSupported', 'VideoProfileNotSupported', 'VideoLevelNotSupported',
  'VideoResolutionNotSupported', 'VideoBitDepthNotSupported', 'AudioChannelsNotSupported',
  'ContainerBitrateExceedsLimit', 'VideoBitrateNotSupported', 'AudioBitrateNotSupported',
  'VideoRangeTypeNotSupported', 'DirectPlayError',
])
const RESOLUTIONS = [
  { key: '4K', color: 'var(--seq-6)' },
  { key: '1080p', color: 'var(--seq-4)' },
  { key: '720p', color: 'var(--seq-3)', ink: 'var(--ink)' },
  { key: 'SD', color: 'var(--seq-2)', ink: 'var(--ink)' },
] as const
const BITRATE_LABELS = ['< 2', '2–5', '5–10', '10–20', '20–40', '≥ 40']

const total = (m: MethodSplit) => m.direct + m.stream + m.transcode
const share = (n: number, d: number) => (d > 0 ? n / d : 0)

export default function PlaybackPage() {
  const { t } = useTranslation()
  const f = useFormat()
  const [period] = usePeriod()
  const methods = usePlaybackMethods()
  const clients = usePlaybackClients()
  const quality = usePlaybackQuality()
  const network = usePlaybackNetwork()

  const reasonLabel = (r: string) => (KNOWN_REASONS.has(r) ? t(`app.playback.reasons.${r}`) : r)
  const codec = (c: string) => (c === 'copy' ? t('app.playback.copy') : c || '—')
  /** Note shown when the capture started inside the selected period. */
  const partial = (since: string | null) => (since && since > period.from
    ? <p className={styles.note}>{t('app.analytics.collectedSince', { date: f.day(new Date(`${since}T12:00:00`)) })}</p>
    : null)

  const tiles = [methods, clients, quality]
  const tilesFailed = tiles.some((q) => q.isError && q.data === undefined)
  const tilesLoading = !tilesFailed && tiles.some((q) => q.data === undefined)

  const pts = (cur: number, prev: number | undefined, upIsGood: boolean): StatDelta | undefined => {
    if (prev === undefined) return undefined
    const diff = Math.round((cur - prev) * 100)
    const direction = diff > 0 ? 'up' : diff < 0 ? 'down' : 'flat'
    return { text: t('app.playback.points', { value: `${diff > 0 ? '+' : diff < 0 ? '−' : ''}${Math.abs(diff)}` }), direction, good: diff === 0 || (diff > 0) === upIsGood }
  }

  const renderTiles = () => {
    const m = methods.data!, q = quality.data!, cl = clients.data!
    const n = total(m.totals)
    const prevN = m.previous ? total(m.previous) : 0
    const directShare = share(m.totals.direct, n), transcodeShare = share(m.totals.transcode, n)
    const topReason = q.reasons[0]
    const worst = [...cl].filter((c) => c.views > 0 && (c.transcodeRatio ?? 0) > 0).sort((a, b) => (b.transcodeRatio ?? 0) - (a.transcodeRatio ?? 0))[0]
    return (
      <StatStrip columns={4}>
        <StatTile label={t('app.playback.tiles.direct')} value={n ? f.percent(directShare) : '—'}
          delta={m.previous && prevN ? pts(directShare, share(m.previous.direct, prevN), true) : undefined} />
        <StatTile label={t('app.playback.tiles.transcode')} value={n ? f.percent(transcodeShare) : '—'}
          delta={m.previous && prevN ? pts(transcodeShare, share(m.previous.transcode, prevN), false) : undefined} />
        <StatTile label={t('app.playback.tiles.reason')} value={topReason ? reasonLabel(topReason.label) : '—'}
          foot={topReason && q.transcodes ? t('app.playback.tiles.reasonFoot', { value: f.percent(share(topReason.views, q.transcodes)) }) : undefined} />
        <StatTile label={t('app.playback.tiles.client')} value={worst ? worst.client : '—'}
          foot={worst ? t('app.playback.tiles.clientFoot', { value: f.percent(worst.transcodeRatio ?? 0) }) : undefined} />
      </StatStrip>
    )
  }

  const historyClient = (c: string) => `/history?client=${encodeURIComponent(c)}`

  return (
    <div className={styles.page}>
      <PageHeader title={t('app.playback.title')} subtitle={t('app.playback.subtitle')} />
      <Card>
        {tilesFailed ? (
          <CardBody><ErrorState title={t('app.error.generic')} onRetry={() => tiles.forEach((q) => { if (q.isError) q.refetch() })} /></CardBody>
        ) : tilesLoading ? (
          <CardBody><div aria-busy="true" className={styles.tilesSkeleton}><Skeleton height={64} radius="md" /></div></CardBody>
        ) : renderTiles()}
      </Card>

      <div className={styles.grid}>
        <QueryCard title={t('app.playback.methods.title')} query={methods} wide isEmpty={(d) => total(d.totals) === 0}>
          {(d) => {
            const series = [
              { name: t('app.method.direct'), color: 'var(--m-direct)', values: d.daily.map((x) => x.direct) },
              { name: t('app.method.stream'), color: 'var(--m-stream)', values: d.daily.map((x) => x.stream) },
              { name: t('app.method.transcode'), color: 'var(--m-transcode)', values: d.daily.map((x) => x.transcode) },
            ]
            return (
              <ChartFrame
                title={t('app.playback.methods.title')} subtitle={t('app.chart.views')} csvName="play-methods"
                legend={series.map((s) => ({ label: s.name, color: s.color }))}
                table={{ columns: [t('app.playback.date'), ...series.map((s) => s.name)], rows: d.daily.map((x) => [x.date, x.direct, x.stream, x.transcode]) }}
              >
                <StackedColumns labels={d.daily.map((x) => f.day(new Date(`${x.date}T12:00:00`)))} series={series} xEvery={Math.max(1, Math.ceil(d.daily.length / 6))} />
              </ChartFrame>
            )
          }}
        </QueryCard>

        <QueryCard title={t('app.playback.clients.title')} query={clients} isEmpty={(d) => !d.some((c) => c.views > 0)}>
          {(d) => (
            <Card>
              <CardHeader title={t('app.playback.clients.title')} subtitle={t('app.playback.clients.subtitle')} />
              <CardBody>
                <BarList labelHeader={t('app.playback.client')} valueLabel={t('app.playback.clients.value')} max={1} format={f.percent}
                  rows={d.filter((c) => c.views > 0).map((c) => ({ key: c.client, label: c.client, value: c.transcodeRatio ?? 0, to: historyClient(c.client) }))} />
              </CardBody>
            </Card>
          )}
        </QueryCard>

        <QueryCard title={t('app.playback.falseStarts.title')} query={clients} isEmpty={(d) => d.length === 0}>
          {(d) => (
            <Card>
              <CardHeader title={t('app.playback.falseStarts.title')} subtitle={t('app.playback.falseStarts.subtitle')} />
              <CardBody>
                <BarList labelHeader={t('app.playback.client')} valueLabel={t('app.playback.falseStarts.value')} format={f.percent}
                  max={Math.max(0.01, ...d.map((c) => c.falseStartRatio ?? 0))}
                  rows={d.map((c) => ({
                    key: c.client, label: c.client, value: c.falseStartRatio ?? 0, to: historyClient(c.client),
                    badge: (c.transcodeRatio ?? 0) >= 0.5
                      ? <Badge tone="serious" icon={<Cpu size={12} strokeWidth={1.6} />}>{t('app.playback.falseStarts.transcoded', { value: f.percent(c.transcodeRatio ?? 0) })}</Badge>
                      : undefined,
                  }))} />
              </CardBody>
            </Card>
          )}
        </QueryCard>

        <QueryCard title={t('app.playback.resolution.title')} query={quality} isEmpty={(d) => RESOLUTIONS.every((r) => !d.resolution.served[r.key])}>
          {(d) => (
            <Card>
              <CardHeader title={t('app.playback.resolution.title')} subtitle={t('app.playback.resolution.subtitle')} />
              <CardBody>
                <SegBar parts={RESOLUTIONS.map((r) => ({ label: r.key, value: d.resolution.served[r.key] ?? 0, color: r.color, ink: 'ink' in r ? r.ink : undefined }))} />
                {d.resolution.sourceUhd > 0 && (
                  <p className={styles.note}>{t('app.playback.resolution.downscaled', { value: f.percent(share(d.resolution.downscaledUhd, d.resolution.sourceUhd)) })}</p>
                )}
              </CardBody>
            </Card>
          )}
        </QueryCard>

        <QueryCard title={t('app.playback.hdr.title')} query={quality} isEmpty={(d) => d.hdr.native + d.hdr.toneMapped + d.hdr.sdr === 0}>
          {(d) => (
            <Card>
              <CardHeader title={t('app.playback.hdr.title')} subtitle={t('app.playback.hdr.subtitle')} />
              <CardBody>
                <SegBar parts={[
                  { label: t('app.playback.hdr.native'), value: d.hdr.native, color: 'var(--m-direct)', ink: 'var(--m-direct-ink)' },
                  { label: t('app.playback.hdr.toneMapped'), value: d.hdr.toneMapped, color: 'var(--m-stream)', ink: 'var(--m-stream-ink)' },
                  { label: t('app.playback.hdr.sdr'), value: d.hdr.sdr, color: 'var(--m-transcode)', ink: 'var(--m-transcode-ink)' },
                ]} />
              </CardBody>
            </Card>
          )}
        </QueryCard>

        <QueryCard title={t('app.playback.reasons.title')} query={quality} isEmpty={(d) => d.reasons.length === 0}
          empty={quality.data && quality.data.captured === 0 ? <CollectedEmpty since={quality.data.collectedSince} /> : undefined}>
          {(d) => (
            <Card>
              <CardHeader title={t('app.playback.reasons.title')} subtitle={t('app.playback.reasons.subtitle')} />
              <CardBody>
                <BarList labelHeader={t('app.playback.reasons.reason')} valueLabel={t('app.playback.reasons.value')} format={f.percent}
                  max={1} rows={d.reasons.slice(0, 8).map((r) => ({ key: r.label, label: reasonLabel(r.label), value: share(r.views, d.transcodes) }))} />
                {partial(d.collectedSince)}
              </CardBody>
            </Card>
          )}
        </QueryCard>

        <QueryCard title={t('app.playback.streams.title')} query={quality} isEmpty={(d) => d.streams.audioOnly + d.streams.videoOnly + d.streams.both === 0}>
          {(d) => (
            <Card>
              <CardHeader title={t('app.playback.streams.title')} subtitle={t('app.playback.streams.subtitle')} />
              <CardBody>
                <SegBar parts={[
                  { label: t('app.playback.streams.audioOnly'), value: d.streams.audioOnly, color: 'var(--m-transcode)', ink: 'var(--m-transcode-ink)' },
                  { label: t('app.playback.streams.videoOnly'), value: d.streams.videoOnly, color: 'var(--m-stream)', ink: 'var(--m-stream-ink)' },
                  { label: t('app.playback.streams.both'), value: d.streams.both, color: 'var(--m-direct)', ink: 'var(--m-direct-ink)' },
                ]} />
                {d.hwAccel.length > 0 && (
                  <p className={styles.note}>{t('app.playback.streams.hwAccel', { value: d.hwAccel.map((h) => `${h.label} (${f.number(h.views)})`).join(', ') })}</p>
                )}
              </CardBody>
            </Card>
          )}
        </QueryCard>

        <QueryCard title={t('app.playback.conversions.title')} query={quality} isEmpty={(d) => d.conversions.length === 0}>
          {(d) => (
            <Card>
              <CardHeader title={t('app.playback.conversions.title')} subtitle={t('app.playback.conversions.subtitle')} />
              <CardBody flush>
                <table className={styles.table}>
                  <thead><tr><th scope="col">{t('app.playback.conversions.video')}</th><th scope="col">{t('app.playback.conversions.audio')}</th><th scope="col" className={styles.num}>{t('app.chart.views')}</th></tr></thead>
                  <tbody>
                    {d.conversions.map((c, i) => (
                      <tr key={i}>
                        <td className={styles.mono}>{`${codec(c.videoFrom)} → ${codec(c.videoTo)}`}</td>
                        <td className={styles.mono}>{`${codec(c.audioFrom)} → ${codec(c.audioTo)}`}</td>
                        <td className={styles.num}>{f.number(c.views)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </CardBody>
            </Card>
          )}
        </QueryCard>

        <QueryCard title={t('app.playback.matrix.title')} query={quality} isEmpty={(d) => d.conversions.length === 0}>
          {(d) => {
            const rows = [...new Set(d.conversions.map((c) => codec(c.videoFrom)))]
            const cols = [...new Set(d.conversions.map((c) => codec(c.videoTo)))]
            const values = rows.map((r) => cols.map((col) => {
              const n = d.conversions.filter((c) => codec(c.videoFrom) === r && codec(c.videoTo) === col).reduce((a, c) => a + c.views, 0)
              return n || null
            }))
            return (
              <ChartFrame
                title={t('app.playback.matrix.title')} subtitle={t('app.playback.matrix.subtitle')} csvName="codec-matrix"
                table={{ columns: [t('app.playback.matrix.source'), ...cols], rows: rows.map((r, i) => [r, ...values[i].map((v) => v ?? 0)]) }}
              >
                <Matrix rows={rows} cols={cols} values={values} unit={t('app.chart.views')} labelWidth={96} />
              </ChartFrame>
            )
          }}
        </QueryCard>

        <QueryCard title={t('app.playback.network.title')} query={network} isEmpty={(d) => d.captured === 0}
          empty={network.data ? <CollectedEmpty since={network.data.collectedSince} /> : undefined}>
          {(d) => (
            <Card>
              <CardHeader title={t('app.playback.network.title')} subtitle={t('app.playback.network.subtitle')} />
              <CardBody>
                <SegBar parts={[
                  { label: t('app.playback.network.local'), value: d.local, color: 'var(--m-direct)', ink: 'var(--m-direct-ink)' },
                  { label: t('app.playback.network.remote'), value: d.remote, color: 'var(--m-transcode)', ink: 'var(--m-transcode-ink)' },
                ]} />
                {d.places.length > 0 && (
                  <>
                    <div className={styles.gap} />
                    <BarList labelHeader={t('app.playback.network.place')} valueLabel={t('app.chart.views')}
                      rows={d.places.map((p) => ({ key: `${p.country}/${p.city ?? ''}`, label: p.city ? `${p.city}, ${p.country}` : p.country, value: p.views }))} />
                  </>
                )}
                {partial(d.collectedSince)}
              </CardBody>
            </Card>
          )}
        </QueryCard>

        <QueryCard title={t('app.playback.bitrate.title')} query={quality} isEmpty={(d) => d.bitrate.samples === 0}
          empty={quality.data ? <CollectedEmpty since={quality.data.collectedSince} /> : undefined}>
          {(d) => (
            <ChartFrame
              title={t('app.playback.bitrate.title')}
              subtitle={d.bitrate.avg !== null && d.bitrate.p95 !== null
                ? t('app.playback.bitrate.subtitle', { avg: formatBitrate(d.bitrate.avg, f.locale), p95: formatBitrate(d.bitrate.p95, f.locale) })
                : undefined}
              csvName="bitrate"
              table={{ columns: [t('app.playback.bitrate.bucket'), t('app.chart.views')], rows: BITRATE_LABELS.map((l, i) => [`${l} Mb/s`, d.bitrate.buckets[i] ?? 0]) }}
            >
              <Columns labels={BITRATE_LABELS.map((l) => `${l} Mb/s`)} values={d.bitrate.buckets} name={t('app.chart.views')}
                emphasis={d.bitrate.buckets.indexOf(Math.max(...d.bitrate.buckets))} />
              {partial(d.collectedSince)}
            </ChartFrame>
          )}
        </QueryCard>
      </div>
    </div>
  )
}
