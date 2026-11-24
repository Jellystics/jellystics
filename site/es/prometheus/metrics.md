# Referencia de métricas

Todas las métricas proceden de las mismas definiciones SQL que el panel (consulta el [glosario](/es/guide/glossary)). «Vistas» significa reproducciones de al menos 30 segundos.

## Totales {#totals}

| Métrica | Tipo | Etiquetas | Descripción |
|---|---|---|---|
| `jellyfin_total_plays` | counter | – | Número de reproducciones, desde el principio. |
| `jellyfin_total_duration_seconds` | counter | – | Tiempo de reproducción de todas las reproducciones, en segundos. |
| `jellyfin_total_users` | gauge | – | Número de usuarios. |
| `jellyfin_total_libraries` | gauge | – | Número de bibliotecas. |

## Por usuario, biblioteca, tipo, cliente y método {#per-user-library-type-client-and-method}

| Métrica | Tipo | Etiquetas | Descripción |
|---|---|---|---|
| `jellyfin_plays_by_user_total` | counter | `user` | Vistas por usuario (top 500). |
| `jellyfin_hours_by_user_total` | counter | `user` | Horas de reproducción por usuario (top 500). |
| `jellyfin_plays_by_library_total` | counter | `library` | Vistas por biblioteca (top 100, bibliotecas archivadas excluidas). |
| `jellyfin_plays_by_media_type_total` | counter | `type` | Vistas por tipo: `Movie`, `Series` (episodios), `Audio`. |
| `jellyfin_plays_by_client_total` | counter | `client` | Vistas por aplicación cliente (top 50). |
| `jellyfin_plays_by_playback_method_total` | counter | `method` | Vistas por método de reproducción (`DirectPlay`, `DirectStream`, `Transcode`). |

Cuando dos usuarios o bibliotecas comparten nombre, el segundo recibe el sufijo ` (2)` para que las series sigan siendo distintas.

## Patrones horarios {#time-patterns}

| Métrica | Tipo | Etiquetas | Descripción |
|---|---|---|---|
| `jellyfin_plays_by_hour` | gauge | `hour` (0–23) | Vistas por hora del día, desde el principio. |
| `jellyfin_plays_by_day_of_week` | gauge | `day` (0 = domingo … 6) | Vistas por día de la semana, desde el principio. |
| `jellyfin_heatmap_plays` | gauge | `day`, `hour` | Vistas por día de la semana × hora, para un mapa de calor. |

## Contenido {#content}

| Métrica | Tipo | Etiquetas | Descripción |
|---|---|---|---|
| `jellyfin_item_play_count` | gauge | `name`, `type` | Vistas de los 50 títulos principales (a nivel de película, serie o álbum). |
| `jellyfin_items_by_type_total` | gauge | `type` | Elementos de las bibliotecas: `Movie`, `Series`, `Episode`, `Audio`. |
| `jellyfin_unwatched_items_by_type` | gauge | `type` | Elementos nunca vistos, por tipo. |

## Finalización y hábitos {#completion-and-habits}

| Métrica | Tipo | Etiquetas | Descripción |
|---|---|---|---|
| `jellyfin_completed_plays_total` | counter | – | Unidades de finalización terminadas (≥ 90 %, o el «reproducido hasta el final» de Jellyfin para las reproducciones del plugin). Una unidad = un usuario, un título y un día local. |
| `jellyfin_abandoned_plays_total` | counter | – | Unidades de finalización por debajo del 90 %. |
| `jellyfin_completion_rate_percent` | gauge | – | `completed ÷ (completed + abandoned) × 100`. |
| `jellyfin_binge_sessions_total` | counter | – | (usuario, serie, día) con al menos 3 episodios distintos vistos. |

## En directo {#live}

| Métrica | Tipo | Etiquetas | Descripción |
|---|---|---|---|
| `jellyfin_active_sessions` | gauge | – | Películas, episodios y pistas que se están reproduciendo ahora mismo. |

## Ejemplo de salida {#sample-output}

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

## Ejemplos de PromQL {#promql-examples}

```txt
# Vistas en las últimas 24 horas
increase(jellyfin_total_plays[24h])

# Horas vistas por día y por usuario
increase(jellyfin_hours_by_user_total[1d])

# Los 5 usuarios con más vistas
topk(5, jellyfin_plays_by_user_total)

# Parte de transcodificación (%)
100 * jellyfin_plays_by_playback_method_total{method="Transcode"}
  / ignoring(method) sum without(method) (jellyfin_plays_by_playback_method_total)

# Parte del catálogo nunca vista, por tipo
100 * jellyfin_unwatched_items_by_type / jellyfin_items_by_type_total

# Hora con más actividad
topk(1, jellyfin_plays_by_hour)
```
