# API HTTP de Jellystics

Le tableau de bord utilise cette même API. La plupart des routes exigent une session du tableau de bord (`Authorization: Bearer <session token>`, obtenue en se connectant avec un administrateur Jellyfin) ; `/metrics` utilise une [clé d'API](/fr/prometheus/).

## Statistiques : `/api/v2` {#statistics-api-v2}

Paramètres communs :

| Paramètre | Description |
|---|---|
| `from`, `to` | Dates locales `YYYY-MM-DD`, `to` exclu. Par défaut : les 30 derniers jours. Les plages de plus de 366 jours commencent à la première lecture enregistrée. |
| `compare` | `1` ajoute la période précédente de même durée. |
| `userId`, `libraryId` | Restreint à un utilisateur ou à une bibliothèque. |
| `type` | `movie`, `series` ou `music`. |
| `page`, `pageSize` | Pagination, `pageSize` vaut 25 par défaut, 200 au plus. |

| Groupe | Routes |
|---|---|
| Vue d'ensemble | `overview`, `overview/plays`, `overview/libraries-daily` |
| Historique | `history`, `history.csv`, `history/calendar` |
| Médias | `movies/:id`, `series/:id`, `episodes/:id`, `artists/:id`, `albums/:id`, `tracks/:id`, `search`, `items/:id/route` |
| Widgets des médias | `movies/:id/sittings`, `items/:id/marks`, `albums/:id/full-listens` |
| Bibliothèques | `libraries`, `libraries/:id`, `libraries/:id/decades`, `libraries/:id/growth`, `libraries/:id/first-view-delay` |
| Utilisateurs | `users`, `users/:id`, `users/retention`, `users/share`, `users/:id/languages`, `users/:id/logins`, `concurrency`, `recap/:userId/:period` |
| Analyses | `insights/completion`, `insights/sessions`, `insights/resumes`, `insights/rewatches`, `insights/genres`, `insights/moments`, `insights/pauses`, `insights/binges`, `insights/rhythm`, `insights/skips` |
| Découverte | `discover/unwatched`, `discover/abandoned`, `discover/added`, `discover/storage` |
| Lecture | `playback/methods`, `playback/clients`, `playback/quality`, `playback/network` |
| Sources | `data-quality` |

Exemple :

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "http://jellystics:3000/api/v2/overview?from=2026-11-01&to=2026-12-01&compare=1"
```

## Administration {#administration}

| Groupe | Routes |
|---|---|
| Authentification | `POST /auth/jellyfin-login`, `POST /auth/logout`, `GET /auth/me`, `GET /auth/isConfigured` (sert aussi de health check au conteneur) |
| Configuration | `GET /api/getconfig`, `POST /api/setconfig`, `GET/POST/DELETE /api/keys` |
| Tâches | `GET /api/getTasks`, `POST /api/runTask/:name`, `GET /api/getTaskSettings`, `POST /api/setTaskSettings` |
| Synchronisation | `POST /sync/full`, `POST /sync/libraries`, `POST /sync/users`, `GET /sync/status`, `POST /sync/importPlaybackBackup`, `POST /sync/fetchItem` |
| Plugin | `GET /api/plugin/status`, `POST /api/plugin/push-history`, `GET /api/plugin/push-history/status`, `POST /api/plugin/resync` |
| Sauvegardes | `GET /backup/files`, `GET /backup/files/:name`, `GET /backup/beginBackup`, `GET /backup/restore/:name`, `POST /backup/upload`, `DELETE /backup/files/:name` |
| Webhooks | `GET/POST /webhooks/`, `GET/PUT/DELETE /webhooks/:id`, `POST /webhooks/:id/test` |
| Journaux | `GET /logs/getLogs` |
| En direct | `GET /sessions/current`, Socket.IO sur `/socket.io` (événements `sessions`, `TaskLog`, résultats des tâches) |

## Prometheus {#prometheus}

`GET /metrics` avec `Authorization: Bearer <api key>`. Voir la [référence des métriques](/fr/prometheus/metrics).

## « Mes statistiques » : `/api/me` {#my-statistics-api-me}

Appelée uniquement par le plugin, avec `X-Jellystics-Secret` et `X-Jellystics-User` : `health`, `profile`, `overview`, `top`, `history`, `types`, `habits`, `devices`, `scatter`. Les sessions du tableau de bord n'y sont pas acceptées.
