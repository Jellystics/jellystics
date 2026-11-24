# API del plugin

Jellyfin sirve todas las rutas bajo `/Jellystics`. Salvo las rutas de «Mis estadísticas», requieren una sesión de **administrador** de Jellyfin o una **clave API** de Jellyfin (política `RequiresElevation`), por ejemplo:

```bash
curl -H 'Authorization: MediaBrowser Token="<jellyfin api key>"' \
  http://jellyfin:8096/Jellystics/Info
```

El JSON usa camelCase; las fechas están en ISO 8601 en UTC.

## Info {#info}

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

`meError` vale `null`, `unreachable` o `secretMismatch`. La huella son los 16 primeros caracteres hexadecimales del SHA-256 del secreto.

## Lectura de datos {#reading-data}

| Ruta | Contenido |
|---|---|
| `GET /Jellystics/Plays` | Reproducciones, abiertas incluidas. |
| `GET /Jellystics/LibraryEvents` | `present`, `added`, `removed`. |
| `GET /Jellystics/UserDataEvents` | Marcas de visto / favorito. |
| `GET /Jellystics/Logins` | Inicios de sesión. |

Parámetros: `since` (cursor, 0 por defecto) y `limit` (500 por defecto, de 1 a 1000). Respuesta:

```json
{ "items": [ … ], "next": 58231, "hasMore": false }
```

Pasa `next` como `since` para leer la página siguiente. Las filas purgadas no se devuelven.

## Vincular Jellystics {#linking-jellystics}

| Ruta | Cuerpo | Efecto |
|---|---|---|
| `POST /Jellystics/Link` | `{ "url": "http://…", "secret": "…" }` | Guarda la URL y el secreto de Jellystics (de 16 a 1024 caracteres, sin espacios) y luego comprueba `GET <url>/api/me/health`. Responde `{ linked, reachable, error }`. |
| `POST /Jellystics/LinkOverride` | `{ "url": "http://…" }` o `{ "url": "" }` | Define o borra la URL override del administrador. Responde `{ saved, effectiveUrl, linked, reachable, error }`. |

Las URL deben ser `http`/`https` absolutas, sin credenciales, query ni fragmento.

## Importación y purga {#import-and-purge}

| Ruta | Cuerpo | Respuesta |
|---|---|---|
| `POST /Jellystics/Import/Plays` | `{ "items": [ play, … ] }`, como máximo 500 | `{ received, imported, skipped, rejected }`. Las filas sin id, usuario, elemento o inicio se rechazan; los ids existentes se omiten (nunca se sobrescriben). |
| `POST /Jellystics/Import/PlaybackReporting` | – | `202 { "started": true }`; 404 si falta el archivo, 409 si ya hay una importación en curso. |
| `GET /Jellystics/Import/Status` | – | `State` (`idle`, `running`, `done`, `failed`), `Read`, `Imported`, `Skipped`, `Repaired`, `Error`, `FinishedAt`. |
| `POST /Jellystics/Purge` | `{ "before": "2025-01-01T00:00:00Z" }` | `{ "deleted": 1234 }` (solo reproducciones cerradas). |

## «Mis estadísticas» {#my-statistics}

| Ruta | Autenticación | Función |
|---|---|---|
| `GET /Jellystics/Me/client.js` | ninguna | Script que añade la entrada de menú. |
| `GET /Jellystics/Me/{path}` | ninguna | Página y recursos reenviados desde `/me/` de Jellystics. |
| `GET /Jellystics/Me/api/{path}` | usuario de Jellyfin | Estadísticas del usuario conectado, reenviadas a `/api/me/{path}` de Jellystics con el secreto. |

Errores de reenvío: `503 notLinked`, `502 unreachable`, `504` tiempo de espera agotado, `502 secretMismatch`, `502 badUpstream`, `400 badPath`, `401 noUser` (las claves API no tienen usuario).
