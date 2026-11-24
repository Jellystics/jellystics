# Sync with Jellystics

Jellystics **pulls** data from the plugin; the plugin never calls Jellystics, except to check the “My statistics” link and to relay that page.

## Detection

Every 30 seconds Jellystics calls `GET /Jellystics/Info` on the Jellyfin server with its Jellyfin API key.

- **404**: the plugin is not installed. Jellystics keeps using its tracker.
- **schemaVersion** different from the one Jellystics supports (1): the plugin is ignored and an error is shown in the plugin card.
- Otherwise the plugin is *detected* and the sync runs.

## Reading changes

For each stream (plays, library events, user data events, logins):

```http
GET /Jellystics/Plays?since=<cursor>&limit=500
```

```json
{ "items": [ … ], "next": 58231, "hasMore": true }
```

Jellystics stores `next` after each page and continues while `hasMore` is true (up to 1000 pages per run; the rest follows on the next run).

## Who records the plays

| Plugin state | Who writes plays |
|---|---|
| **Healthy** (successful sync less than 2 min ago) | The plugin. The tracker still watches sessions for the live page but does not write plays. |
| **Unhealthy** | The tracker. Its plays are kept unless a closed plugin play already covers them. |
| **Back to healthy** | The plugin. Its closed plays replace the tracker rows they cover. |

Duplicate removal:

- after each page, tracker plays covered by a new plugin play (same user, item and device, inside the plugin play with 10 s of slack) are deleted;
- after each run, the same check runs on the last 24 hours;
- when the plugin has been unhealthy for over an hour, its plays left open for over an hour are closed.

::: info Clocks
Matching compares start times between Jellyfin and Jellystics: keep both hosts' clocks synchronised (NTP).
:::

## Reset and resync

- If the plugin's `databaseId` changes or its cursor goes backwards (database deleted or restored), Jellystics resets its cursors and reads everything again. Upserts are idempotent: nothing is duplicated.
- **Resync from the plugin** (*Settings → Configuration*) does the same on demand. Restoring a Jellystics backup does it too.

## Pushing Jellystics' history to the plugin

*Save the history to the plugin* sends the plays Jellystics recorded itself (tracker, Playback Reporting imports, TSV imports, restored backups) to the plugin:

- movies, episodes and tracks with a user, an item and a start date;
- in order, in batches of **500** (`POST /Jellystics/Import/Plays`);
- the plugin inserts them with the source `import-jellystics` and skips ids it already has: running it again is safe;
- progress is saved after each batch: after an error or a restart, *Resume the save* continues where it stopped.

These plays come back to Jellystics through the normal sync and are matched to the existing rows.

## “My statistics” link

After each successful sync, Jellystics makes sure the plugin knows its address and secret (`POST /Jellystics/Link`):

- **address**: `JELLYSTICS_PUBLIC_URL`, or `http://<local IP Jellystics uses to reach Jellyfin>:<PORT>`;
- **secret**: 32 random bytes generated once and kept by Jellystics;
- it re-links when the plugin is not linked, when the address or the secret fingerprint differs, or when the plugin reports a rejected secret. Failures back off from 30 s up to 30 min.

See [“My statistics” page](/plugin/my-statistics).
