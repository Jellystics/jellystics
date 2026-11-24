# Glossary

Definitions used everywhere in Jellystics (dashboard, “My statistics” and Prometheus). Days, hours and weekdays are local to the `TZ` time zone.

| Term | Definition |
|---|---|
| **Click** | A play that was started, whatever its length. A click that never reached 30 s shows a *Click* badge. |
| **View** | A play with at least **30 seconds** really watched (pauses excluded). |
| **Watch time** | Sum of the watched time of all plays, clicks included. |
| **Active user** | A user with at least one play (click) in the period. |
| **Completion** | `max(stop position, watched time) ÷ runtime`, capped at 100 %. Counted once per user, title and day: watching an episode in two sittings on the same day is one unit. *Average completion* is the mean of these units. |
| **Finished view** | Completion ≥ **90 %**. For movies and episodes of 5 min or more recorded by the plugin, Jellyfin's own “played to completion” flag decides instead. Plays still in progress never count. |
| **Abandoned** (completion) | A completion unit under 90 % (Prometheus `jellyfin_abandoned_plays_total`, “drop-offs” on episodes). |
| **Abandoned** (Discover) | A title last played more than **60 days** ago and left between **10 % and 80 %** (movies), or with 10–80 % of its episodes viewed (series). |
| **Reached half** | Funnel stage: completion ≥ 50 %. |
| **False start** | A click that never became a view (stopped before 30 s). |
| **Skip rate** (music) | Clicks that never became a listen ÷ clicks. Shown in red above 30 %. |
| **Binge** | At least **3 distinct episodes** of the same series by the same user on the same day. |
| **Resumed** | A view that started **30 s or more** into the title. |
| **Rewatch** | The same title viewed by the same person on two or more different days (tracks: two or more views). |
| **Retention** (series) | Viewers of the last regular episode ÷ viewers of the first one (specials excluded). |
| **Retention** (users) | Weekly cohorts: users active in a week who are active again 1 to 4 weeks later (last 8 full weeks). |
| **Streak** | Consecutive days with at least one play. |
| **Concurrent streams** | Maximum number of simultaneous plays, in 15-minute slots. |
| **Time to watch** | Days between the date an item was added and its first view. |
| **Short skip at the start** (intro) | Plugin only: a forward skip of 10 s to 15 % of the runtime (at most 3 min) in a play started from the beginning. An estimate. |
| **Play method** | *Direct play* (file sent as is), *Direct stream* (remuxed, codecs unchanged), *Transcode* (re-encoded). |
| **Local / remote** | From the client IP: private (RFC 1918, ULA), loopback and link-local addresses are local. CGNAT (100.64.0.0/10, e.g. Tailscale) counts as remote. |
| **Original or dubbed** | *Original* = the file's first audio track was played. |
| **Outdated device** | An app version older than the newest version seen for the same client. |
| **Profile scores** | Regularity (active days), Discovery (first views), Concentration (share of the top 3 titles). |
