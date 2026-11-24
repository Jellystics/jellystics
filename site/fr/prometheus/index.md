# Prometheus et Grafana

Jellystics expose un endpoint Prometheus sur `/metrics`. Les valeurs sont calculées à partir de la base de données à chaque collecte (délai d'expiration de 30 s) : elles correspondent donc toujours au tableau de bord.

## 1. Créer une clé d'API {#_1-create-an-api-key}

Dans Jellystics : *Réglages → Clés API → Créer une clé d'API*, nommez-la `prometheus` et copiez-la (elle n'est affichée qu'une fois).

Les déploiements sans interface (`DISABLE_DASHBOARD=true`) définissent plutôt la clé avec `METRICS_API_KEY`.

::: warning Une clé d'API est toujours requise
Sans en-tête `Authorization: Bearer <key>` valide, `/metrics` répond `401`, même quand aucune clé n'existe encore.
:::

Testez-la :

```bash
curl -H "Authorization: Bearer <key>" http://jellystics:3000/metrics | head
```

## 2. Ajouter un job de collecte {#_2-add-a-scrape-job}

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

Ou avec la clé dans un fichier :

```yaml
    authorization:
      type: Bearer
      credentials_file: /etc/prometheus/jellystics.key
```

Un intervalle de 60 s suffit largement : la plupart des métriques sont des totaux depuis le début qui évoluent lentement, et chaque collecte exécute des requêtes SQL.

## 3. Importer le tableau de bord Grafana {#_3-import-the-grafana-dashboard}

Le dépôt [Jellystics-Grafana](https://github.com/Jellystics/Jellystics-Grafana) propose un tableau de bord de 31 panneaux : lectures, utilisateurs, bibliothèques, clients, habitudes horaires, taux de complétion et tendances. Dans Grafana : *Dashboards → New → Import*, téléversez le JSON et choisissez votre source de données Prometheus.

## Ce que sont les métriques {#what-the-metrics-are}

- La plupart des métriques sont des totaux **depuis le début** de **vues** (lectures d'au moins 30 s) ; `jellyfin_total_plays`, `jellyfin_total_duration_seconds` et `jellyfin_active_sessions` sont les exceptions. Utilisez `increase()` ou `delta()` pour obtenir des valeurs par période.
- Les heures et les jours de la semaine sont exprimés dans le fuseau horaire `TZ` de Jellystics.
- Les noms commencent par `jellyfin_`.
- Seules les métriques Jellystics sont exportées (pas de métriques du runtime Go ni du processus).

La liste complète, avec labels et exemples, se trouve dans la [référence des métriques](/fr/prometheus/metrics).

## Exemple d'alerte {#alerting-example}

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
