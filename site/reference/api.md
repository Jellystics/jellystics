# Jellystics HTTP API

The dashboard uses the same API. Most routes need a dashboard session (`Authorization: Bearer <session token>`, obtained by signing in with a Jellyfin administrator); `/metrics` uses an [API key](/prometheus/).

## Statistics: `/api/v2`

Common parameters:

| Parameter | Description |
|---|---|
| `from`, `to` | Local dates `YYYY-MM-DD`, `to` excluded. Default: the last 30 days. Ranges over 366 days start at the first recorded play. |
| `compare` | `1` adds the previous period of the same length. |
| `userId`, `libraryId` | Restrict to a user or a library. |
| `type` | `movie`, `series` or `music`. |
| `page`, `pageSize` | Paging, `pageSize` 25 by default, 200 at most. |

| Group | Routes |
|---|---|
| Overview | `overview`, `overview/plays`, `overview/libraries-daily` |
| History | `history`, `history.csv`, `history/calendar` |
| Media | `movies/:id`, `series/:id`, `episodes/:id`, `artists/:id`, `albums/:id`, `tracks/:id`, `search`, `items/:id/route` |
| Media widgets | `movies/:id/sittings`, `items/:id/marks`, `albums/:id/full-listens` |
| Libraries | `libraries`, `libraries/:id`, `libraries/:id/decades`, `libraries/:id/growth`, `libraries/:id/first-view-delay` |
| Users | `users`, `users/:id`, `users/retention`, `users/share`, `users/:id/languages`, `users/:id/logins`, `concurrency`, `recap/:userId/:period` |
| Insights | `insights/completion`, `insights/sessions`, `insights/resumes`, `insights/rewatches`, `insights/genres`, `insights/moments`, `insights/pauses`, `insights/binges`, `insights/rhythm`, `insights/skips` |
| Discover | `discover/unwatched`, `discover/abandoned`, `discover/added`, `discover/storage` |
| Playback | `playback/methods`, `playback/clients`, `playback/quality`, `playback/network` |
| Sources | `data-quality` |

Example:

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "http://jellystics:3000/api/v2/overview?from=2026-11-01&to=2026-12-01&compare=1"
```

## Administration

| Group | Routes |
|---|---|
| Auth | `POST /auth/jellyfin-login`, `POST /auth/logout`, `GET /auth/me`, `GET /auth/isConfigured` (also the container health check) |
| Configuration | `GET /api/getconfig`, `POST /api/setconfig`, `GET/POST/DELETE /api/keys` |
| Tasks | `GET /api/getTasks`, `POST /api/runTask/:name`, `GET /api/getTaskSettings`, `POST /api/setTaskSettings` |
| Sync | `POST /sync/full`, `POST /sync/libraries`, `POST /sync/users`, `GET /sync/status`, `POST /sync/importPlaybackBackup`, `POST /sync/fetchItem` |
| Plugin | `GET /api/plugin/status`, `POST /api/plugin/push-history`, `GET /api/plugin/push-history/status`, `POST /api/plugin/resync` |
| Backups | `GET /backup/files`, `GET /backup/files/:name`, `GET /backup/beginBackup`, `GET /backup/restore/:name`, `POST /backup/upload`, `DELETE /backup/files/:name` |
| Webhooks | `GET/POST /webhooks/`, `GET/PUT/DELETE /webhooks/:id`, `POST /webhooks/:id/test` |
| Logs | `GET /logs/getLogs` |
| Live | `GET /sessions/current`, Socket.IO on `/socket.io` (events `sessions`, `TaskLog`, task results) |

## Prometheus

`GET /metrics` with `Authorization: Bearer <api key>`. See the [metrics reference](/prometheus/metrics).

## “My statistics”: `/api/me`

Called only by the plugin, with `X-Jellystics-Secret` and `X-Jellystics-User`: `health`, `profile`, `overview`, `top`, `history`, `types`, `habits`, `devices`, `scatter`. Dashboard sessions are not accepted there.
