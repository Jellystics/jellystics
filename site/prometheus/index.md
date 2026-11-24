# Prometheus and Grafana

Jellystics exposes a Prometheus endpoint at `/metrics`. Values are computed from the database at each scrape (30 s time-out), so they always match the dashboard.

## 1. Create an API key

In Jellystics: *Settings → API keys → Create an API key*, name it `prometheus` and copy it (it is shown once).

Headless deployments (`DISABLE_DASHBOARD=true`) set the key with `METRICS_API_KEY` instead.

::: warning An API key is always required
Without a valid `Authorization: Bearer <key>` header, `/metrics` answers `401`, even when no key exists yet.
:::

Test it:

```bash
curl -H "Authorization: Bearer <key>" http://jellystics:3000/metrics | head
```

## 2. Add a scrape job

```yaml
scrape_configs:
  - job_name: jellystics
    scrape_interval: 60s
    static_configs:
      - targets: ['jellystics:3000']
    authorization:
      type: Bearer
      credentials: <key>
```

Or with the key in a file:

```yaml
    authorization:
      type: Bearer
      credentials_file: /etc/prometheus/jellystics.key
```

A 60 s interval is plenty: most metrics are all-time totals that change slowly, and every scrape runs SQL queries.

## 3. Import the Grafana dashboard

The [Jellystics-Grafana](https://github.com/Jellystics/Jellystics-Grafana) repository has a 31-panel dashboard: plays, users, libraries, clients, time patterns, completion rate and trends. In Grafana: *Dashboards → New → Import*, upload the JSON and pick your Prometheus data source.

## What the metrics are

- Most metrics are **all-time** totals of **views** (plays of at least 30 s); `jellyfin_total_plays`, `jellyfin_total_duration_seconds` and `jellyfin_active_sessions` are the exceptions. Use `increase()` or `delta()` to get values per period.
- Hours and weekdays are in the `TZ` time zone of Jellystics.
- Names start with `jellyfin_`.
- Only the Jellystics metrics are exported (no Go runtime or process metrics).

The full list with labels and examples is in the [metrics reference](/prometheus/metrics).

## Alerting example

```yaml
groups:
  - name: jellystics
    rules:
      - alert: JellysticsDown
        expr: up{job="jellystics"} == 0
        for: 5m
      - alert: ManyStreams
        expr: jellyfin_active_sessions > 5
        for: 10m
        annotations:
          summary: "More than 5 streams for 10 minutes"
```
