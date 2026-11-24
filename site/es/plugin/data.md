# Datos recopilados

El plugin lo guarda todo en una base de datos SQLite:

```txt
<jellyfin config>/plugins/Jellyfin.Plugin.Jellystics/jellystics.db
```

La base de datos funciona en modo WAL. Nada se elimina automáticamente: solo la [purga](/es/plugin/configuration#purge-plays-before) manual borra reproducciones. Una reproducción ocupa aproximadamente 1 KB.

## Reproducciones {#plays}

Una fila por reproducción.

| Grupo | Columnas |
|---|---|
| Identidad | `Id`, `PlaySessionId`, `UserId`, `UserName`, `ItemId`, `ItemName`, `ItemType` (Movie, Episode, Audio…), `SeriesId`, `SeriesName`, `SeasonId`, `AlbumId`, `AlbumName`, `ParentIndexNumber`, `IndexNumber`, `MediaSourceId`, `LibraryId` |
| Tiempo | `StartedAt`, `EndedAt` (UTC), `WatchedMs`, `PausedMs`, `PauseCount`, `SeekCount`, `SkippedMs`, `PositionStartTicks`, `PositionEndTicks`, `RunTimeTicks`, `PlayedToCompletion` |
| Contexto | `DeviceId`, `DeviceName`, `Client`, `ApplicationVersion`, `RemoteEndPoint`, `IsRemote`, `PlayMethod` (DirectPlay, DirectStream, Transcode) |
| Flujos | `AudioStreamIndex`, `AudioLanguage`, `AudioCodec`, `SubtitleStreamIndex`, `SubtitleLanguage`, `StreamChanges` (JSON) |
| Transcodificación | `VideoCodecFrom`, `VideoCodecTo`, `AudioCodecFrom`, `AudioCodecTo`, `IsVideoDirect`, `IsAudioDirect`, `Width`, `Height`, `SourceWidth`, `SourceHeight`, `Bitrate`, `Framerate`, `HwAccel`, `TranscodeReasons` (JSON), `VideoRange`, `VideoRangeType` |
| Estado | `State` (`open`, `closed`), `LastEventAt`, `Source` (`plugin`, `import-pbr`, `import-jellystics`), `DurationRepaired`, `Version` |

## Eventos de biblioteca {#library-events}

Altas y bajas de películas, episodios, pistas de música, series y álbumes (elementos virtuales y extras excluidos).

| Tipo | Cuándo |
|---|---|
| `present` | Instantánea del catálogo existente, tomada una sola vez en el primer arranque (páginas de 1000 elementos). |
| `added` | Un elemento apareció en una biblioteca (`DateCreated` = fecha de creación en Jellyfin). |
| `removed` | Se eliminó un elemento. |

Cada evento conserva el elemento, su tipo, la serie o el álbum, la biblioteca, la duración y el tamaño del archivo. Jellystics los usa para las fechas de incorporación reales y para las bajas en *crecimiento vs consumo*.

## Eventos de datos de usuario {#user-data-events}

Cambios de las marcas de **visto** y **favorito**, con el motivo de Jellyfin:

| Motivo | Qué se registra |
|---|---|
| `TogglePlayed` | Marcado como visto / no visto a mano. |
| `PlaybackFinished` | Marcado como visto al final de una reproducción. |
| `PlaybackStart` | Visto, o devuelto a no visto cuando un elemento reanudable vuelve a empezar. |
| `UpdateUserRating` | Favorito / quitado de favoritos. |
| `UpdateUserData` | Cambios de visto / no visto y de favorito desde otros clientes. |

Solo se registran los cambios reales, no cada guardado de progreso. Jellystics los muestra en las páginas de películas y episodios («marcado como visto a mano por N usuarios»).

## Inicios de sesión {#logins}

Inicios de sesión correctos y fallidos: usuario, dispositivo, cliente y versión, dirección remota, éxito. Se muestran en las páginas de usuario (los fallos son visibles para los administradores).

## Versiones y cursores {#versions-and-cursors}

Cada inserción o actualización, en cualquier tabla, toma el siguiente valor de un único contador global (`Version`). Un lector pide «todo lo posterior a la versión N» y recibe las filas en orden: así se sincroniza Jellystics sin perder ni releer datos. Las reproducciones en curso reciben una nueva versión en cada punto de control, por lo que se reenvían a medida que crecen.

El esquema de la base de datos está versionado (`schema_version`, actualmente 2) y se migra automáticamente. Un `DatabaseId` aleatorio identifica la base de datos: si cambia (base de datos eliminada), Jellystics lo vuelve a leer todo.
