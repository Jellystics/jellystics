# Plugin settings

Open **Jellyfin → Dashboard → Plugins → Jellystics** (or *Jellystics* in the server section of the dashboard menu). The page has four blocks.

## Status

Read from `GET /Jellystics/Info`:

```txt
Version 0.2.1.0
1 840 plays, 3 512 library events, 412 marks, 96 logins
First play 2024-03-02
Sync cursor 58 231
```

The cursor is the global version number Jellystics uses to read what changed (see [Sync](/plugin/sync)).

## My statistics page

Every user gets a *My statistics* entry in their avatar menu. Jellystics links itself automatically on its next sync: it sends its URL and a shared secret to the plugin. This block shows the result:

| Message | Meaning | What to do |
|---|---|---|
| Not linked | Jellystics has not pushed its address yet. | Wait for the next sync (30 s). Check that the plugin card is *Active* in Jellystics. |
| Linked to `<url>` · reachable | All good. | – |
| UNREACHABLE from Jellyfin | Jellyfin cannot reach that address. | Set the **URL override** to an address of Jellystics that the Jellyfin server can reach, or set `JELLYSTICS_PUBLIC_URL` in Jellystics. |
| Jellystics rejected the shared secret | The secret changed. | Jellystics re-links within a minute. If it persists, make sure only one Jellystics instance targets this Jellyfin. |

**Jellystics URL override**: leave empty to use the pushed URL. Set it when Jellystics is reachable from Jellyfin at another address, for example `http://192.168.1.10:3000` or `http://jellystics:3000` on a shared Docker network. It must be an absolute `http` or `https` URL without query, fragment or credentials.

The secret is stored only in the plugin XML configuration; it is never returned by any API (only a SHA-256 fingerprint is).

## Import history

Imports the history of the official **Playback Reporting** plugin.

- **Path of playback_reporting.db**: leave empty to use `<jellyfin data>/playback_reporting.db`, or point to a copy.
- **Import history** saves the path, then imports in the background. The status line shows `read / imported / skipped / repaired`.

What the import does:

- opens the file **read-only**, 500 rows at a time;
- gives each row the id `pbr-<rowid>`: running the import again never duplicates anything;
- sets the start to the logged time − 20 s (Playback Reporting logs the end of the first 20 seconds), read in the server time zone;
- repairs impossible durations: negative → 0; longer than 1.5 × the runtime → the runtime; longer than 12 h when the runtime is unknown → 12 h. Repaired rows are flagged;
- skips rows that overlap a play the plugin recorded itself;
- fills series, season, album, runtime and library from the current Jellyfin library when the item still exists.

## Purge plays before

Permanently deletes the **closed** plays that started before the chosen date (midnight in your browser's time zone).

::: warning
The purge is local to the plugin. It is **not** propagated to Jellystics, and importing Playback Reporting again restores the purged rows that came from it.
:::
