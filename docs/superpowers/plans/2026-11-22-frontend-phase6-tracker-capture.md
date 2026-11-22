# Phase 6 — Tracker capture of new playback fields

**Spec:** `docs/superpowers/specs/2026-11-22-frontend-redesign-design.md` §8.3. Design « Nouveautés › Données à capter ».

**Goal:** record, for every play, the audio/subtitle choice, transcode reasons, bitrate, hardware acceleration, video range, pause count and local/remote, so the phase 5 views (VO/VF, raisons du transcodage, débit, HDR, local/distant, pauses) have data.

## Global constraints
- Commits one-line Conventional Commits, no attribution. `cd backend && POSTGRES_PORT=55432 go test ./...` green before each commit.
- Do not change the accounting logic of `backend/internal/service/sync/tracker.go` (watched time, head/tail, replay, silent client) — only add capture of new fields. All existing tracker tests must stay green.
- Verify field names/shapes against Jellyfin source (`gh api repos/jellyfin/jellyfin/contents/<path>?ref=master -H "Accept: application/vnd.github.raw"`): `MediaBrowser.Model/Session/PlayerStateInfo.cs` (AudioStreamIndex, SubtitleStreamIndex), `MediaBrowser.Model/Session/TranscodingInfo.cs` (TranscodeReasons, Bitrate, HardwareAccelerationType, IsVideoDirect, IsAudioDirect, VideoCodec, AudioCodec, Width, Height), `MediaBrowser.Model/Entities/MediaStream.cs` (Index, Type, Language, VideoRange, VideoRangeType, IsExternal). Use FlexTime-style tolerant decoding (unknown/missing → zero value, never fail the whole /Sessions decode).

## Tasks
1. **Migration 011** (`backend/internal/migrations/sql/011_playback_capture.up.sql` + down): on both `jf_activity_watchdog` and `jf_playback_activity` add `"AudioStreamIndex" int`, `"AudioLanguage" text`, `"SubtitleStreamIndex" int`, `"SubtitleLanguage" text`, `"TranscodeReasons" jsonb`, `"Bitrate" bigint`, `"HwAccel" text`, `"VideoRange" text`, `"PauseCount" int NOT NULL DEFAULT 0`, `"IsRemote" boolean`. Model fields + upsert column lists in `repos.go`.
2. **Jellyfin types** (`backend/internal/jellyfin/types.go`): extend `PlayState` (AudioStreamIndex, SubtitleStreamIndex as *int), `TranscodingInfo` (TranscodeReasons []string — Jellyfin serialises it as an array of strings in 10.10+; tolerate a comma-separated string too —, Bitrate *int64, HardwareAccelerationType *string, Width/Height *int), `MediaStream` (Index, Language, VideoRange, VideoRangeType, IsExternal). Tests decoding real-looking payloads (both TranscodeReasons forms).
3. **Tracker capture** (`tracker.go` `refresh`/`newEntry`/`accumulate`/`finalize`): resolve audio/subtitle language from `NowPlayingItem.MediaStreams` by `Index` (subtitle index -1 → none); `VideoRange` from the video stream (prefer `VideoRangeType`); `IsRemote` from `RemoteEndPoint` (RFC1918, loopback, link-local, ULA fc00::/7 → local; empty → null); `PauseCount` increments on each transition not-paused → paused between polls; `TranscodeReasons`/`Bitrate`/`HwAccel` from `TranscodingInfo` (keep the last non-empty values); copy all to the playback row in `finalize`. Tests for each (including pause transitions and private/public IP classification).
4. **Geolocation cache**: table in the same migration `ip_geo_cache (ip text PRIMARY KEY, country text, city text, updated_at timestamptz)`; a background task (registered like the existing scheduled tasks, daily + after sync) that fills it for distinct remote IPs of the last 90 days using the existing `/utils/geolocateIp` implementation (`backend/internal/handler/utils.go` — extract the lookup into a reusable service function; rate-limit, skip private IPs, cache misses for 7 days). Tests with a fake geolocator.
5. **Backfill**: none required (new columns are null for old rows); document it in the migration comment.

## Commit plan
`feat: add playback capture columns`, `feat: decode more jellyfin session fields`, `feat: capture audio subtitle transcode and network fields`, `feat: add ip geolocation cache task`.
