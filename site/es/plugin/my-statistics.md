# Página «Mis estadísticas»

Cada usuario de Jellyfin tiene una entrada **Mis estadísticas** en el menú del avatar (la segunda, después de *Perfil*). Abre `/Jellystics/Me/` en el servidor Jellyfin y solo muestra las estadísticas de ese usuario.

![Mis estadísticas](/screenshots/es/me.jpg){.shot}

## Qué ven los usuarios {#what-users-see}

- Tiempo de reproducción, vistas, clics, vistas terminadas, finalización media y racha actual, comparados con el periodo anterior.
- Periodo: 7 días, 30 días, 90 días, 1 año o desde el principio.
- Tiempo de reproducción, vistas o clics por día, contenidos favoritos, reparto por tipo de contenido y por método de reproducción, un mapa de calor día de la semana × hora, cada reproducción como un punto, dispositivos y aplicaciones, y su historial completo.

La página sigue el idioma de Jellyfin del usuario (inglés, francés o español) y su tema. Si Jellystics aún no ha sincronizado la cuenta, muestra «Tus estadísticas están en camino».

## Cómo funciona {#how-it-works}

```txt
Browser ──► Jellyfin /Jellystics/Me/…  ──► Jellystics /me/…        (page and assets)
Browser ──► Jellyfin /Jellystics/Me/api/… ──► Jellystics /api/me/… (statistics, with the secret)
```

1. El plugin añade un pequeño script (`/Jellystics/Me/client.js`) al `index.html` de jellyfin-web. El script añade la entrada de menú, con una etiqueta en el idioma de la página.
2. El navegador **solo habla con Jellyfin**. Las llamadas de estadísticas requieren haber iniciado sesión en Jellyfin.
3. El plugin las reenvía a Jellystics con dos cabeceras: `X-Jellystics-Secret` (el secreto compartido) y `X-Jellystics-User` (el id de usuario tomado de la autenticación de Jellyfin, nunca del navegador). Por tanto, un usuario solo puede leer sus propios datos.
4. Jellystics solo responde con el secreto correcto y a un usuario conocido.

Los usuarios nunca necesitan una cuenta de Jellystics, y Jellystics no tiene que estar expuesto a Internet: solo el servidor Jellyfin debe poder llegar a él.

## Requisitos {#requirements}

- El plugin está vinculado: *Panel de control → Complementos → Jellystics* indica «Linked … reachable».
- Jellyfin puede llegar a Jellystics. Si la dirección detectada automáticamente no es correcta, define `JELLYSTICS_PUBLIC_URL` en Jellystics o la **URL override** en los ajustes del plugin.
- El panel está activado (`DISABLE_DASHBOARD` no vale `true`).

## Solución de problemas {#troubleshooting}

| La página dice | Causa | Solución |
|---|---|---|
| Estadísticas no disponibles (no vinculado) | Jellystics aún no se ha vinculado. | Comprueba la tarjeta del plugin en Jellystics (*Activo*) y espera 30 s. |
| Estadísticas no disponibles (inaccesible) | Jellyfin no puede llegar a la URL de Jellystics. | Define la URL override o `JELLYSTICS_PUBLIC_URL`. |
| Estadísticas no disponibles (reconectando) | Se rechazó el secreto; Jellystics se vuelve a vincular en menos de un minuto. | Asegúrate de que solo una instancia de Jellystics apunta a este Jellyfin. |
| Inicia sesión en Jellyfin | No hay sesión de Jellyfin en este navegador. | Inicia sesión primero en Jellyfin. |
| Falta la entrada de menú | El navegador tiene en caché el `index.html` antiguo. | Recarga sin caché; comprueba que el plugin está activado. |

## Seguridad {#security}

La página y sus scripts vienen de Jellystics pero se ejecutan en el origen de Jellyfin, donde podrían usar la sesión de Jellyfin del usuario conectado. Por tanto, Jellyfin confía en la instancia de Jellystics a la que está vinculado tanto como en su propio cliente web. Ejecuta Jellystics en una red local o una red Docker de confianza, o dale una URL `https`.

Protecciones del plugin:

- solo se reenvían la página, `assets/**`, `locales/**/*.json` y las imágenes de la raíz, cada una con un tipo de contenido permitido;
- Content-Security-Policy estricta en la página (`script-src 'self'`, `connect-src 'self'`, `frame-ancestors 'self'`), `X-Content-Type-Options: nosniff`, `Referrer-Policy: same-origin`;
- no se siguen redirecciones, tiempo de espera de 15 s, límite de tamaño de 10 MB;
- las estadísticas nunca se almacenan en caché (`Cache-Control: private, no-store`);
- los parámetros de query como `userId` o `api_key` se eliminan antes de reenviar;
- el secreto solo vive en el XML del plugin y ninguna API lo devuelve nunca.
