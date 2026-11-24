# Configuración

Jellystics se configura en dos lugares: las **variables de entorno** al arrancar (consulta [Instalación](/es/guide/installation#environment-variables)) y los **Ajustes** del panel (menú de usuario → *Ajustes*). Esta página enumera todos los ajustes; el [recorrido por los ajustes](/es/platform/settings) muestra las pantallas.

## Servidor Jellyfin {#jellyfin-server}

*Ajustes → Configuración → Servidor Jellyfin*

| Campo | Descripción |
|---|---|
| URL de Jellyfin | Debe empezar por `http://` o `https://`. Es la dirección que usa Jellystics para leer bibliotecas, usuarios y sesiones. |
| Clave API | Déjala vacía para conservar la clave guardada. Necesaria para probar la conexión. Crea una en Jellyfin: *Panel de control → Claves API*. |
| URL pública de Jellystics | Dirección pública de Jellystics. Se usa como avatar predeterminado del bot (`<url>/logo.png`) en las notificaciones de Discord. **No** es la dirección enviada al plugin (consulta `JELLYSTICS_PUBLIC_URL`). |

*Probar la conexión* comprueba la URL y la clave, y responde «Conectado a *nombre del servidor*».

## Tarjeta del plugin Jellystics {#jellystics-plugin-card}

*Ajustes → Configuración → Plugin Jellystics* muestra el estado del [plugin](/es/plugin/):

- **Activo**: el plugin respondió hace menos de 2 minutos. **Sin sincronizar**: ninguna sincronización correcta desde hace más de 2 minutos (mientras tanto, el seguimiento integrado registra las reproducciones). **No detectado**: instálalo con la URL del manifiesto que aparece en la tarjeta.
- Versión, esquema de API, última sincronización, reproducciones sincronizadas, último error.
- **Página «Mis estadísticas»**: estado del vínculo (Activa, Jellystics inaccesible desde Jellyfin, Reconectando, Aún no vinculada) y dirección enviada al plugin.
- **Guardar el historial en el plugin**: envía al plugin las reproducciones que Jellystics registró por sí mismo (seguimiento, importaciones, restauraciones), para que el plugin contenga el historial completo. Se puede repetir sin riesgo; se reanuda tras un error o un reinicio.
- **Resincronizar desde el plugin**: vuelve a leer todo desde el plugin. Las reproducciones registradas dos veces se sustituyen por la versión del plugin.

Detalles: [Sincronización con Jellystics](/es/plugin/sync).

## Tarjeta de calidad de los datos {#data-quality-card}

Muestra de dónde vienen las reproducciones de un periodo (plugin, seguimiento, importaciones de Playback Reporting, otras importaciones, copias restauradas) y qué detalles capturó cada fuente (posición de parada, pista de audio/subtítulos, detalles de reproducción, detalles de transcodificación). También cuenta las duraciones importadas que se repararon y las reproducciones sin fecha utilizable (excluidas de todas las estadísticas).

## Tareas programadas {#scheduled-tasks}

*Ajustes → Tareas*. Las expresiones cron tienen 5 campos (`minute hour day month weekday`) y siguen `TZ`.

| Tarea | Programación predeterminada | Qué hace |
|---|---|---|
| Full Jellyfin Sync | `0 0 * * *` (siempre activa) | Sincroniza todas las bibliotecas, elementos, temporadas, episodios, pistas y usuarios. También lanza la geolocalización por IP. |
| Recently Added Sync | `0 * * * *` (desactivada por defecto) | Sincroniza los 100 últimos elementos de cada biblioteca. |
| Backup | `0 3 * * *` (desactivada por defecto) | Crea una copia de seguridad en `/app/data/backups`. |
| Jellyfin Playback Reporting Plugin Sync | `0 4 * * *` (desactivada por defecto) | Importa el historial del plugin oficial Playback Reporting. Se omite cuando se detecta el plugin Jellystics (el plugin lo importa por sí mismo). |

Trabajos en segundo plano que no aparecen en la lista: la sincronización del plugin (cada 30 s), el seguimiento (cada 5 s), la limpieza de sesiones (diaria) y la geolocalización (2 min después del arranque y luego a diario).

::: warning
Solo se ejecuta una tarea a la vez: una tarea lanzada mientras otra está en curso se omite. En una instalación nueva, guarda una vez la programación de Full Jellyfin Sync para que quede programada.
:::

La misma pestaña importa una **exportación TSV/TXT de Playback Reporting** (el archivo que exporta el plugin oficial): cada línea se convierte en una reproducción, las duraciones negativas se reparan y las reproducciones ya registradas por otra fuente se descartan.

## Copias de seguridad {#backups}

*Ajustes → Copias*

- **Crear una copia** escribe `backup_YYYY-MM-DD HH-mm-ss.json` en `/app/data/backups`. Se conservan las 5 copias más recientes.
- Una copia contiene: bibliotecas, elementos, temporadas, episodios, usuarios, todas las reproducciones y las filas de Playback Reporting aún no importadas. **No** contiene los ajustes, las claves API, los webhooks ni las pistas de música: estos se recrean con una sincronización o hay que volver a configurarlos.
- **Restaurar** inserta o actualiza la copia en la base de datos (las reproducciones existentes se actualizan y las ya registradas por otra fuente se omiten) y después reinicia los cursores del plugin. También se aceptan copias de Jellystat.
- **Subir una copia** acepta archivos `.json`.

## Webhooks {#webhooks}

*Ajustes → Webhooks* envía notificaciones a **Discord**.

| Campo | Descripción |
|---|---|
| Nombre | Texto libre. |
| URL del webhook de Discord | `https://discord.com/api/webhooks/…` (en Discord: *ajustes del canal → Integraciones → Webhooks*). |
| Nombre del bot | Por defecto `jellystics_bot`. |
| URL del avatar del bot | Opcional; por defecto, el logotipo de la URL pública de Jellystics. |
| Notificaciones | Al menos una de: tarea iniciada, tarea completada, tarea fallida, copia completada, clave API creada, clave API eliminada. |

Cada webhook se puede activar/desactivar, editar, eliminar o probar. El envío tiene un tiempo de espera de 10 s y no se reintenta.

## Claves API {#api-keys}

*Ajustes → Claves API*. Una clave solo se muestra completa una vez, al crearla. Las claves autentican [`/metrics`](/es/prometheus/) con `Authorization: Bearer <key>`. Crear o eliminar una clave puede notificar a un webhook.

## Registros {#logs}

*Ajustes → Registros* muestra las 50 últimas ejecuciones de tareas (nivel, fecha, tarea, mensaje) con búsqueda y filtro por nivel. *Ajustes → Tareas* también muestra en directo la salida de la tarea en curso.

## Acerca de {#about}

Versión instalada y última versión, enlace a la release y «Buscar actualizaciones».

## Idioma y tema del panel {#dashboard-language-and-theme}

- **Idioma**: menú de usuario → *Idioma* (English, Français, Español). Se guarda en el navegador (`i18nextLng`); `?lng=fr` en la URL también funciona.
- **Tema**: menú de usuario → *Tema claro / Tema oscuro*, o `⌘K` → *Cambiar tema*. Si no eliges ninguno, sigue al sistema.
