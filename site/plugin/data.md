# Collected data

The plugin stores everything in a SQLite database:

```txt
<jellyfin config>/plugins/Jellyfin.Plugin.Jellystics/jellystics.db
```

The database runs in WAL mode. Nothing is deleted automatically: only the manual [purge](/plugin/configuration#purge-plays-before) removes plays. A play takes about 1 KB.

## Plays

One row per play.

| Group | Columns |
|---|---|
| Identity | `Id`, `PlaySessionId`, `UserId`, `UserName`, `ItemId`, `ItemName`, `ItemType` (Movie, Episode, Audio…), `SeriesId`, `SeriesName`, `SeasonId`, `AlbumId`, `AlbumName`, `ParentIndexNumber`, `IndexNumber`, `MediaSourceId`, `LibraryId` |
| Time | `StartedAt`, `EndedAt` (UTC), `WatchedMs`, `PausedMs`, `PauseCount`, `SeekCount`, `SkippedMs`, `PositionStartTicks`, `PositionEndTicks`, `RunTimeTicks`, `PlayedToCompletion` |
| Context | `DeviceId`, `DeviceName`, `Client`, `ApplicationVersion`, `RemoteEndPoint`, `IsRemote`, `PlayMethod` (DirectPlay, DirectStream, Transcode) |
| Streams | `AudioStreamIndex`, `AudioLanguage`, `AudioCodec`, `SubtitleStreamIndex`, `SubtitleLanguage`, `StreamChanges` (JSON) |
| Transcoding | `VideoCodecFrom`, `VideoCodecTo`, `AudioCodecFrom`, `AudioCodecTo`, `IsVideoDirect`, `IsAudioDirect`, `Width`, `Height`, `SourceWidth`, `SourceHeight`, `Bitrate`, `Framerate`, `HwAccel`, `TranscodeReasons` (JSON), `VideoRange`, `VideoRangeType` |
| State | `State` (`open`, `closed`), `LastEventAt`, `Source` (`plugin`, `import-pbr`, `import-jellystics`), `DurationRepaired`, `Version` |

## Library events

Additions and removals of movies, episodes, music tracks, series and albums (virtual items and extras excluded).

| Kind | When |
|---|---|
| `present` | Snapshot of the existing catalog, taken once on the first start (pages of 1000 items). |
| `added` | An item appeared in a library (`DateCreated` = Jellyfin's creation date). |
| `removed` | An item was removed. |

Each event keeps the item, its type, series or album, library, runtime and file size. Jellystics uses them for real “added” dates and for removals in *growth vs consumption*.

## User data events

Changes of the **played** and **favourite** marks, with Jellyfin's reason:

| Reason | Recorded |
|---|---|
| `TogglePlayed` | Marked played / unplayed by hand. |
| `PlaybackFinished` | Marked played at the end of a playback. |
| `PlaybackStart` | Played, or set back to unplayed when a resumable item restarts. |
| `UpdateUserRating` | Favourite / unfavourite. |
| `UpdateUserData` | Played / unplayed and favourite changes from other clients. |

Only real changes are recorded, not every progress save. Jellystics shows them on movie and episode pages (“marked as played by hand by N users”).

## Logins

Successful and failed sign-ins: user, device, client and version, remote address, success. Shown on user pages (failures are visible to administrators).

## Versions and cursors

Every insert or update, in any table, takes the next value of one global counter (`Version`). A reader asks for “everything after version N” and gets rows in order: this is how Jellystics syncs without missing or re-reading data. Plays in progress get a new version at each checkpoint, so they are re-sent as they grow.

The database schema is versioned (`schema_version`, currently 2) and migrated automatically. A random `DatabaseId` identifies the database: if it changes (database deleted), Jellystics re-reads everything.
