# Prometheus y Grafana

Jellystics expone un endpoint de Prometheus en `/metrics`. Los valores se calculan a partir de la base de datos en cada recogida (tiempo de espera de 30 s), así que siempre coinciden con el panel.

## 1. Crear una clave API {#_1-create-an-api-key}

En Jellystics: *Ajustes → Claves API → Crear una clave API*, llámala `prometheus` y cópiala (solo se muestra una vez).

Los despliegues sin interfaz (`DISABLE_DASHBOARD=true`) definen la clave con `METRICS_API_KEY`.

::: warning Siempre se necesita una clave API
Sin una cabecera `Authorization: Bearer <key>` válida, `/metrics` responde `401`, aunque todavía no exista ninguna clave.
:::

Pruébala:

```bash
curl -H "Authorization: Bearer <key>" http://jellystics:3000/metrics | head
```

## 2. Añadir un job de recogida {#_2-add-a-scrape-job}

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

O con la clave en un archivo:

```yaml
    authorization:
      type: Bearer
      credentials_file: /etc/prometheus/jellystics.key
```

Un intervalo de 60 s es más que suficiente: la mayoría de las métricas son totales históricos que cambian despacio, y cada recogida ejecuta consultas SQL.

## 3. Importar el panel de Grafana {#_3-import-the-grafana-dashboard}

El repositorio [Jellystics-Grafana](https://github.com/Jellystics/Jellystics-Grafana) contiene un panel de 31 paneles: reproducciones, usuarios, bibliotecas, clientes, patrones horarios, tasa de finalización y tendencias. En Grafana: *Dashboards → New → Import*, sube el JSON y elige tu fuente de datos de Prometheus.

## Qué son las métricas {#what-the-metrics-are}

- La mayoría de las métricas son totales **históricos** de **vistas** (reproducciones de al menos 30 s); `jellyfin_total_plays`, `jellyfin_total_duration_seconds` y `jellyfin_active_sessions` son las excepciones. Usa `increase()` o `delta()` para obtener valores por periodo.
- Las horas y los días de la semana están en la zona horaria `TZ` de Jellystics.
- Los nombres empiezan por `jellyfin_`.
- Solo se exportan las métricas de Jellystics (ni métricas del runtime de Go ni del proceso).

La lista completa con etiquetas y ejemplos está en la [referencia de métricas](/es/prometheus/metrics).

## Ejemplo de alertas {#alerting-example}

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
