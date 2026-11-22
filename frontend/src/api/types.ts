// Response types of the /api/v2 endpoints (backend/internal/handler/v2_*.go).
// Raw values only: seconds, counts, ticks, ratios in 0..1, ISO timestamps with
// offset, dates YYYY-MM-DD. Formatting belongs to the UI.

export type MediaFamily = 'movie' | 'series' | 'music'
export type GroupKind = 'movie' | 'series' | 'album'
export type ItemKind = 'movie' | 'episode' | 'track'
export type MediaType = 'Movie' | 'Episode' | 'Audio'
export type ISODate = string
export type ISOTime = string

export interface ByType<T> { movie: T; series: T; music: T }
export interface RefUser { id: string; name: string }
export interface TypeMetrics { watchSeconds: number; views: number; clicks: number }
export interface MethodSplit { direct: number; stream: number; transcode: number }

export interface Totals {
  watchSeconds: number
  views: number
  clicks: number
  activeUsers: number
  /** Average completion ratio of (user, item, day) view units; null without data. */
  completion: number | null
}

export interface Paged<T> { rows: T[]; total: number; page: number; pageSize: number }

// ── /overview ────────────────────────────────────────────────────────────────
export interface DailyPoint { date: ISODate; watchSeconds: number; views: number; clicks: number; byType: ByType<TypeMetrics> }
export interface TopItem { id: string; kind: GroupKind; name: string; views: number; watchSeconds: number; artId: string }
export interface TopUser { id: string; name: string; watchSeconds: number; views: number }
export interface PeakPoint { date: ISODate; peak: number }
export interface OnThisDayRow { itemId: string; kind: GroupKind; name: string; users: string[]; date: ISODate }
export interface Overview {
  /** Granularity of daily / previousDaily / concurrencyPeak: ISO weeks (labelled by Monday) when the range exceeds 400 days. */
  bucket: 'day' | 'week'
  totals: Totals
  previous?: Totals
  daily: DailyPoint[]
  previousDaily?: DailyPoint[]
  /** Up to 10 per kind, sorted by views. */
  topItems: TopItem[]
  topUsers: TopUser[]
  byMethod: MethodSplit
  /** 7 × 24 views, Monday first, local hours. */
  weekHour: number[][]
  concurrencyPeak: PeakPoint[]
  onThisDay: OnThisDayRow[]
}

// ── /history ─────────────────────────────────────────────────────────────────
export interface HistoryItem {
  id: string
  kind: ItemKind
  name: string
  parentId?: string
  parentName?: string
  season?: number
  episode?: number
}
export interface HistoryRow {
  id: string
  startedAt: ISOTime
  endedAt: ISOTime
  user: RefUser
  item: HistoryItem
  mediaType: MediaType
  watchSeconds: number
  completion: number | null
  client: string
  device: string
  method: string
  isView: boolean
  pausedSeconds: number
}
export type HistoryPage = Paged<HistoryRow>
export interface CalendarItem { id: string; name: string; kind: GroupKind; user: string }
export interface CalendarDay { date: ISODate; views: number; watchSeconds: number; items: CalendarItem[] }

// ── /search ──────────────────────────────────────────────────────────────────
export type SearchKind = GroupKind | 'artist' | 'episode' | 'track'
export interface SearchItem { id: string; kind: SearchKind; name: string; year?: number; parentId?: string; parentName?: string }
export interface SearchResult { items: SearchItem[]; users: RefUser[] }

// ── /users ───────────────────────────────────────────────────────────────────
export interface UserRow {
  id: string
  name: string
  watchSeconds: number
  views: number
  /** Watch seconds per family. */
  byType: ByType<number>
  lastActivity: ISOTime | null
  /** Watch seconds per day, last 14 days of the range (oldest first). */
  trend: number[]
}
export interface NamedSeconds { name: string; watchSeconds: number }
export interface InProgress { seriesId: string; name: string; nextSeason: number | null; nextEpisode: number | null; nextId: string; lastPlayed: ISOTime; progress: number }
export interface UserDetail {
  id: string
  name: string
  memberSince: ISOTime | null
  favoriteDevice: string | null
  totals: Totals
  previous?: Totals
  calendar: { date: ISODate; watchSeconds: number }[]
  inProgress: InProgress[]
  hours: number[]
  profile: { regularity: number | null; discovery: number | null; concentration: number | null; completion: number | null }
  genres: ByType<NamedSeconds[]>
  devices: { device: string; client: string; version: string; views: number }[]
  streak: { current: number; best: number }
}

// ── Entities ─────────────────────────────────────────────────────────────────
export interface Funnel { clicks: number; views: number; half: number; completed: number }
/** Library an entity belongs to (null when unknown). */
export interface LibraryRef { id: string; name: string }
/** Neighbouring episode: regular episodes cross seasons, specials only link to specials. */
export interface EpisodeRef { id: string; season: number | null; episode: number | null }

export interface MovieDetail {
  item: { id: string; name: string; year: number | null; runtimeSeconds: number; genres: string[]; sizeBytes: number | null; resolution: string | null; library: LibraryRef | null }
  totals: { views: number; clicks: number; viewers: number; completedRatio: number | null; transcodeRatio: number | null; watchSeconds: number }
  funnel: Funnel
  /** 24 bins over the runtime: furthest stop of each (user, item, day) view unit. */
  stopPositions: number[]
  viewers: { user: RefUser; views: number; lastWatched: ISOTime; bestCompletion: number | null }[]
  history: HistoryRow[]
}

export type CadenceBucket = '<1min' | '1-15min' | '15-60min' | 'sameDay' | 'nextDay' | '2-7d' | '>7d'
export type ReleaseLagBucket = '<24h' | '1-7d' | '1-4w' | '1-6m' | '6-12m' | '>1y'
export interface SeriesEpisode { id: string; number: number | null; name: string; views: number; viewers: number; avgCompletion: number | null; abandons: number }
export interface SeriesDetail {
  item: { id: string; name: string; years: { from: number | null; to: number | null }; seasons: number; episodes: number; genres: string[]; library: LibraryRef | null }
  totals: { watchSeconds: number; views: number; clicks: number; viewers: number; retention: number | null; binges: number }
  /** Viewers per episode, one array per season (catalog order). */
  retention: number[][]
  /** Regular seasons in order, then specials (season 0, `special: true`). */
  seasons: { number: number; special?: boolean; episodes: SeriesEpisode[] }[]
  viewers: { user: RefUser; episodesWatched: number; lastWatched: ISOTime }[]
  cadence: { bucket: CadenceBucket; count: number }[]
  releaseLag: { bucket: ReleaseLagBucket; count: number }[]
}

export interface EpisodeDetail {
  item: {
    id: string; name: string; seriesId: string; seriesName: string; season: number | null; episode: number | null
    runtimeSeconds: number; premiereDate: ISODate | null; library: LibraryRef | null
    prev: EpisodeRef | null; next: EpisodeRef | null
  }
  totals: { views: number; viewers: number; avgCompletion: number | null; pausedSecondsAvg: number | null }
  funnel: Funnel
  /** 20 bins over the runtime. */
  stopPositions: number[]
  viewers: { user: RefUser; startedAt: ISOTime; watchSeconds: number; completion: number | null; device: string }[]
}

export interface TrackStat { id: string; name: string; albumId: string; albumName: string; views: number; clicks: number; skipRate: number | null; listeners: number }
export interface ArtistDetail {
  item: { id: string; name: string; genres: string[]; albums: number; tracks: number; library: LibraryRef | null }
  totals: { listens: number; watchSeconds: number; listeners: number; skipRate: number | null }
  topTracks: TrackStat[]
  albums: { id: string; name: string; year: number | null; tracks: number; listens: number }[]
  /** Skip rate (0..1) per local hour. */
  skipByHour: number[]
}

export interface AlbumDetail {
  item: { id: string; name: string; artistId: string | null; artistName: string | null; year: number | null; tracks: number; runtimeSeconds: number; library: LibraryRef | null }
  totals: { listens: number; fullListenRatio: number | null; fullAlbumListens: number; listeners: number; skipRate: number | null }
  tracks: { id: string; number: number | null; name: string; runtimeSeconds: number; views: number; clicks: number; skipRate: number | null; listeners: number }[]
}

export interface TrackDetail {
  item: { id: string; name: string; albumId: string | null; albumName: string | null; artistId: string | null; artistName: string | null }
  totals: { views: number; clicks: number; skipRate: number | null; listeners: number }
  recent: { user: RefUser; startedAt: ISOTime; watchSeconds: number }[]
}

// ── /libraries ───────────────────────────────────────────────────────────────
export type LibraryType = 'movies' | 'tvshows' | 'music'
export interface LibraryCard { id: string; name: string; type: LibraryType; items: number; sizeBytes: number; totals: Totals }
export interface LibraryItem { id: string; kind: GroupKind; name: string; views: number; viewers: number; watchSeconds: number; retention?: number; trend: number[] }
export interface LibraryDetail {
  library: LibraryCard
  items: Paged<LibraryItem>
  genres: { name: string; watchSeconds: number; views: number }[]
  methods: MethodSplit
  unwatched: { count: number; list: { id: string; kind: GroupKind; name: string; dateAdded: ISODate | null }[] }
  timeToWatch: { count: number; avgDays: number | null; medianDays: number | null }
}
