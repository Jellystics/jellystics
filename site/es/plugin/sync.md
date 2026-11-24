# Sincronización con Jellystics

Jellystics **extrae** los datos del plugin; el plugin nunca llama a Jellystics, salvo para comprobar el vínculo de «Mis estadísticas» y para servir esa página.

## Detección {#detection}

Cada 30 segundos, Jellystics llama a `GET /Jellystics/Info` en el servidor Jellyfin con su clave API de Jellyfin.

- **404**: el plugin no está instalado. Jellystics sigue usando su seguimiento.
- **schemaVersion** distinto del que admite Jellystics (1): el plugin se ignora y se muestra un error en la tarjeta del plugin.
- En los demás casos, el plugin queda *detectado* y la sincronización se ejecuta.

## Lectura de los cambios {#reading-changes}

Para cada flujo (reproducciones, eventos de biblioteca, eventos de datos de usuario, inicios de sesión):

```http
GET /Jellystics/Plays?since=<cursor>&limit=500
```

```json
{ "items": [ … ], "next": 58231, "hasMore": true }
```

Jellystics guarda `next` después de cada página y continúa mientras `hasMore` sea verdadero (hasta 1000 páginas por ejecución; el resto sigue en la siguiente).

## Quién registra las reproducciones {#who-records-the-plays}

| Estado del plugin | Quién escribe las reproducciones |
|---|---|
| **En buen estado** (sincronización correcta hace menos de 2 min) | El plugin. El seguimiento sigue observando las sesiones para la página En directo, pero no escribe reproducciones. |
| **En mal estado** | El seguimiento. Sus reproducciones se conservan salvo que una reproducción cerrada del plugin ya las cubra. |
| **De vuelta en buen estado** | El plugin. Sus reproducciones cerradas sustituyen a las filas del seguimiento que cubren. |

Eliminación de duplicados:

- después de cada página, se eliminan las reproducciones del seguimiento cubiertas por una nueva reproducción del plugin (mismo usuario, elemento y dispositivo, dentro de la reproducción del plugin con 10 s de margen);
- después de cada ejecución, se repite la misma comprobación sobre las últimas 24 horas;
- cuando el plugin lleva más de una hora en mal estado, se cierran sus reproducciones que llevan abiertas más de una hora.

::: info Relojes
La correspondencia compara las horas de inicio entre Jellyfin y Jellystics: mantén sincronizados los relojes de ambos hosts (NTP).
:::

## Reinicio y resincronización {#reset-and-resync}

- Si cambia el `databaseId` del plugin o su cursor retrocede (base de datos eliminada o restaurada), Jellystics reinicia sus cursores y lo vuelve a leer todo. Las inserciones/actualizaciones son idempotentes: nada se duplica.
- **Resincronizar desde el plugin** (*Ajustes → Configuración*) hace lo mismo bajo demanda. Restaurar una copia de seguridad de Jellystics también.

## Enviar el historial de Jellystics al plugin {#pushing-jellystics-history-to-the-plugin}

*Guardar el historial en el plugin* envía al plugin las reproducciones que Jellystics registró por sí mismo (seguimiento, importaciones de Playback Reporting, importaciones TSV, copias restauradas):

- películas, episodios y pistas con un usuario, un elemento y una fecha de inicio;
- en orden, en lotes de **500** (`POST /Jellystics/Import/Plays`);
- el plugin las inserta con la fuente `import-jellystics` y omite los ids que ya tiene: volver a ejecutarlo es seguro;
- el progreso se guarda después de cada lote: tras un error o un reinicio, *Reanudar el guardado* continúa donde se detuvo.

Estas reproducciones vuelven a Jellystics mediante la sincronización normal y se asocian a las filas existentes.

## Vínculo de «Mis estadísticas» {#my-statistics-link}

Después de cada sincronización correcta, Jellystics se asegura de que el plugin conoce su dirección y su secreto (`POST /Jellystics/Link`):

- **dirección**: `JELLYSTICS_PUBLIC_URL`, o `http://<IP local que usa Jellystics para llegar a Jellyfin>:<PORT>`;
- **secreto**: 32 bytes aleatorios generados una sola vez y conservados por Jellystics;
- se vuelve a vincular cuando el plugin no está vinculado, cuando la dirección o la huella del secreto difieren, o cuando el plugin informa de un secreto rechazado. Los fallos se reintentan con una espera creciente de 30 s hasta 30 min.

Consulta [la página «Mis estadísticas»](/es/plugin/my-statistics).
