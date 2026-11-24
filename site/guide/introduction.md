# Introduction

Jellystics is a statistics tracker for [Jellyfin](https://jellyfin.org). It answers questions such as *who watches what, when, on which device, how far, and how well it streams*, and keeps the answer for years.

![Overview page](/screenshots/en/overview.jpg){.shot}

## The two components

| Component | Where it runs | Role |
|---|---|---|
| **Jellystics** | Docker container (Go API, React dashboard) + PostgreSQL | Stores the history, computes the statistics, serves the dashboard, the API, `/metrics`, webhooks and backups. |
| **Jellystics plugin** | Inside Jellyfin 10.11 | Records every playback from server events into its own SQLite database, and relays the “My statistics” page to every Jellyfin user. |

Jellystics works on its own. The plugin is a precision upgrade and a backup:

| | Without the plugin | With the plugin |
|---|---|---|
| Source | Jellystics polls Jellyfin `/Sessions` every 5 s | Jellyfin pushes events to the plugin in real time |
| Watch time | Very good, limited by the 5 s polling | Exact, from playback events |
| Jellyfin restart | The play in progress is closed when it disappears | At most 15 s lost (checkpoints) |
| Forward skips / intro skipping | – | Counted, with skipped time |
| “Played to the end” | Estimated (≥ 90 % of runtime) | Reported by Jellyfin |
| Library additions and removals | Catalog dates (approximate) | Real events |
| Logins, “marked as played” by hand | – | Recorded |
| History backup inside Jellyfin | – | Full history in `jellystics.db` |
| “My statistics” for users | – | Yes |

When the plugin is healthy, it is the source of truth; when it stops answering, Jellystics switches back to its own tracker automatically, without gaps or duplicates. See [how it works](/guide/how-it-works).

## What you get

- **Dashboard**: overview with period comparison, live sessions, history (table and calendar), [libraries and media pages](/platform/libraries), [users and recaps](/platform/users), [insights, discover and playback health](/platform/analysis).
- **Precise definitions**: clicks, views (≥ 30 s), completion, binges, resumes, rewatches. See the [glossary](/guide/glossary).
- **Integrations**: [Prometheus metrics and Grafana](/prometheus/), Discord webhooks, an HTTP API with API keys.
- **Operations**: scheduled tasks, backups and restore, import of the official Playback Reporting history.
- **Three languages**: English, French and Spanish, in the dashboard and in this documentation.

## Requirements

- Jellyfin **10.11** (for the plugin; Jellystics itself only needs the Jellyfin HTTP API).
- Docker and Docker Compose for Jellystics.
- A Jellyfin **API key** and a Jellyfin **administrator** account to sign in to the dashboard.

## Next steps

1. [Install Jellystics](/guide/installation).
2. [Install the plugin](/plugin/).
3. [Plug Prometheus and Grafana](/prometheus/) if you use them.
