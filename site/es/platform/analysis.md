# Análisis, Descubrir y Reproducción

## Análisis {#insights}

![Análisis](/screenshots/es/insights.jpg){.shot}

Cómo ve la gente:

- **Distribución de finalización**: vistas por tramos del 10 %; el pico del 100 % son las vistas terminadas. **Vistas terminadas por tipo**.
- **Maratones** por serie, **pausas** (tiempo en pausa ÷ tiempo visto por título), mapa de calor del **ritmo semanal**.
- **Duración de las sesiones** (mediana, sin falsos inicios), **reanudadas vs desde el inicio**, **revisionados**.
- **Intros más saltadas** (plugin), **géneros en el tiempo**, **momentos del día por tipo**.

## Descubrir {#discover}

![Descubrir](/screenshots/es/discover.jpg){.shot}

Lo que duerme en las bibliotecas:

- **Nunca vistos**: número, parte del catálogo y espacio en disco, empezando por las incorporaciones más antiguas.
- **Abandonados**: títulos detenidos entre el 10 y el 80 % y sin reproducir desde hace más de 60 días.
- **Añadidos en el periodo** y cuántos se han visto ya.
- **Almacenamiento vs visionado**: tamaño de archivo frente a vistas; los archivos grandes nunca vistos destacan en rojo.

## Reproducción {#playback}

![Reproducción y transcodificación](/screenshots/es/playback.jpg){.shot}

Salud de la reproducción:

- Partes de **direct play** y de **transcodificación**, **causa n.º 1** de transcodificación, **cliente más transcodificado**.
- **Método de reproducción por día**, **clientes** (% transcodificado), **falsos inicios por cliente**.
- **Resolución realmente servida** (4K, 1080p, 720p, SD) y archivos 4K servidos por debajo de 4K; **HDR** (nativo, con mapeo de tonos, SDR).
- **Por qué se transcodifica**: los motivos que indica Jellyfin (contenedor, códec de vídeo/audio/subtítulos, tasa de bits, resolución, profundidad de bits, rango HDR…).
- **Vídeo o solo audio**, aceleración por hardware, **conversiones de códec**.
- **Local o remoto** con ubicaciones (geolocalización por IP), distribución de la **tasa de bits** (media y percentil 95).

Las tarjetas que dependen de la captura detallada (motivos, red, tasa de bits) indican «Datos recopilados desde el…» cuando los datos empiezan dentro del periodo.
