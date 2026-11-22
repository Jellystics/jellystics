# Jellystics

A modern and lightweight statistics tracker for your [Jellyfin](https://jellyfin.org) server.

<img width="3504" height="2092" alt="image" src="https://github.com/user-attachments/assets/52d0101b-545e-4da3-8781-95a42945f1fb" />

<img width="3504" height="2092" alt="image" src="https://github.com/user-attachments/assets/e54e2dd6-5dbe-4f93-93c5-6e3e5b610b1c" />

<img width="3504" height="2092" alt="image" src="https://github.com/user-attachments/assets/a6be3a2f-79e0-4eb4-946a-bbbf98af85c0" />

---

## Features

- **Accurate playback tracking** — the tracker polls Jellyfin `/Sessions` every 5 s and accumulates real watched time (pauses excluded)
- **Clicks vs views** — every play start is a click, a view needs at least 30 s watched; replays are detected
- **Playback quality details** — audio and subtitle language, transcode reasons, bitrate, hardware acceleration, video range, pause count
- **Local vs remote** — playback origin with an IP geolocation cache
- **Overview, history and calendar** — period picker with comparison, history table and calendar view
- **Per-user statistics and recaps** — plays, hours, genres, clients, completion rate, and a recap page per user and period
- **Library and media pages** — libraries, movies, series, episodes, artists and albums (with a track side panel)
- **Real-time sessions** — live view of active playback sessions
- **Insights and discovery** — time patterns, heatmaps, binge detection (3+ episodes in a day), unwatched content
- **Webhooks** — Discord and HTTP notifications for playback events
- **Backup & restore** — one-click database backups with restore support
- **Prometheus metrics** — full `/metrics` endpoint for Grafana integration
- **Import** — import history from the Jellyfin Playback Reporting plugin
- **Jellyfin-backed login** — sessions follow the Jellyfin token lifetime, are revalidated against Jellyfin, and logout revokes them

---

## Quick Start

**1. Copy the compose file**

```yaml
services:
  jellystics-db:
    image: postgres:18.1
    shm_size: '1gb'
    container_name: jellystics-db
    restart: unless-stopped
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: mypassword
      POSTGRES_DB: jellystics
    volumes:
      - postgres-data:/var/lib/postgresql
    healthcheck:
      test: ["CMD-SHELL", "pg_isready --dbname=postgres --username=postgres"]
      interval: 10s
      timeout: 5s
      retries: 5

  jellystics:
    image: ghcr.io/jellystics/jellystics:latest
    container_name: jellystics
    restart: unless-stopped
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: mypassword
      POSTGRES_IP: jellystics-db
      POSTGRES_PORT: 5432
      JWT_SECRET: "change-me-to-a-random-secret"
      TZ: Europe/Paris
    volumes:
      - jellystics-backup-data:/app/data
    ports:
      - "3000:3000"
    depends_on:
      jellystics-db:
        condition: service_healthy

volumes:
  postgres-data:
  jellystics-backup-data:
```

**2. Start**

```bash
docker compose up -d
```

**3. Open** `http://localhost:3000` and follow the setup wizard to connect your Jellyfin server.

---

## Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `JWT_SECRET` | **Yes** | — | Secret key for session tokens. Use a long random string. |
| `POSTGRES_IP` | No | `localhost` | PostgreSQL host |
| `POSTGRES_PORT` | No | `5432` | PostgreSQL port |
| `POSTGRES_USER` | No | `postgres` | PostgreSQL user |
| `POSTGRES_PASSWORD` | No | — | PostgreSQL password |
| `POSTGRES_DB` | No | `jellystics` | PostgreSQL database name |
| `DATABASE_URL` | No | — | Full connection string (overrides individual `POSTGRES_*` vars) |
| `PORT` | No | `3000` | HTTP port Jellystics listens on |
| `TZ` | No | UTC | Timezone (e.g. `Europe/Paris`) |
| `DISABLE_DASHBOARD` | No | `false` | Set to `true` to serve only the API (no frontend) |
| `METRICS_API_KEY` | No | — | Pre-seeds an API key at startup. Required when `DISABLE_DASHBOARD=true` to authenticate `/metrics`. |
| `JF_HOST` | No | — | Jellyfin server URL (e.g. `http://jellyfin:8096`). Applied at startup only when no Jellyfin server is configured yet; a server set from the setup wizard or the settings is never overwritten. |
| `JF_API_KEY` | No | — | Jellyfin API key, used with `JF_HOST` under the same rule (only filled when no key is stored). |

### Docker Secrets

All variables support the `FILE__` prefix to load values from a file (useful with Docker Secrets):

```yaml
environment:
  FILE__JWT_SECRET: /run/secrets/jwt_secret
  FILE__POSTGRES_PASSWORD: /run/secrets/db_password
```

---

## Updating

```bash
docker compose pull
docker compose up -d
```

Migrations run automatically on startup — no manual steps required.

---

## Grafana

Jellystics exposes a Prometheus-compatible `/metrics` endpoint secured by API key.

**1. Create an API key** — Settings → API Keys in the Jellystics dashboard.

**2. Add a scrape config** to Prometheus:

```yaml
scrape_configs:
  - job_name: jellystics
    static_configs:
      - targets: ['your-jellystics-host:3000']
    authorization:
      credentials: YOUR_API_KEY
    scrape_interval: 60s
```

**3. Import the dashboard** from [Jellystics-Grafana](https://github.com/Jellystics/Jellystics-Grafana) — 31 panels covering plays, users, libraries, clients, time patterns, completion rate, and trends over time.

---

## Reverse Proxy

Jellystics runs on port `3000`. Example Nginx config:

```nginx
location / {
    proxy_pass http://localhost:3000;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
}
```

The `Upgrade` / `Connection` headers are required for WebSocket support (live sessions page).

---

## Stack

- **Backend** — Go, Gin, GORM, PostgreSQL
- **Frontend** — React 19, TypeScript, Vite, React Router 7, TanStack Query and Table, Radix primitives, CSS Modules with design tokens, custom SVG charts (no MUI)
- **Metrics** — Prometheus client

The visual design reference lives in [`design/jellystics-design.html`](design/jellystics-design.html) — open it directly in a browser.

---

## Pages

| Route | Page |
|---|---|
| `/` | Overview |
| `/live` | Live sessions |
| `/history` | History (table and calendar view) |
| `/libraries`, `/libraries/:id` | Libraries |
| `/movies/:id`, `/series/:id`, `/series/:id/episodes/:episodeId` | Movie, series and episode details |
| `/artists/:id`, `/albums/:id` | Artist and album details (`?track=` opens the track side panel) |
| `/users`, `/users/:id`, `/users/:id/recap/:period` | Users and per-user recap |
| `/insights` | Insights |
| `/discover` | Discover |
| `/playback` | Playback quality |
| `/settings/:section` | Settings |

---

## API v2

Statistics are served under `/api/v2/*`. Endpoints accept `from` / `to` (local dates, `to` exclusive), `compare`, `userId`, `libraryId` and `type`.

---

## Development

Requirements: Go, Node with pnpm, Docker (for PostgreSQL) and [Task](https://taskfile.dev).

```bash
task setup      # create backend/.env from backend/.env.example
task install    # install frontend dependencies
task db         # start PostgreSQL (docker-compose maps 5432)
task backend    # Go API on :3000
task frontend   # Vite dev server on :3001, proxying to the backend on :3000
```

The dev database can run on any port: set `POSTGRES_PORT` in `backend/.env`.

Checks:

```bash
cd frontend
pnpm test        # unit tests (Vitest)
pnpm test:e2e    # end-to-end tests (Playwright)
pnpm lint
pnpm build

cd backend
go test ./...    # needs a running PostgreSQL (see POSTGRES_PORT)
```
