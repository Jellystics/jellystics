# Installation

Jellystics runs as a Docker container next to a PostgreSQL database.

## Docker Compose

Create a folder and a `docker-compose.yaml` file:

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

Then:

```bash
docker compose up -d
```

and open `http://<host>:3000`. The [first start](#first-start) section below walks you through the wizard.

::: warning Change the secrets
Replace `mypassword` and `JWT_SECRET`. `JWT_SECRET` signs the dashboard sessions: use a long random string, for example `openssl rand -hex 32`. Jellystics refuses to start without it.
:::

::: tip Time zone
Set `TZ` to your time zone. Every “day”, “hour” and “weekday” in the statistics is computed in this time zone, and scheduled tasks follow it. Without it, Jellystics uses UTC and logs a warning.
:::

The `/app/data` volume holds the backups (`/app/data/backups`) and the uploaded files to restore (`/app/data/imports`).

## Environment variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `JWT_SECRET` | **yes** | – | Secret that signs dashboard sessions. |
| `POSTGRES_IP` | no | `localhost` | PostgreSQL host. |
| `POSTGRES_PORT` | no | `5432` | PostgreSQL port. |
| `POSTGRES_USER` | no | `postgres` | PostgreSQL user. |
| `POSTGRES_PASSWORD` | no | empty | PostgreSQL password. |
| `POSTGRES_DB` | no | `jellystics` | Database name. |
| `DATABASE_URL` | no | – | Full PostgreSQL connection string. Overrides every `POSTGRES_*` variable. |
| `PORT` | no | `3000` | HTTP port. |
| `TZ` | no | UTC | Time zone of the statistics and of the scheduler, e.g. `Europe/Paris`. |
| `JF_HOST` | no | – | Jellyfin URL, e.g. `http://jellyfin:8096`. Only used when no server is configured yet: a server saved in the wizard or in the settings always wins. |
| `JF_API_KEY` | no | – | Jellyfin API key, same rule as `JF_HOST` (only filled when no key is stored and the stored host is `JF_HOST`). |
| `JELLYSTICS_PUBLIC_URL` | no | auto-detected | Jellystics URL **as seen from the Jellyfin server**. Sent to the plugin for the “My statistics” page. Auto-detected as `http://<local IP used to reach Jellyfin>:<PORT>`. |
| `DISABLE_DASHBOARD` | no | `false` | `true` serves only the API and `/metrics` (no dashboard, no “My statistics”). |
| `METRICS_API_KEY` | no | – | Adds this value as an API key named `metrics` at every start. Needed with `DISABLE_DASHBOARD=true`, since there is no interface to create keys. |
| `JS_GEOLITE_ACCOUNT_ID` | no | – | MaxMind GeoLite account id, for IP geolocation of remote plays. |
| `JS_GEOLITE_LICENSE_KEY` | no | – | MaxMind GeoLite license key. Without both, plays are classified local/remote but not geolocated. |

### Docker secrets

In the Docker image, every variable can be read from a file with the `FILE__` prefix:

```yaml
environment:
  FILE__JWT_SECRET: /run/secrets/jwt_secret
  FILE__POSTGRES_PASSWORD: /run/secrets/db_password
```

A missing file stops the container with an error. If both `X` and `FILE__X` are set, the file wins (a warning is logged).

### Metrics-only mode

To run Jellystics only as a Prometheus exporter:

```yaml
environment:
  DISABLE_DASHBOARD: "true"
  METRICS_API_KEY: "a-long-random-key"
  JF_HOST: http://jellyfin:8096
  JF_API_KEY: your-jellyfin-api-key
```

## First start

1. Open Jellystics. The **Connect Jellyfin** step asks for the Jellyfin URL (for example `http://192.168.1.10:8096`) and an API key (Jellyfin: *Dashboard → API Keys → +*).
2. The **Sign in** step asks for a Jellyfin username and password. Only **Jellyfin administrators** can use the dashboard; other accounts get “Administrator access required”. Regular users get their own [“My statistics”](/plugin/my-statistics) page inside Jellyfin instead.
3. Jellystics runs a first full sync of libraries and users. Plays start being recorded immediately.

::: tip Schedule the full sync once
On a brand-new installation, open *Settings → Tasks* and save the schedule of **Full Jellyfin Sync** once (the default cron `0 0 * * *` is fine). Until a task setting has been saved, the nightly full sync is not scheduled.
:::

How sign-in works:

- Jellystics signs you in against Jellyfin, then keeps a session tied to your Jellyfin token.
- Every 5 minutes the session is checked against Jellyfin: if the token is revoked or you are no longer an administrator, the session ends.
- Sessions unused for 30 days are revoked. *Log out* also revokes the Jellyfin token.

## Reverse proxy

Jellystics listens on port `3000`. The live page uses a WebSocket (Socket.IO), so forward the `Upgrade` and `Connection` headers. Nginx:

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

Caddy forwards WebSockets on its own:

```txt
stats.example.com {
    reverse_proxy localhost:3000
}
```

## Updating

```bash
docker compose pull
docker compose up -d
```

Database migrations run automatically on start-up. *Settings → About* tells you when a new version is available.

## Next

- [Configure Jellystics](/guide/configuration)
- [Install the plugin](/plugin/)
