# “My statistics” page

Every Jellyfin user gets a **My statistics** entry in the avatar menu (second, after *Profile*). It opens `/Jellystics/Me/` on the Jellyfin server and shows only that user's statistics.

![My statistics](/screenshots/en/me.jpg){.shot}

## What users see

- Watch time, views, clicks, finished views, average completion and current streak, compared with the previous period.
- Period: 7 days, 30 days, 90 days, 1 year or all time.
- Watch time, views or clicks per day, top content, split by media type and play method, a weekday × hour heatmap, every play as a dot, devices and apps, and their full history.

The page follows the user's Jellyfin language (English, French or Spanish) and theme. If Jellystics has not synced the account yet, it shows “Your stats are on the way”.

## How it works

```txt
Browser ──► Jellyfin /Jellystics/Me/…  ──► Jellystics /me/…        (page and assets)
Browser ──► Jellyfin /Jellystics/Me/api/… ──► Jellystics /api/me/… (statistics, with the secret)
```

1. The plugin adds a small script (`/Jellystics/Me/client.js`) to jellyfin-web's `index.html`. The script adds the menu entry, with a label in the page language.
2. The browser **only talks to Jellyfin**. Statistics calls need a Jellyfin login.
3. The plugin forwards them to Jellystics with two headers: `X-Jellystics-Secret` (the shared secret) and `X-Jellystics-User` (the user id taken from the Jellyfin authentication, never from the browser). A user can therefore only read their own data.
4. Jellystics answers only to the right secret and to a known user.

Users never need a Jellystics account, and Jellystics does not need to be exposed to the Internet: only the Jellyfin server must reach it.

## Requirements

- The plugin is linked: *Dashboard → Plugins → Jellystics* says “Linked … reachable”.
- Jellyfin can reach Jellystics. If the auto-detected address is wrong, set `JELLYSTICS_PUBLIC_URL` in Jellystics or the **URL override** in the plugin settings.
- The dashboard is enabled (`DISABLE_DASHBOARD` is not `true`).

## Troubleshooting

| The page says | Cause | Fix |
|---|---|---|
| Statistics unavailable (not linked) | Jellystics has not linked itself yet. | Check the plugin card in Jellystics (*Active*), wait 30 s. |
| Statistics unavailable (unreachable) | Jellyfin cannot reach the Jellystics URL. | Set the URL override or `JELLYSTICS_PUBLIC_URL`. |
| Statistics unavailable (reconnecting) | The secret was rejected; Jellystics re-links within a minute. | Make sure only one Jellystics instance targets this Jellyfin. |
| Sign in to Jellyfin | No Jellyfin session in this browser. | Sign in to Jellyfin first. |
| The menu entry is missing | The browser cached the old `index.html`. | Reload without cache; check the plugin is enabled. |

## Security

The page and its scripts come from Jellystics but run on the Jellyfin origin, where they could use the signed-in user's Jellyfin session. Jellyfin therefore trusts the Jellystics instance it is linked to as much as its own web client. Run Jellystics on a trusted LAN or Docker network, or give it an `https` URL.

Protections in the plugin:

- only the page, `assets/**`, `locales/**/*.json` and root images are relayed, each with an allowed content type;
- strict Content-Security-Policy on the page (`script-src 'self'`, `connect-src 'self'`, `frame-ancestors 'self'`), `X-Content-Type-Options: nosniff`, `Referrer-Policy: same-origin`;
- no redirects followed, 15 s time-out, 10 MB size limit;
- statistics are never cached (`Cache-Control: private, no-store`);
- query parameters such as `userId` or `api_key` are dropped before forwarding;
- the secret lives only in the plugin XML and is never returned by an API.
