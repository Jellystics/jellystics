---
aside: false
---

# How it works

This page follows a playback from the moment someone presses *Play* until it shows up in a chart and in Prometheus.

<Workflow />

## 1. Recording a playback

There are two recorders. The plugin is used when it is healthy; otherwise the built-in tracker takes over.

### With the plugin (event-driven)

The plugin listens to Jellyfin's own events (`PlaybackStart`, `PlaybackProgress`, `PlaybackStopped`) and keeps one state machine per playback. Each event adds:

- **watched time** = the smaller of the real time elapsed and the progress of the position (+ 2 s margin). Pauses, buffering and backward seeks add nothing;
- **paused time** while the player is paused, and one **pause** per playing → paused switch;
- a **seek** when the position jumps more than 5 s beyond what playback explains, with the **skipped time** for forward seeks.

Open plays are saved every 15 s; after a Jellyfin crash a play is closed at its last checkpoint. Full details: [Playback tracking](/plugin/tracking).

### Without the plugin (polling)

Jellystics calls Jellyfin `/Sessions` every **5 seconds** and applies the same accounting:

- each poll adds the forward progress of the position, capped at the elapsed time + 2 s;
- if a client that normally reports every ≤ 30 s goes silent for more than 45 s, time stops counting;
- when the play stops, the time since the last poll is recovered from the position Jellyfin saved (at most 55 s);
- a new play starts when the item or the user changes, or when the position goes back to the start after the end of the previous one (replay).

Only movies, episodes and music tracks are tracked; trailers, extras and theme media are ignored.

## 2. Getting plugin data into Jellystics

Every **30 seconds** Jellystics asks the plugin for what changed since last time, one cursor per stream (plays, library events, user data events, logins), 500 rows per page. A cursor is a global version number: each insert or update in the plugin gets a new one, so plays in progress are re-sent as they grow.

- **Healthy** = the plugin answered successfully less than **2 minutes** ago.
- While the plugin is healthy, the tracker does not write plays; the plugin owns them.
- When a plugin play arrives, any tracker row for the same user, item and device that it covers is deleted (10 s slack), and the same clean-up runs on the last 24 hours after each sync.
- When the plugin is down, tracker plays are kept, unless a closed plugin play already covers them. When the plugin comes back, its closed version replaces the tracker row.
- If the plugin database was recreated (new database id, or a cursor that goes backwards), Jellystics resets its cursors and re-reads everything; upserts are idempotent.

Details: [Sync with Jellystics](/plugin/sync).

## 3. From plays to statistics

All plays, whatever their source, land in one table. Statistics are computed in SQL at query time, in the `TZ` time zone:

| Notion | Rule |
|---|---|
| Click | Any recorded play. |
| View | A play with **≥ 30 s** really watched. |
| Watch time | Sum of the watched time of **all** plays (clicks included). |
| Completion | `max(stop position, watched time) / runtime`, capped at 100 %, per user, title and local day. |
| Finished view | Completion ≥ **90 %**, or, for movies and episodes of 5 min or more recorded by the plugin, Jellyfin's “played to completion” flag. |
| Binge | ≥ **3 distinct episodes** of one series by one user on one local day. |

The full list is in the [glossary](/guide/glossary).

## 4. Sources and precision

Each play keeps its source, visible in *Settings → Configuration → Data quality*:

| Source | Precision |
|---|---|
| Jellystics plugin | Exact events, pauses, seeks, tracks, transcoding. |
| Jellystics tracker | 5 s polling. |
| Playback Reporting imports | Start time estimated (logged time − 20 s), no pauses; impossible durations repaired (negative → 0, > 1.5 × runtime → runtime, > 12 h with unknown runtime → 12 h). |
| Other imports, restored backups | As recorded by the original tool. |

Imports never create duplicates: a play already recorded by another source (same user and item, start within 2 min, similar duration) is skipped.

## 5. The rest of the data

- **Libraries, items, users** come from the Jellyfin API (full sync, recently added sync, and on demand when a new item is played).
- **Library events, logins and played/favourite marks** come from the plugin only.
- **Remote or local**: from the client IP; private, loopback and link-local addresses are local. Remote IPs can be geolocated with MaxMind GeoLite (optional).
- **Live sessions** are pushed to the dashboard over a WebSocket every 5 s.

## 6. Exposing the data

- The **dashboard** reads `/api/v2/*`.
- **Prometheus** reads [`/metrics`](/prometheus/metrics), computed live at each scrape.
- **Jellyfin users** read their own numbers through the plugin: [“My statistics”](/plugin/my-statistics).
