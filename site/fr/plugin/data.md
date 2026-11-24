# Données collectées

Le plugin stocke tout dans une base de données SQLite :

```txt
<jellyfin config>/plugins/Jellyfin.Plugin.Jellystics/jellystics.db
```

La base fonctionne en mode WAL. Rien n'est supprimé automatiquement : seule la [purge](/fr/plugin/configuration#purge-plays-before) manuelle retire des lectures. Une lecture occupe environ 1 Ko.

## Lectures {#plays}

Une ligne par lecture.

| Groupe | Colonnes |
|---|---|
| Identité | `Id`, `PlaySessionId`, `UserId`, `UserName`, `ItemId`, `ItemName`, `ItemType` (Movie, Episode, Audio…), `SeriesId`, `SeriesName`, `SeasonId`, `AlbumId`, `AlbumName`, `ParentIndexNumber`, `IndexNumber`, `MediaSourceId`, `LibraryId` |
| Temps | `StartedAt`, `EndedAt` (UTC), `WatchedMs`, `PausedMs`, `PauseCount`, `SeekCount`, `SkippedMs`, `PositionStartTicks`, `PositionEndTicks`, `RunTimeTicks`, `PlayedToCompletion` |
| Contexte | `DeviceId`, `DeviceName`, `Client`, `ApplicationVersion`, `RemoteEndPoint`, `IsRemote`, `PlayMethod` (DirectPlay, DirectStream, Transcode) |
| Flux | `AudioStreamIndex`, `AudioLanguage`, `AudioCodec`, `SubtitleStreamIndex`, `SubtitleLanguage`, `StreamChanges` (JSON) |
| Transcodage | `VideoCodecFrom`, `VideoCodecTo`, `AudioCodecFrom`, `AudioCodecTo`, `IsVideoDirect`, `IsAudioDirect`, `Width`, `Height`, `SourceWidth`, `SourceHeight`, `Bitrate`, `Framerate`, `HwAccel`, `TranscodeReasons` (JSON), `VideoRange`, `VideoRangeType` |
| État | `State` (`open`, `closed`), `LastEventAt`, `Source` (`plugin`, `import-pbr`, `import-jellystics`), `DurationRepaired`, `Version` |

## Événements de bibliothèque {#library-events}

Ajouts et suppressions de films, d'épisodes, de pistes musicales, de séries et d'albums (contenus virtuels et bonus exclus).

| Type | Quand |
|---|---|
| `present` | Instantané du catalogue existant, pris une seule fois au premier démarrage (par pages de 1000 contenus). |
| `added` | Un contenu est apparu dans une bibliothèque (`DateCreated` = date de création dans Jellyfin). |
| `removed` | Un contenu a été supprimé. |

Chaque événement conserve le contenu, son type, sa série ou son album, sa bibliothèque, sa durée et la taille de son fichier. Jellystics s'en sert pour les vraies dates d'ajout et pour les suppressions dans *croissance vs consommation*.

## Événements de données utilisateur {#user-data-events}

Changements des marques **vu** et **favori**, avec la raison fournie par Jellyfin :

| Raison | Enregistré |
|---|---|
| `TogglePlayed` | Marqué vu / non vu à la main. |
| `PlaybackFinished` | Marqué vu à la fin d'une lecture. |
| `PlaybackStart` | Lu, ou remis à non vu quand un contenu reprenable redémarre. |
| `UpdateUserRating` | Ajouté aux / retiré des favoris. |
| `UpdateUserData` | Changements vu / non vu et favori venant d'autres clients. |

Seuls les vrais changements sont enregistrés, pas chaque sauvegarde de progression. Jellystics les affiche sur les pages des films et des épisodes (« Marqué vu manuellement par N utilisateurs »).

## Connexions {#logins}

Connexions réussies et échouées : utilisateur, appareil, client et version, adresse distante, succès. Affichées sur les pages utilisateur (les échecs sont visibles par les administrateurs).

## Versions et curseurs {#versions-and-cursors}

Chaque insertion ou mise à jour, dans n'importe quelle table, prend la valeur suivante d'un compteur global unique (`Version`). Un lecteur demande « tout ce qui suit la version N » et obtient les lignes dans l'ordre : c'est ainsi que Jellystics se synchronise sans rien manquer ni relire. Les lectures en cours reçoivent une nouvelle version à chaque point de sauvegarde, et sont donc renvoyées au fur et à mesure qu'elles s'allongent.

Le schéma de la base est versionné (`schema_version`, actuellement 2) et migré automatiquement. Un `DatabaseId` aléatoire identifie la base : s'il change (base supprimée), Jellystics relit tout.
