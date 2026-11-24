# Référence des métriques

Toutes les métriques reposent sur les mêmes définitions SQL que le tableau de bord (voir le [glossaire](/fr/guide/glossary)). « Vues » désigne les lectures d'au moins 30 secondes.

## Totaux {#totals}

| Métrique | Type | Labels | Description |
|---|---|---|---|
| `jellyfin_total_plays` | counter | – | Nombre de lectures, depuis le début. |
| `jellyfin_total_duration_seconds` | counter | – | Temps de lecture de toutes les lectures, en secondes. |
| `jellyfin_total_users` | gauge | – | Nombre d'utilisateurs. |
| `jellyfin_total_libraries` | gauge | – | Nombre de bibliothèques. |

## Par utilisateur, bibliothèque, type, client et méthode {#per-user-library-type-client-and-method}

| Métrique | Type | Labels | Description |
|---|---|---|---|
| `jellyfin_plays_by_user_total` | counter | `user` | Vues par utilisateur (top 500). |
| `jellyfin_hours_by_user_total` | counter | `user` | Heures de lecture par utilisateur (top 500). |
| `jellyfin_plays_by_library_total` | counter | `library` | Vues par bibliothèque (top 100, bibliothèques archivées exclues). |
| `jellyfin_plays_by_media_type_total` | counter | `type` | Vues par type : `Movie`, `Series` (épisodes), `Audio`. |
| `jellyfin_plays_by_client_total` | counter | `client` | Vues par application cliente (top 50). |
| `jellyfin_plays_by_playback_method_total` | counter | `method` | Vues par méthode de lecture (`DirectPlay`, `DirectStream`, `Transcode`). |

Quand deux utilisateurs ou bibliothèques portent le même nom, le second reçoit le suffixe ` (2)` afin que les séries temporelles restent distinctes.

## Habitudes horaires {#time-patterns}

| Métrique | Type | Labels | Description |
|---|---|---|---|
| `jellyfin_plays_by_hour` | gauge | `hour` (0–23) | Vues par heure de la journée, depuis le début. |
| `jellyfin_plays_by_day_of_week` | gauge | `day` (0 = dimanche … 6) | Vues par jour de la semaine, depuis le début. |
| `jellyfin_heatmap_plays` | gauge | `day`, `hour` | Vues par jour de la semaine × heure, pour une heatmap. |

## Contenus {#content}

| Métrique | Type | Labels | Description |
|---|---|---|---|
| `jellyfin_item_play_count` | gauge | `name`, `type` | Vues des 50 titres les plus vus (au niveau du film, de la série ou de l'album). |
| `jellyfin_items_by_type_total` | gauge | `type` | Contenus des bibliothèques : `Movie`, `Series`, `Episode`, `Audio`. |
| `jellyfin_unwatched_items_by_type` | gauge | `type` | Contenus jamais vus, par type. |

## Complétion et habitudes {#completion-and-habits}

| Métrique | Type | Labels | Description |
|---|---|---|---|
| `jellyfin_completed_plays_total` | counter | – | Unités de complétion terminées (≥ 90 %, ou « lu jusqu'à la fin » selon Jellyfin pour les lectures du plugin). Une unité = un utilisateur, un titre et un jour local. |
| `jellyfin_abandoned_plays_total` | counter | – | Unités de complétion sous 90 %. |
| `jellyfin_completion_rate_percent` | gauge | – | `completed ÷ (completed + abandoned) × 100`. |
| `jellyfin_binge_sessions_total` | counter | – | Triplets (utilisateur, série, jour) avec au moins 3 épisodes distincts vus. |

## En direct {#live}

| Métrique | Type | Labels | Description |
|---|---|---|---|
| `jellyfin_active_sessions` | gauge | – | Films, épisodes et pistes en cours de lecture à cet instant. |

## Exemple de sortie {#sample-output}

```txt
# HELP jellyfin_total_plays Total number of plays all time
# TYPE jellyfin_total_plays counter
jellyfin_total_plays 4507
# HELP jellyfin_plays_by_media_type_total Total plays per media type
# TYPE jellyfin_plays_by_media_type_total counter
jellyfin_plays_by_media_type_total{type="Audio"} 612
jellyfin_plays_by_media_type_total{type="Movie"} 980
jellyfin_plays_by_media_type_total{type="Series"} 1820
# HELP jellyfin_active_sessions Number of currently active playback sessions
# TYPE jellyfin_active_sessions gauge
jellyfin_active_sessions 2
```

## Exemples PromQL {#promql-examples}

```txt
# Vues sur les dernières 24 heures
increase(jellyfin_total_plays[24h])

# Heures regardées par jour, par utilisateur
increase(jellyfin_hours_by_user_total[1d])

# Top 5 des utilisateurs par vues
topk(5, jellyfin_plays_by_user_total)

# Part de transcodage (%)
100 * jellyfin_plays_by_playback_method_total{method="Transcode"}
  / ignoring(method) sum without(method) (jellyfin_plays_by_playback_method_total)

# Part du catalogue jamais regardée, par type
100 * jellyfin_unwatched_items_by_type / jellyfin_items_by_type_total

# Heure la plus chargée
topk(1, jellyfin_plays_by_hour)
```
