# Development

## Plugin

Repository: [Jellystics/jellystics-plugin-playbackreporting](https://github.com/Jellystics/jellystics-plugin-playbackreporting).

Use the Nix dev shell (`nix develop`, or direnv): .NET 9, Node, go-task, Docker and SQLite.

```bash
task test    # xUnit tests, then node --test for client.js
task build   # build, copy the dll to config/plugins/Jellystics/, restart the dev Jellyfin
task logs    # follow the dev Jellyfin logs
```

The dev Jellyfin (`docker compose up -d`) runs on http://localhost:8097 (container `jellystics-plugin-jellyfin`). `config/` and `media/` are git-ignored.

Manual end-to-end check:

1. `docker compose up -d`, `task build`, open http://localhost:8097, create the admin and add `/media` as a library.
2. `docker logs jellystics-plugin-jellyfin 2>&1 | grep -i jellystics`: the plugin must load.
3. Play a clip 60 s, pause 30 s, seek forward 2 min, play 30 s, stop, then:

   ```bash
   sqlite3 config/plugins/Jellyfin.Plugin.Jellystics/jellystics.db \
     "select WatchedMs, PausedMs, PauseCount, SeekCount, SkippedMs, State from Plays"
   ```

   Expect about `90000 | 30000 | 1 | 1 | ~120000 | closed`. On macOS, read the database after `docker stop`, or copy `jellystics.db*` first.

### Releases

Push a four-part tag, for example `v0.2.2.0`. The release workflow runs the tests, builds the dll, zips it, adds the version to `manifest.json` on `master` and publishes a GitHub release.

## Jellystics

Repository: [Jellystics/jellystics](https://github.com/Jellystics/jellystics). Requires Go, Node with pnpm, Docker and Task.

```bash
task setup      # create backend/.env
task install    # frontend dependencies
task db         # PostgreSQL in Docker
task backend    # Go API on :3000
task frontend   # Vite on :3001, proxied to :3000
```

Checks: `pnpm test`, `pnpm test:e2e`, `pnpm lint`, `pnpm build` in `frontend/`, and `go test ./...` in `backend/` (needs PostgreSQL).

## This website

The site lives in `site/` of the Jellystics repository and is built with [VitePress](https://vitepress.dev).

```bash
cd site
pnpm install
pnpm docs:dev       # http://localhost:5173/jellystics/
pnpm docs:build     # output in site/.vitepress/dist
pnpm docs:preview   # serve the build on http://localhost:4173/jellystics/
```

- English pages are at the root of `site/`, French in `site/fr/`, Spanish in `site/es/`, with the same file names.
- Screenshots are in `site/public/screenshots/<lang>/`. They are taken from the Jellystics frontend with its e2e API mocks, so they never contain real data.
- Pushing to `main` deploys the site with GitHub Actions (`.github/workflows/pages.yml`).
