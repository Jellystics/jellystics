# API du plugin

Toutes les routes sont servies par Jellyfin sous `/Jellystics`. À l'exception des routes « Mes statistiques », elles exigent une session **administrateur** Jellyfin ou une **clé d'API** Jellyfin (politique `RequiresElevation`), par exemple :

```bash
curl -H 'Authorization: MediaBrowser Token="<jellyfin api key>"' \
  http://jellyfin:8096/Jellystics/Info
```

Le JSON utilise le camelCase ; les dates sont au format ISO 8601 en UTC.

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

`meError` vaut `null`, `unreachable` ou `secretMismatch`. L'empreinte correspond aux 16 premiers caractères hexadécimaux du SHA-256 du secret.

## Lire les données {#reading-data}

| Route | Contenu |
|---|---|
| `GET /Jellystics/Plays` | Lectures, y compris celles encore ouvertes. |
| `GET /Jellystics/LibraryEvents` | `present`, `added`, `removed`. |
| `GET /Jellystics/UserDataEvents` | Marques vu / favori. |
| `GET /Jellystics/Logins` | Connexions. |

Paramètres : `since` (curseur, 0 par défaut) et `limit` (500 par défaut, de 1 à 1000). Réponse :

```json
{ "items": [ … ], "next": 58231, "hasMore": false }
```

Passez `next` comme `since` pour lire la page suivante. Les lignes purgées ne sont pas renvoyées.

## Relier Jellystics {#linking-jellystics}

| Route | Corps | Effet |
|---|---|---|
| `POST /Jellystics/Link` | `{ "url": "http://…", "secret": "…" }` | Enregistre l'URL et le secret de Jellystics (16 à 1024 caractères, sans espace), puis vérifie `GET <url>/api/me/health`. Répond `{ linked, reachable, error }`. |
| `POST /Jellystics/LinkOverride` | `{ "url": "http://…" }` ou `{ "url": "" }` | Définit ou efface l'URL de remplacement définie par l'administrateur. Répond `{ saved, effectiveUrl, linked, reachable, error }`. |

Les URL doivent être des URL `http`/`https` absolues, sans identifiants, paramètres de requête ni fragment.

## Import et purge {#import-and-purge}

| Route | Corps | Réponse |
|---|---|---|
| `POST /Jellystics/Import/Plays` | `{ "items": [ play, … ] }`, 500 au plus | `{ received, imported, skipped, rejected }`. Les lignes sans identifiant, utilisateur, contenu ou début sont rejetées ; les identifiants existants sont ignorés (jamais écrasés). |
| `POST /Jellystics/Import/PlaybackReporting` | – | `202 { "started": true }` ; 404 si le fichier est absent, 409 si un import est en cours. |
| `GET /Jellystics/Import/Status` | – | `State` (`idle`, `running`, `done`, `failed`), `Read`, `Imported`, `Skipped`, `Repaired`, `Error`, `FinishedAt`. |
| `POST /Jellystics/Purge` | `{ "before": "2025-01-01T00:00:00Z" }` | `{ "deleted": 1234 }` (lectures clôturées uniquement). |

## « Mes statistiques » {#my-statistics}

| Route | Authentification | Rôle |
|---|---|---|
| `GET /Jellystics/Me/client.js` | aucune | Script qui ajoute l'entrée de menu. |
| `GET /Jellystics/Me/{path}` | aucune | Page et ressources relayées depuis `/me/` de Jellystics. |
| `GET /Jellystics/Me/api/{path}` | Utilisateur Jellyfin | Statistiques de l'utilisateur connecté, relayées vers `/api/me/{path}` de Jellystics avec le secret. |

Erreurs du relais : `503 notLinked`, `502 unreachable`, `504` délai dépassé, `502 secretMismatch`, `502 badUpstream`, `400 badPath`, `401 noUser` (les clés d'API n'ont pas d'utilisateur).
