# Instalación

Jellystics se ejecuta como un contenedor Docker junto a una base de datos PostgreSQL.

## Docker Compose {#docker-compose}

Crea una carpeta y un archivo `docker-compose.yaml`:

```yaml
services:
  jellystics-db:
    image: postgres:18.1
    shm_size: '1gb'
    container_name: jellystics-db
    restart: unless-stopped
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: mypassword
      POSTGRES_DB: jellystics
    volumes:
      - postgres-data:/var/lib/postgresql
    healthcheck:
      test: ["CMD-SHELL", "pg_isready --dbname=postgres --username=postgres"]
      interval: 10s
      timeout: 5s
      retries: 5

  jellystics:
    image: ghcr.io/jellystics/jellystics:latest
    container_name: jellystics
    restart: unless-stopped
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: mypassword
      POSTGRES_IP: jellystics-db
      POSTGRES_PORT: 5432
      JWT_SECRET: "change-me-to-a-random-secret"
      TZ: Europe/Paris
    volumes:
      - jellystics-backup-data:/app/data
    ports:
      - "3000:3000"
    depends_on:
      jellystics-db:
        condition: service_healthy

volumes:
  postgres-data:
  jellystics-backup-data:
```

Después:

```bash
docker compose up -d
```

y abre `http://<host>:3000`. La sección [primer arranque](#first-start) más abajo te guía por el asistente.

::: warning Cambia los secretos
Sustituye `mypassword` y `JWT_SECRET`. `JWT_SECRET` firma las sesiones del panel: usa una cadena larga y aleatoria, por ejemplo `openssl rand -hex 32`. Jellystics se niega a arrancar sin ella.
:::

::: tip Zona horaria
Define `TZ` con tu zona horaria. Cada «día», «hora» y «día de la semana» de las estadísticas se calcula en esta zona horaria, y las tareas programadas la siguen. Sin ella, Jellystics usa UTC y registra una advertencia.
:::

El volumen `/app/data` contiene las copias de seguridad (`/app/data/backups`) y los archivos subidos para restaurar (`/app/data/imports`).

## Variables de entorno {#environment-variables}

| Variable | Obligatoria | Valor predeterminado | Descripción |
|---|---|---|---|
| `JWT_SECRET` | **sí** | – | Secreto que firma las sesiones del panel. |
| `POSTGRES_IP` | no | `localhost` | Host de PostgreSQL. |
| `POSTGRES_PORT` | no | `5432` | Puerto de PostgreSQL. |
| `POSTGRES_USER` | no | `postgres` | Usuario de PostgreSQL. |
| `POSTGRES_PASSWORD` | no | vacío | Contraseña de PostgreSQL. |
| `POSTGRES_DB` | no | `jellystics` | Nombre de la base de datos. |
| `DATABASE_URL` | no | – | Cadena de conexión completa de PostgreSQL. Prevalece sobre todas las variables `POSTGRES_*`. |
| `PORT` | no | `3000` | Puerto HTTP. |
| `TZ` | no | UTC | Zona horaria de las estadísticas y del planificador, p. ej. `Europe/Paris`. |
| `JF_HOST` | no | – | URL de Jellyfin, p. ej. `http://jellyfin:8096`. Solo se usa cuando aún no hay ningún servidor configurado: un servidor guardado en el asistente o en los ajustes siempre tiene prioridad. |
| `JF_API_KEY` | no | – | Clave API de Jellyfin, con la misma regla que `JF_HOST` (solo se rellena cuando no hay ninguna clave guardada y el host guardado es `JF_HOST`). |
| `JELLYSTICS_PUBLIC_URL` | no | detectada automáticamente | URL de Jellystics **tal como la ve el servidor Jellyfin**. Se envía al plugin para la página «Mis estadísticas». Se detecta automáticamente como `http://<IP local usada para llegar a Jellyfin>:<PORT>`. |
| `DISABLE_DASHBOARD` | no | `false` | `true` sirve solo la API y `/metrics` (sin panel ni «Mis estadísticas»). |
| `METRICS_API_KEY` | no | – | Añade este valor como clave API llamada `metrics` en cada arranque. Necesaria con `DISABLE_DASHBOARD=true`, ya que no hay interfaz para crear claves. |
| `JS_GEOLITE_ACCOUNT_ID` | no | – | ID de cuenta de MaxMind GeoLite, para la geolocalización por IP de las reproducciones remotas. |
| `JS_GEOLITE_LICENSE_KEY` | no | – | Clave de licencia de MaxMind GeoLite. Sin ambas, las reproducciones se clasifican como locales/remotas pero no se geolocalizan. |

### Secretos de Docker {#docker-secrets}

En la imagen Docker, cualquier variable puede leerse desde un archivo con el prefijo `FILE__`:

```yaml
environment:
  FILE__JWT_SECRET: /run/secrets/jwt_secret
  FILE__POSTGRES_PASSWORD: /run/secrets/db_password
```

Si falta el archivo, el contenedor se detiene con un error. Si se definen `X` y `FILE__X` a la vez, gana el archivo (se registra una advertencia).

### Modo solo métricas {#metrics-only-mode}

Para usar Jellystics únicamente como exportador de Prometheus:

```yaml
environment:
  DISABLE_DASHBOARD: "true"
  METRICS_API_KEY: "a-long-random-key"
  JF_HOST: http://jellyfin:8096
  JF_API_KEY: your-jellyfin-api-key
```

## Primer arranque {#first-start}

1. Abre Jellystics. El paso **Conectar Jellyfin** pide la URL de Jellyfin (por ejemplo `http://192.168.1.10:8096`) y una clave API (en Jellyfin: *Panel de control → Claves API → +*).
2. El paso **Iniciar sesión** pide un nombre de usuario y una contraseña de Jellyfin. Solo los **administradores de Jellyfin** pueden usar el panel; las demás cuentas reciben «Se requiere acceso de administrador». Los usuarios normales tienen en su lugar su propia página [«Mis estadísticas»](/es/plugin/my-statistics) dentro de Jellyfin.
3. Jellystics ejecuta una primera sincronización completa de bibliotecas y usuarios. Las reproducciones empiezan a registrarse de inmediato.

::: tip Programa la sincronización completa una vez
En una instalación nueva, abre *Ajustes → Tareas* y guarda una vez la programación de **Full Jellyfin Sync** (el cron predeterminado `0 0 * * *` sirve). Hasta que no se guarde un ajuste de tarea, la sincronización completa nocturna no queda programada.
:::

Cómo funciona el inicio de sesión:

- Jellystics te autentica contra Jellyfin y luego mantiene una sesión vinculada a tu token de Jellyfin.
- Cada 5 minutos se comprueba la sesión con Jellyfin: si el token se revoca o dejas de ser administrador, la sesión termina.
- Las sesiones sin uso durante 30 días se revocan. *Cerrar sesión* también revoca el token de Jellyfin.

## Proxy inverso {#reverse-proxy}

Jellystics escucha en el puerto `3000`. La página En directo usa un WebSocket (Socket.IO), así que reenvía las cabeceras `Upgrade` y `Connection`. Con Nginx:

```nginx
location / {
    proxy_pass http://localhost:3000;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
}
```

Caddy reenvía los WebSockets por sí solo:

```txt
stats.example.com {
    reverse_proxy localhost:3000
}
```

## Actualizar {#updating}

```bash
docker compose pull
docker compose up -d
```

Las migraciones de la base de datos se ejecutan automáticamente al arrancar. *Ajustes → Acerca de* te avisa cuando hay una nueva versión disponible.

## Siguiente {#next}

- [Configurar Jellystics](/es/guide/configuration)
- [Instalar el plugin](/es/plugin/)
