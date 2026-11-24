# Configuration

Jellystics is configured in two places: **environment variables** at start-up (see [Installation](/guide/installation#environment-variables)) and **Settings** in the dashboard (user menu → *Settings*). This page lists every setting; the [Settings tour](/platform/settings) shows the screens.

## Jellyfin server

*Settings → Configuration → Jellyfin server*

| Field | Description |
|---|---|
| Jellyfin URL | Must be `http://` or `https://`. This is the address Jellystics uses to read libraries, users and sessions. |
| API key | Leave empty to keep the stored key. Required to test the connection. Create one in Jellyfin: *Dashboard → API Keys*. |
| Public Jellystics URL | Public address of Jellystics. Used as the default bot avatar (`<url>/logo.png`) in Discord notifications. It is **not** the address sent to the plugin (see `JELLYSTICS_PUBLIC_URL`). |

*Test connection* checks the URL and key and answers “Connected to *server name*”.

## Jellystics plugin card

*Settings → Configuration → Jellystics plugin* shows the state of the [plugin](/plugin/):

- **Active**: the plugin answered less than 2 minutes ago. **Not syncing**: no successful sync for more than 2 minutes (the built-in tracker records plays meanwhile). **Not detected**: install it with the manifest URL shown in the card.
- Version, API schema, last sync, plays synced, last error.
- **“My statistics” page**: link state (Active, Jellystics unreachable from Jellyfin, Reconnecting, Not linked yet) and the address sent to the plugin.
- **Save the history to the plugin**: sends the plays Jellystics recorded itself (tracker, imports, restores) to the plugin, so the plugin holds the complete history. Safe to run again; resumable after an error or a restart.
- **Resync from the plugin**: re-reads everything from the plugin. Plays recorded twice are replaced by the plugin version.

Details: [Sync with Jellystics](/plugin/sync).

## Data quality card

Shows where the plays of a period come from (plugin, tracker, Playback Reporting imports, other imports, restored backups) and which details each source captured (stop position, audio/subtitle track, playback details, transcode details). It also counts imported durations that were repaired and plays without a usable date (excluded from every statistic).

## Scheduled tasks

*Settings → Tasks*. Cron expressions have 5 fields (`minute hour day month weekday`) and follow `TZ`.

| Task | Default schedule | What it does |
|---|---|---|
| Full Jellyfin Sync | `0 0 * * *` (always on) | Syncs every library, item, season, episode, track and user. Also triggers IP geolocation. |
| Recently Added Sync | `0 * * * *` (off by default) | Syncs the 100 latest items of each library. |
| Backup | `0 3 * * *` (off by default) | Creates a backup in `/app/data/backups`. |
| Jellyfin Playback Reporting Plugin Sync | `0 4 * * *` (off by default) | Imports the official Playback Reporting plugin history. Skipped when the Jellystics plugin is detected (the plugin imports it itself). |

Background jobs that are not listed: the plugin sync (every 30 s), the tracker (every 5 s), session clean-up (daily) and geolocation (2 min after start, then daily).

::: warning
Only one task runs at a time: a task started while another is running is skipped. On a new installation, save the Full Jellyfin Sync schedule once so it gets scheduled.
:::

The same tab imports a **Playback Reporting TSV/TXT export** (the file exported by the official plugin): each line becomes a play, negative durations are repaired, and plays already recorded by another source are dropped.

## Backups

*Settings → Backup*

- **Create a backup** writes `backup_YYYY-MM-DD HH-mm-ss.json` to `/app/data/backups`. The 5 most recent backups are kept.
- A backup contains: libraries, items, seasons, episodes, users, all plays, and Playback Reporting rows not imported yet. It does **not** contain the settings, API keys, webhooks or music tracks: those are recreated by a sync or must be set again.
- **Restore** upserts the backup into the database (existing plays are updated, plays already recorded by another source are skipped), then resets the plugin cursors. Jellystat backups are accepted too.
- **Upload a backup** accepts `.json` files.

## Webhooks

*Settings → Webhooks* sends **Discord** notifications.

| Field | Description |
|---|---|
| Name | Free text. |
| Discord webhook URL | `https://discord.com/api/webhooks/…` (Discord: *channel settings → Integrations → Webhooks*). |
| Bot username | Default `jellystics_bot`. |
| Bot avatar URL | Optional; defaults to the public Jellystics URL logo. |
| Notifications | At least one of: task started, task completed, task failed, backup completed, API key created, API key deleted. |

Each webhook can be enabled/disabled, edited, deleted or tested. Delivery has a 10 s timeout and no retry.

## API keys

*Settings → API keys*. A key is shown in full only once, at creation. Keys authenticate [`/metrics`](/prometheus/) with `Authorization: Bearer <key>`. Creating or deleting a key can notify a webhook.

## Logs

*Settings → Logs* lists the last 50 task runs (level, date, task, message) with search and a level filter. *Settings → Tasks* also streams the live output of the running task.

## About

Installed and latest version, link to the release, “Check for updates”.

## Dashboard language and theme

- **Language**: user menu → *Language* (English, Français, Español). Stored in the browser (`i18nextLng`); `?lng=fr` in the URL also works.
- **Theme**: user menu → *Light theme / Dark theme*, or `⌘K` → *Toggle theme*. Without a choice it follows the system.
