# Introducción

Jellystics es un sistema de estadísticas para [Jellyfin](https://jellyfin.org). Responde a preguntas como *quién ve qué, cuándo, en qué dispositivo, hasta dónde y con qué calidad de reproducción*, y conserva la respuesta durante años.

![Página Resumen](/screenshots/es/overview.jpg){.shot}

## Los dos componentes {#the-two-components}

| Componente | Dónde se ejecuta | Función |
|---|---|---|
| **Jellystics** | Contenedor Docker (API en Go, panel en React) + PostgreSQL | Almacena el historial, calcula las estadísticas y sirve el panel, la API, `/metrics`, los webhooks y las copias de seguridad. |
| **Plugin Jellystics** | Dentro de Jellyfin 10.11 | Registra cada reproducción a partir de los eventos del servidor en su propia base de datos SQLite y ofrece la página «Mis estadísticas» a cada usuario de Jellyfin. |

Jellystics funciona por sí solo. El plugin aporta más precisión y una copia de seguridad:

| | Sin el plugin | Con el plugin |
|---|---|---|
| Fuente | Jellystics consulta `/Sessions` de Jellyfin cada 5 s | Jellyfin envía los eventos al plugin en tiempo real |
| Tiempo de reproducción | Muy bueno, limitado por el sondeo cada 5 s | Exacto, a partir de los eventos de reproducción |
| Reinicio de Jellyfin | La reproducción en curso se cierra cuando desaparece | Como máximo 15 s perdidos (puntos de control) |
| Saltos hacia delante / saltar la intro | – | Contados, con el tiempo saltado |
| «Reproducido hasta el final» | Estimado (≥ 90 % de la duración) | Lo indica Jellyfin |
| Altas y bajas en las bibliotecas | Fechas del catálogo (aproximadas) | Eventos reales |
| Inicios de sesión, «marcado como visto» a mano | – | Registrados |
| Copia del historial dentro de Jellyfin | – | Historial completo en `jellystics.db` |
| «Mis estadísticas» para los usuarios | – | Sí |

Cuando el plugin está en buen estado, es la fuente de referencia; cuando deja de responder, Jellystics vuelve automáticamente a su propio seguimiento, sin huecos ni duplicados. Consulta [cómo funciona](/es/guide/how-it-works).

## Qué obtienes {#what-you-get}

- **Panel**: resumen con comparación de periodos, sesiones en directo, historial (tabla y calendario), [bibliotecas y páginas de contenido](/es/platform/libraries), [usuarios y resúmenes](/es/platform/users), [análisis, descubrimiento y salud de la reproducción](/es/platform/analysis).
- **Definiciones precisas**: clics, vistas (≥ 30 s), finalización, maratones, reanudaciones, revisionados. Consulta el [glosario](/es/guide/glossary).
- **Integraciones**: [métricas de Prometheus y Grafana](/es/prometheus/), webhooks de Discord y una API HTTP con claves API.
- **Operaciones**: tareas programadas, copias de seguridad y restauración, importación del historial del plugin oficial Playback Reporting.
- **Tres idiomas**: inglés, francés y español, en el panel y en esta documentación.

## Requisitos {#requirements}

- Jellyfin **10.11** (para el plugin; Jellystics en sí solo necesita la API HTTP de Jellyfin).
- Docker y Docker Compose para Jellystics.
- Una **clave API** de Jellyfin y una cuenta de **administrador** de Jellyfin para iniciar sesión en el panel.

## Próximos pasos {#next-steps}

1. [Instala Jellystics](/es/guide/installation).
2. [Instala el plugin](/es/plugin/).
3. [Conecta Prometheus y Grafana](/es/prometheus/) si los usas.
