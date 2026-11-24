# Desarrollo

## Plugin {#plugin}

Repositorio: [Jellystics/jellystics-plugin-playbackreporting](https://github.com/Jellystics/jellystics-plugin-playbackreporting).

Usa el entorno de desarrollo de Nix (`nix develop`, o direnv): .NET 9, Node, go-task, Docker y SQLite.

```bash
task test    # pruebas xUnit y luego node --test para client.js
task build   # compila, copia la dll en config/plugins/Jellystics/ y reinicia el Jellyfin de desarrollo
task logs    # sigue los logs del Jellyfin de desarrollo
```

El Jellyfin de desarrollo (`docker compose up -d`) se ejecuta en http://localhost:8097 (contenedor `jellystics-plugin-jellyfin`). `config/` y `media/` están ignorados por git.

Comprobación manual de extremo a extremo:

1. `docker compose up -d`, `task build`, abre http://localhost:8097, crea el administrador y añade `/media` como biblioteca.
2. `docker logs jellystics-plugin-jellyfin 2>&1 | grep -i jellystics`: el plugin debe cargarse.
3. Reproduce un clip 60 s, pausa 30 s, salta 2 min hacia delante, reproduce 30 s, detén y luego:

   ```bash
   sqlite3 config/plugins/Jellyfin.Plugin.Jellystics/jellystics.db \
     "select WatchedMs, PausedMs, PauseCount, SeekCount, SkippedMs, State from Plays"
   ```

   Deberías obtener aproximadamente `90000 | 30000 | 1 | 1 | ~120000 | closed`. En macOS, lee la base de datos después de `docker stop`, o copia antes `jellystics.db*`.

### Releases {#releases}

Sube una etiqueta de cuatro partes, por ejemplo `v0.2.2.0`. El workflow de release ejecuta las pruebas, compila la dll, la comprime en zip, añade la versión a `manifest.json` en `master` y publica una release en GitHub.

## Jellystics {#jellystics}

Repositorio: [Jellystics/jellystics](https://github.com/Jellystics/jellystics). Requiere Go, Node con pnpm, Docker y Task.

```bash
task setup      # crea backend/.env
task install    # dependencias del frontend
task db         # PostgreSQL en Docker
task backend    # API en Go en :3000
task frontend   # Vite en :3001, con proxy a :3000
```

Comprobaciones: `pnpm test`, `pnpm test:e2e`, `pnpm lint`, `pnpm build` en `frontend/`, y `go test ./...` en `backend/` (requiere PostgreSQL).

## Este sitio web {#this-website}

El sitio está en `site/` del repositorio de Jellystics y se genera con [VitePress](https://vitepress.dev).

```bash
cd site
pnpm install
pnpm docs:dev       # http://localhost:5173/jellystics/
pnpm docs:build     # salida en site/.vitepress/dist
pnpm docs:preview   # sirve la build en http://localhost:4173/jellystics/
```

- Las páginas en inglés están en la raíz de `site/`, las francesas en `site/fr/` y las españolas en `site/es/`, con los mismos nombres de archivo.
- Las capturas de pantalla están en `site/public/screenshots/<lang>/`. Se toman del frontend de Jellystics con sus mocks de API e2e, así que nunca contienen datos reales.
- Hacer push a `main` despliega el sitio con GitHub Actions (`.github/workflows/pages.yml`).
