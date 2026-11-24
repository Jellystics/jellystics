# Ajustes del plugin

Abre **Jellyfin → Panel de control → Complementos → Jellystics** (o *Jellystics* en la sección del servidor del menú del panel de control). La página tiene cuatro bloques (sus etiquetas están en inglés).

## Estado {#status}

Se lee de `GET /Jellystics/Info`:

```txt
Version 0.2.1.0
1 840 plays, 3 512 library events, 412 marks, 96 logins
First play 2024-03-02
Sync cursor 58 231
```

El cursor es el número de versión global que usa Jellystics para leer lo que ha cambiado (consulta [Sincronización](/es/plugin/sync)).

## Página «Mis estadísticas» {#my-statistics-page}

Bloque «My statistics page». Cada usuario tiene una entrada *Mis estadísticas* en el menú de su avatar. Jellystics se vincula automáticamente en su siguiente sincronización: envía al plugin su URL y un secreto compartido. Este bloque muestra el resultado:

| Mensaje | Significado | Qué hacer |
|---|---|---|
| Not linked | Jellystics aún no ha enviado su dirección. | Espera a la siguiente sincronización (30 s). Comprueba que la tarjeta del plugin está *Activo* en Jellystics. |
| Linked to `<url>` · reachable | Todo correcto. | – |
| UNREACHABLE from Jellyfin | Jellyfin no puede llegar a esa dirección. | Define la **URL override** con una dirección de Jellystics a la que pueda llegar el servidor Jellyfin, o define `JELLYSTICS_PUBLIC_URL` en Jellystics. |
| Jellystics rejected the shared secret | El secreto ha cambiado. | Jellystics se vuelve a vincular en menos de un minuto. Si persiste, asegúrate de que solo una instancia de Jellystics apunta a este Jellyfin. |

**«Jellystics URL override»**: déjalo vacío para usar la URL enviada por Jellystics. Rellénalo cuando Jellystics sea accesible desde Jellyfin en otra dirección, por ejemplo `http://192.168.1.10:3000` o `http://jellystics:3000` en una red Docker compartida. Debe ser una URL `http` o `https` absoluta, sin query, fragmento ni credenciales.

El secreto se guarda solo en la configuración XML del plugin; ninguna API lo devuelve nunca (solo una huella SHA-256).

## Importar el historial {#import-history}

Importa el historial del plugin oficial **Playback Reporting**.

- **«Path of playback_reporting.db»**: déjalo vacío para usar `<jellyfin data>/playback_reporting.db`, o indica la ruta de una copia.
- **«Import history»** guarda la ruta y luego importa en segundo plano. La línea de estado muestra `read / imported / skipped / repaired`.

Qué hace la importación:

- abre el archivo en **solo lectura**, de 500 en 500 filas;
- da a cada fila el id `pbr-<rowid>`: volver a ejecutar la importación nunca duplica nada;
- fija el inicio en la hora registrada − 20 s (Playback Reporting registra el final de los 20 primeros segundos), interpretada en la zona horaria del servidor;
- repara las duraciones imposibles: negativa → 0; más de 1,5 × la duración → la duración; más de 12 h cuando la duración es desconocida → 12 h. Las filas reparadas quedan marcadas;
- omite las filas que se solapan con una reproducción registrada por el propio plugin;
- completa serie, temporada, álbum, duración y biblioteca a partir de la biblioteca actual de Jellyfin cuando el elemento todavía existe.

## Purgar las reproducciones anteriores a {#purge-plays-before}

Elimina definitivamente las reproducciones **cerradas** que empezaron antes de la fecha elegida (medianoche en la zona horaria de tu navegador).

::: warning
La purga es local al plugin. **No** se propaga a Jellystics, y volver a importar Playback Reporting restaura las filas purgadas que procedían de él.
:::
