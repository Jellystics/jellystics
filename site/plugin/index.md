# Jellystics plugin for Jellyfin

The Jellystics plugin runs inside Jellyfin. It records every playback from the server's own events, keeps the complete history in its own database, and relays the “My statistics” page to every Jellyfin user.

- **Precise**: real watch time, pauses, seeks and skipped time, audio and subtitle tracks and their changes, full transcoding details, “played to completion” from Jellyfin.
- **Complete**: library additions and removals, logins (including failed ones), played and favourite marks with their reason.
- **Durable**: everything is kept in `jellystics.db` with no time limit; a crash loses at most 15 seconds of the play in progress.
- **Light**: events are handled in memory and written by a single background writer, so Jellyfin is never slowed down.
- **Independent**: Jellystics works without it and switches back to its own tracker if the plugin stops answering.

It is based on the official Playback Reporting plugin (GPL-3), with a new identity: both plugins can be installed side by side, and the Jellystics plugin can import Playback Reporting's history.

## Requirements

- Jellyfin **10.11** (`targetAbi 10.11.0.0`, .NET 9).
- Jellystics reachable from Jellyfin over HTTP, if you want the “My statistics” page.

## Install

1. In Jellyfin, open **Dashboard → Plugins → Repositories** and add:

   ```txt
   https://raw.githubusercontent.com/Jellystics/jellystics-plugin-playbackreporting/master/manifest.json
   ```

2. Open **Catalog**, install **Jellystics**, then **restart Jellyfin**.
3. Check that *Jellystics* appears in **Dashboard → Plugins** and in the dashboard menu (server section).

Nothing else is needed on the Jellystics side: within 30 seconds, *Settings → Configuration → Jellystics plugin* turns **Active**, and Jellystics links the “My statistics” page automatically.

::: tip Manual install
Download `Jellyfin.Plugin.Jellystics-v<version>.zip` from the [releases](https://github.com/Jellystics/jellystics-plugin-playbackreporting/releases), unzip `Jellyfin.Plugin.Jellystics.dll` into `<jellyfin config>/plugins/Jellystics/` and restart Jellyfin.
:::

## First steps after installing

1. **Import the old history** (optional): if you used the official Playback Reporting plugin, run *Import history* in the [plugin settings](/plugin/configuration#import-history).
2. **Push Jellystics' own history** (optional): in Jellystics, *Settings → Configuration → Jellystics plugin → Save the history to the plugin*. The plugin then holds a full backup of everything.
3. **Check “My statistics”**: open the avatar menu in Jellyfin; *My statistics* is the second entry.

## Updating

Jellyfin shows updates from the repository in **Dashboard → Plugins**. Each release adds an entry to `manifest.json` with its checksum; install it and restart Jellyfin.

## Uninstalling

Remove the plugin from **Dashboard → Plugins** and restart Jellyfin. Jellystics detects that it is gone and keeps recording with its tracker. The plugin database (`<jellyfin config>/plugins/Jellyfin.Plugin.Jellystics/jellystics.db`) is left on disk.

## Learn more

- [Plugin settings](/plugin/configuration)
- [How playbacks are measured](/plugin/tracking)
- [Collected data](/plugin/data)
- [Sync with Jellystics](/plugin/sync)
- [“My statistics” page](/plugin/my-statistics)
- [Plugin API](/plugin/api)
