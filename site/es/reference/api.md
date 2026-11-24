# API HTTP de Jellystics

El panel usa la misma API. La mayoría de las rutas requieren una sesión del panel (`Authorization: Bearer <session token>`, obtenida al iniciar sesión con un administrador de Jellyfin); `/metrics` usa una [clave API](/es/prometheus/).

## Estadísticas: `/api/v2` {#statistics-api-v2}

Parámetros comunes:

| Parámetro | Descripción |
|---|---|
| `from`, `to` | Fechas locales `YYYY-MM-DD`, `to` excluida. Por defecto: los últimos 30 días. Los rangos de más de 366 días empiezan en la primera reproducción registrada. |
| `compare` | `1` añade el periodo anterior de la misma duración. |
| `userId`, `libraryId` | Restringe a un usuario o a una biblioteca. |
| `type` | `movie`, `series` o `music`. |
| `page`, `pageSize` | Paginación, `pageSize` 25 por defecto, 200 como máximo. |

| Grupo | Rutas |
|---|---|
| Resumen | `overview`, `overview/plays`, `overview/libraries-daily` |
| Historial | `history`, `history.csv`, `history/calendar` |
| Contenido | `movies/:id`, `series/:id`, `episodes/:id`, `artists/:id`, `albums/:id`, `tracks/:id`, `search`, `items/:id/route` |
| Widgets de contenido | `movies/:id/sittings`, `items/:id/marks`, `albums/:id/full-listens` |
| Bibliotecas | `libraries`, `libraries/:id`, `libraries/:id/decades`, `libraries/:id/growth`, `libraries/:id/first-view-delay` |
| Usuarios | `users`, `users/:id`, `users/retention`, `users/share`, `users/:id/languages`, `users/:id/logins`, `concurrency`, `recap/:userId/:period` |
| Análisis | `insights/completion`, `insights/sessions`, `insights/resumes`, `insights/rewatches`, `insights/genres`, `insights/moments`, `insights/pauses`, `insights/binges`, `insights/rhythm`, `insights/skips` |
| Descubrir | `discover/unwatched`, `discover/abandoned`, `discover/added`, `discover/storage` |
| Reproducción | `playback/methods`, `playback/clients`, `playback/quality`, `playback/network` |
| Fuentes | `data-quality` |

Ejemplo:

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "http://jellystics:3000/api/v2/overview?from=2026-11-01&to=2026-12-01&compare=1"
```

## Administración {#administration}

| Grupo | Rutas |
|---|---|
| Autenticación | `POST /auth/jellyfin-login`, `POST /auth/logout`, `GET /auth/me`, `GET /auth/isConfigured` (también la comprobación de salud del contenedor) |
| Configuración | `GET /api/getconfig`, `POST /api/setconfig`, `GET/POST/DELETE /api/keys` |
| Tareas | `GET /api/getTasks`, `POST /api/runTask/:name`, `GET /api/getTaskSettings`, `POST /api/setTaskSettings` |
| Sincronización | `POST /sync/full`, `POST /sync/libraries`, `POST /sync/users`, `GET /sync/status`, `POST /sync/importPlaybackBackup`, `POST /sync/fetchItem` |
| Plugin | `GET /api/plugin/status`, `POST /api/plugin/push-history`, `GET /api/plugin/push-history/status`, `POST /api/plugin/resync` |
| Copias de seguridad | `GET /backup/files`, `GET /backup/files/:name`, `GET /backup/beginBackup`, `GET /backup/restore/:name`, `POST /backup/upload`, `DELETE /backup/files/:name` |
| Webhooks | `GET/POST /webhooks/`, `GET/PUT/DELETE /webhooks/:id`, `POST /webhooks/:id/test` |
| Registros | `GET /logs/getLogs` |
| En directo | `GET /sessions/current`, Socket.IO en `/socket.io` (eventos `sessions`, `TaskLog`, resultados de tareas) |

## Prometheus {#prometheus}

`GET /metrics` con `Authorization: Bearer <api key>`. Consulta la [referencia de métricas](/es/prometheus/metrics).

## «Mis estadísticas»: `/api/me` {#my-statistics-api-me}

Solo la llama el plugin, con `X-Jellystics-Secret` y `X-Jellystics-User`: `health`, `profile`, `overview`, `top`, `history`, `types`, `habits`, `devices`, `scatter`. Ahí no se aceptan las sesiones del panel.
