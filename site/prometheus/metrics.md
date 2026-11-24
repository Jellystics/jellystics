# Metrics reference

All metrics come from the same SQL definitions as the dashboard (see the [glossary](/guide/glossary)). “Views” means plays of at least 30 seconds.

## Totals

| Metric | Type | Labels | Description |
|---|---|---|---|
| `jellyfin_total_plays` | counter | – | Number of plays, all time. |
| `jellyfin_total_duration_seconds` | counter | – | Watch time of all plays, in seconds. |
| `jellyfin_total_users` | gauge | – | Number of users. |
| `jellyfin_total_libraries` | gauge | – | Number of libraries. |

## Per user, library, type, client and method

| Metric | Type | Labels | Description |
|---|---|---|---|
| `jellyfin_plays_by_user_total` | counter | `user` | Views per user (top 500). |
| `jellyfin_hours_by_user_total` | counter | `user` | Watch hours per user (top 500). |
| `jellyfin_plays_by_library_total` | counter | `library` | Views per library (top 100, archived libraries excluded). |
| `jellyfin_plays_by_media_type_total` | counter | `type` | Views per type: `Movie`, `Series` (episodes), `Audio`. |
| `jellyfin_plays_by_client_total` | counter | `client` | Views per client application (top 50). |
| `jellyfin_plays_by_playback_method_total` | counter | `method` | Views per play method (`DirectPlay`, `DirectStream`, `Transcode`). |

When two users or libraries share a name, the second one gets a ` (2)` suffix so the series stay distinct.

## Time patterns

| Metric | Type | Labels | Description |
|---|---|---|---|
| `jellyfin_plays_by_hour` | gauge | `hour` (0–23) | Views per hour of the day, all time. |
| `jellyfin_plays_by_day_of_week` | gauge | `day` (0 = Sunday … 6) | Views per weekday, all time. |
| `jellyfin_heatmap_plays` | gauge | `day`, `hour` | Views per weekday × hour, for a heatmap. |

## Content

| Metric | Type | Labels | Description |
|---|---|---|---|
| `jellyfin_item_play_count` | gauge | `name`, `type` | Views of the top 50 titles (movie, series or album level). |
| `jellyfin_items_by_type_total` | gauge | `type` | Items in the libraries: `Movie`, `Series`, `Episode`, `Audio`. |
| `jellyfin_unwatched_items_by_type` | gauge | `type` | Items never viewed, per type. |

## Completion and habits

| Metric | Type | Labels | Description |
|---|---|---|---|
| `jellyfin_completed_plays_total` | counter | – | Finished completion units (≥ 90 %, or Jellyfin's “played to completion” for plugin plays). One unit = one user, title and local day. |
| `jellyfin_abandoned_plays_total` | counter | – | Completion units under 90 %. |
| `jellyfin_completion_rate_percent` | gauge | – | `completed ÷ (completed + abandoned) × 100`. |
| `jellyfin_binge_sessions_total` | counter | – | (user, series, day) with at least 3 distinct episodes viewed. |

## Live

| Metric | Type | Labels | Description |
|---|---|---|---|
| `jellyfin_active_sessions` | gauge | – | Movies, episodes and tracks being played right now. |

## Sample output

```txt
# HELP jellyfin_total_plays Total number of plays all time
# TYPE jellyfin_total_plays counter
jellyfin_total_plays 4507
# HELP jellyfin_plays_by_media_type_total Total plays per media type
# TYPE jellyfin_plays_by_media_type_total counter
jellyfin_plays_by_media_type_total{type="Audio"} 612
jellyfin_plays_by_media_type_total{type="Movie"} 980
jellyfin_plays_by_media_type_total{type="Series"} 1820
# HELP jellyfin_active_sessions Number of currently active playback sessions
# TYPE jellyfin_active_sessions gauge
jellyfin_active_sessions 2
```

## PromQL examples

```txt
# Views over the last 24 hours
increase(jellyfin_total_plays[24h])

# Hours watched per day, per user
increase(jellyfin_hours_by_user_total[1d])

# Top 5 users by views
topk(5, jellyfin_plays_by_user_total)

# Transcode share (%)
100 * jellyfin_plays_by_playback_method_total{method="Transcode"}
  / ignoring(method) sum without(method) (jellyfin_plays_by_playback_method_total)

# Share of the catalog never watched, per type
100 * jellyfin_unwatched_items_by_type / jellyfin_items_by_type_total

# Busiest hour
topk(1, jellyfin_plays_by_hour)
```
