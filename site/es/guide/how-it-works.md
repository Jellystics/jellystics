---
aside: false
---

# Cómo funciona

Esta página sigue una reproducción desde el momento en que alguien pulsa *Reproducir* hasta que aparece en un gráfico y en Prometheus.

<Workflow />

## 1. Registrar una reproducción {#_1-recording-a-playback}

Hay dos registradores. Se usa el plugin cuando está en buen estado; si no, toma el relevo el seguimiento integrado.

### Con el plugin (por eventos) {#with-the-plugin-event-driven}

El plugin escucha los propios eventos de Jellyfin (`PlaybackStart`, `PlaybackProgress`, `PlaybackStopped`) y mantiene una máquina de estados por reproducción. Cada evento suma:

- **tiempo visto** = el menor entre el tiempo real transcurrido y el avance de la posición (+ 2 s de margen). Las pausas, el búfer y los retrocesos no suman nada;
- **tiempo en pausa** mientras el reproductor está en pausa, y una **pausa** por cada paso de reproduciendo → en pausa;
- un **salto** cuando la posición avanza más de 5 s por encima de lo que explica la reproducción, con el **tiempo saltado** en los saltos hacia delante.

Las reproducciones abiertas se guardan cada 15 s; tras un fallo de Jellyfin, una reproducción se cierra en su último punto de control. Todos los detalles: [Seguimiento de las reproducciones](/es/plugin/tracking).

### Sin el plugin (sondeo) {#without-the-plugin-polling}

Jellystics llama a `/Sessions` de Jellyfin cada **5 segundos** y aplica la misma contabilidad:

- cada consulta suma el avance de la posición, con un máximo del tiempo transcurrido + 2 s;
- si un cliente que normalmente informa cada ≤ 30 s se queda en silencio más de 45 s, el tiempo deja de contar;
- cuando la reproducción se detiene, el tiempo desde la última consulta se recupera a partir de la posición que guardó Jellyfin (como máximo 55 s);
- empieza una nueva reproducción cuando cambia el elemento o el usuario, o cuando la posición vuelve al inicio después del final de la anterior (repetición).

Solo se siguen películas, episodios y pistas de música; los tráileres, los extras y los medios temáticos se ignoran.

## 2. Llevar los datos del plugin a Jellystics {#_2-getting-plugin-data-into-jellystics}

Cada **30 segundos**, Jellystics pide al plugin lo que ha cambiado desde la última vez, con un cursor por flujo (reproducciones, eventos de biblioteca, eventos de datos de usuario, inicios de sesión) y 500 filas por página. Un cursor es un número de versión global: cada inserción o actualización en el plugin recibe uno nuevo, de modo que las reproducciones en curso se reenvían a medida que crecen.

- **En buen estado** = el plugin respondió correctamente hace menos de **2 minutos**.
- Mientras el plugin está en buen estado, el seguimiento no escribe reproducciones; de ellas se encarga el plugin.
- Cuando llega una reproducción del plugin, se elimina cualquier fila del seguimiento del mismo usuario, elemento y dispositivo que quede cubierta por ella (10 s de margen), y la misma limpieza se aplica a las últimas 24 horas después de cada sincronización.
- Cuando el plugin no está disponible, se conservan las reproducciones del seguimiento, salvo que una reproducción cerrada del plugin ya las cubra. Cuando el plugin vuelve, su versión cerrada sustituye a la fila del seguimiento.
- Si se recreó la base de datos del plugin (nuevo id de base de datos o un cursor que retrocede), Jellystics reinicia sus cursores y lo vuelve a leer todo; las inserciones/actualizaciones son idempotentes.

Detalles: [Sincronización con Jellystics](/es/plugin/sync).

## 3. De las reproducciones a las estadísticas {#_3-from-plays-to-statistics}

Todas las reproducciones, sea cual sea su origen, acaban en una única tabla. Las estadísticas se calculan en SQL en el momento de la consulta, en la zona horaria `TZ`:

| Noción | Regla |
|---|---|
| Clic | Cualquier reproducción registrada. |
| Vista | Una reproducción con **≥ 30 s** realmente vistos. |
| Tiempo de reproducción | Suma del tiempo visto de **todas** las reproducciones (clics incluidos). |
| Finalización | `max(posición de parada, tiempo visto) / duración`, con un máximo del 100 %, por usuario, título y día local. |
| Vista terminada | Finalización ≥ **90 %** o, para películas y episodios de 5 min o más registrados por el plugin, el indicador «reproducido hasta el final» de Jellyfin. |
| Maratón | ≥ **3 episodios distintos** de una serie por un usuario en un mismo día local. |

La lista completa está en el [glosario](/es/guide/glossary).

## 4. Fuentes y precisión {#_4-sources-and-precision}

Cada reproducción conserva su fuente, visible en *Ajustes → Configuración → Calidad de los datos*:

| Fuente | Precisión |
|---|---|
| Plugin Jellystics | Eventos exactos, pausas, saltos, pistas, transcodificación. |
| Seguimiento Jellystics | Sondeo cada 5 s. |
| Importaciones Playback Reporting | Hora de inicio estimada (hora registrada − 20 s), sin pausas; duraciones imposibles reparadas (negativa → 0, > 1,5 × duración → duración, > 12 h con duración desconocida → 12 h). |
| Otras importaciones, copias restauradas | Tal como las registró la herramienta original. |

Las importaciones nunca crean duplicados: una reproducción ya registrada por otra fuente (mismo usuario y elemento, inicio con menos de 2 min de diferencia, duración similar) se omite.

## 5. El resto de los datos {#_5-the-rest-of-the-data}

- **Bibliotecas, elementos y usuarios** vienen de la API de Jellyfin (sincronización completa, sincronización de lo añadido recientemente y bajo demanda cuando se reproduce un elemento nuevo).
- **Eventos de biblioteca, inicios de sesión y marcas de visto/favorito** vienen solo del plugin.
- **Remoto o local**: según la IP del cliente; las direcciones privadas, de loopback y de enlace local son locales. Las IP remotas se pueden geolocalizar con MaxMind GeoLite (opcional).
- **Las sesiones en directo** se envían al panel por WebSocket cada 5 s.

## 6. Exponer los datos {#_6-exposing-the-data}

- El **panel** lee `/api/v2/*`.
- **Prometheus** lee [`/metrics`](/es/prometheus/metrics), calculado en directo en cada recogida.
- **Los usuarios de Jellyfin** consultan sus propias cifras a través del plugin: [«Mis estadísticas»](/es/plugin/my-statistics).
