# Plugin Jellystics para Jellyfin

El plugin Jellystics se ejecuta dentro de Jellyfin. Registra cada reproducción a partir de los propios eventos del servidor, conserva el historial completo en su propia base de datos y ofrece la página «Mis estadísticas» a cada usuario de Jellyfin.

- **Preciso**: tiempo de reproducción real, pausas, saltos y tiempo saltado, pistas de audio y subtítulos y sus cambios, todos los detalles de transcodificación, «reproducido hasta el final» según Jellyfin.
- **Completo**: altas y bajas en las bibliotecas, inicios de sesión (también los fallidos), marcas de visto y favorito con su motivo.
- **Duradero**: todo se guarda en `jellystics.db` sin límite de tiempo; un fallo pierde como máximo 15 segundos de la reproducción en curso.
- **Ligero**: los eventos se procesan en memoria y los escribe un único proceso en segundo plano, así que Jellyfin nunca se ralentiza.
- **Independiente**: Jellystics funciona sin él y vuelve a su propio seguimiento si el plugin deja de responder.

Se basa en el plugin oficial Playback Reporting (GPL-3), con una identidad nueva: ambos plugins pueden instalarse a la vez, y el plugin Jellystics puede importar el historial de Playback Reporting.

## Requisitos {#requirements}

- Jellyfin **10.11** (`targetAbi 10.11.0.0`, .NET 9).
- Jellystics accesible desde Jellyfin por HTTP, si quieres la página «Mis estadísticas».

## Instalación {#install}

1. En Jellyfin, abre **Panel de control → Complementos → Repositorios** y añade:

   ```txt
   https://raw.githubusercontent.com/Jellystics/jellystics-plugin-playbackreporting/master/manifest.json
   ```

2. Abre **Catálogo**, instala **Jellystics** y después **reinicia Jellyfin**.
3. Comprueba que *Jellystics* aparece en **Panel de control → Complementos** y en el menú del panel de control (sección del servidor).

No hace falta nada más del lado de Jellystics: en menos de 30 segundos, *Ajustes → Configuración → Plugin Jellystics* pasa a **Activo**, y Jellystics vincula automáticamente la página «Mis estadísticas».

::: tip Instalación manual
Descarga `Jellyfin.Plugin.Jellystics-v<version>.zip` desde las [releases](https://github.com/Jellystics/jellystics-plugin-playbackreporting/releases), descomprime `Jellyfin.Plugin.Jellystics.dll` en `<jellyfin config>/plugins/Jellystics/` y reinicia Jellyfin.
:::

## Primeros pasos tras la instalación {#first-steps-after-installing}

1. **Importa el historial antiguo** (opcional): si usabas el plugin oficial Playback Reporting, ejecuta *Import history* en los [ajustes del plugin](/es/plugin/configuration#import-history).
2. **Envía el historial de Jellystics** (opcional): en Jellystics, *Ajustes → Configuración → Plugin Jellystics → Guardar el historial en el plugin*. Así el plugin contiene una copia completa de todo.
3. **Comprueba «Mis estadísticas»**: abre el menú del avatar en Jellyfin; *Mis estadísticas* es la segunda entrada.

## Actualizar {#updating}

Jellyfin muestra las actualizaciones del repositorio en **Panel de control → Complementos**. Cada release añade una entrada a `manifest.json` con su suma de comprobación; instálala y reinicia Jellyfin.

## Desinstalar {#uninstalling}

Elimina el plugin desde **Panel de control → Complementos** y reinicia Jellyfin. Jellystics detecta que ya no está y sigue registrando con su seguimiento. La base de datos del plugin (`<jellyfin config>/plugins/Jellyfin.Plugin.Jellystics/jellystics.db`) se queda en el disco.

## Más información {#learn-more}

- [Ajustes del plugin](/es/plugin/configuration)
- [Cómo se miden las reproducciones](/es/plugin/tracking)
- [Datos recopilados](/es/plugin/data)
- [Sincronización con Jellystics](/es/plugin/sync)
- [Página «Mis estadísticas»](/es/plugin/my-statistics)
- [API del plugin](/es/plugin/api)
