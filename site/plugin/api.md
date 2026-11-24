# Plugin API

All routes are served by Jellyfin under `/Jellystics`. Except the “My statistics” routes, they require a Jellyfin **administrator** session or a Jellyfin **API key** (policy `RequiresElevation`), for example:

```bash
curl -H 'Authorization: MediaBrowser Token="<jellyfin api key>"' \
  http://jellyfin:8096/Jellystics/Info
```

JSON uses camelCase; dates are ISO 8601 in UTC.

## Info

`GET /Jellystics/Info`

```json
{
  "pluginVersion": "0.2.1.0",
  "schemaVersion": 1,
  "cursor": 58231,
  "databaseId": "0d6c…",
  "firstPlayAt": "2024-03-02T19:41:07Z",
  "counts": { "plays": 1840, "libraryEvents": 3512, "userDataEvents": 412, "logins": 96 },
  "serverTimeZone": "Europe/Paris",
  "meLinked": true,
  "meReachable": true,
  "meError": null,
  "meLinkedUrl": "http://192.168.1.10:3000",
  "meSecretFingerprint": "3f9a0c2b7d1e4a65"
}
```

`meError` is `null`, `unreachable` or `secretMismatch`. The fingerprint is the first 16 hex characters of the SHA-256 of the secret.

## Reading data

| Route | Content |
|---|---|
| `GET /Jellystics/Plays` | Plays, open ones included. |
| `GET /Jellystics/LibraryEvents` | `present`, `added`, `removed`. |
| `GET /Jellystics/UserDataEvents` | Played / favourite marks. |
| `GET /Jellystics/Logins` | Sign-ins. |

Parameters: `since` (cursor, default 0) and `limit` (default 500, 1 to 1000). Response:

```json
{ "items": [ … ], "next": 58231, "hasMore": false }
```

Pass `next` as `since` to read the following page. Purged rows are not reported.

## Linking Jellystics

| Route | Body | Effect |
|---|---|---|
| `POST /Jellystics/Link` | `{ "url": "http://…", "secret": "…" }` | Saves the Jellystics URL and secret (16 to 1024 characters, no spaces), then checks `GET <url>/api/me/health`. Answers `{ linked, reachable, error }`. |
| `POST /Jellystics/LinkOverride` | `{ "url": "http://…" }` or `{ "url": "" }` | Sets or clears the administrator URL override. Answers `{ saved, effectiveUrl, linked, reachable, error }`. |

URLs must be absolute `http`/`https`, without credentials, query or fragment.

## Import and purge

| Route | Body | Response |
|---|---|---|
| `POST /Jellystics/Import/Plays` | `{ "items": [ play, … ] }`, at most 500 | `{ received, imported, skipped, rejected }`. Rows without id, user, item or start are rejected; existing ids are skipped (never overwritten). |
| `POST /Jellystics/Import/PlaybackReporting` | – | `202 { "started": true }`; 404 if the file is missing, 409 if an import is running. |
| `GET /Jellystics/Import/Status` | – | `State` (`idle`, `running`, `done`, `failed`), `Read`, `Imported`, `Skipped`, `Repaired`, `Error`, `FinishedAt`. |
| `POST /Jellystics/Purge` | `{ "before": "2025-01-01T00:00:00Z" }` | `{ "deleted": 1234 }` (closed plays only). |

## “My statistics”

| Route | Auth | Role |
|---|---|---|
| `GET /Jellystics/Me/client.js` | none | Script that adds the menu entry. |
| `GET /Jellystics/Me/{path}` | none | Page and assets relayed from Jellystics `/me/`. |
| `GET /Jellystics/Me/api/{path}` | Jellyfin user | Statistics of the signed-in user, relayed to Jellystics `/api/me/{path}` with the secret. |

Relay errors: `503 notLinked`, `502 unreachable`, `504` time-out, `502 secretMismatch`, `502 badUpstream`, `400 badPath`, `401 noUser` (API keys have no user).
