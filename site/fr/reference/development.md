# Développement

## Plugin {#plugin}

Dépôt : [Jellystics/jellystics-plugin-playbackreporting](https://github.com/Jellystics/jellystics-plugin-playbackreporting).

Utilisez le shell de développement Nix (`nix develop`, ou direnv) : .NET 9, Node, go-task, Docker et SQLite.

```bash
task test    # tests xUnit, puis node --test pour client.js
task build   # build, copie la dll dans config/plugins/Jellystics/, redémarre le Jellyfin de dev
task logs    # suit les logs du Jellyfin de dev
```

Le Jellyfin de développement (`docker compose up -d`) tourne sur http://localhost:8097 (conteneur `jellystics-plugin-jellyfin`). `config/` et `media/` sont ignorés par git.

Vérification manuelle de bout en bout :

1. `docker compose up -d`, `task build`, ouvrez http://localhost:8097, créez l'administrateur et ajoutez `/media` comme bibliothèque.
2. `docker logs jellystics-plugin-jellyfin 2>&1 | grep -i jellystics` : le plugin doit se charger.
3. Lisez un extrait pendant 60 s, mettez en pause 30 s, avancez de 2 min, lisez 30 s, arrêtez, puis :

   ```bash
   sqlite3 config/plugins/Jellyfin.Plugin.Jellystics/jellystics.db \
     "select WatchedMs, PausedMs, PauseCount, SeekCount, SkippedMs, State from Plays"
   ```

   Attendez-vous à environ `90000 | 30000 | 1 | 1 | ~120000 | closed`. Sur macOS, lisez la base après `docker stop`, ou copiez d'abord `jellystics.db*`.

### Releases {#releases}

Poussez un tag en quatre parties, par exemple `v0.2.2.0`. Le workflow de release exécute les tests, compile la dll, la zippe, ajoute la version à `manifest.json` sur `master` et publie une release GitHub.

## Jellystics {#jellystics}

Dépôt : [Jellystics/jellystics](https://github.com/Jellystics/jellystics). Nécessite Go, Node avec pnpm, Docker et Task.

```bash
task setup      # crée backend/.env
task install    # dépendances du frontend
task db         # PostgreSQL dans Docker
task backend    # API Go sur :3000
task frontend   # Vite sur :3001, avec proxy vers :3000
```

Vérifications : `pnpm test`, `pnpm test:e2e`, `pnpm lint`, `pnpm build` dans `frontend/`, et `go test ./...` dans `backend/` (nécessite PostgreSQL).

## Ce site {#this-website}

Le site se trouve dans `site/` du dépôt Jellystics et est construit avec [VitePress](https://vitepress.dev).

```bash
cd site
pnpm install
pnpm docs:dev       # http://localhost:5173/jellystics/
pnpm docs:build     # sortie dans site/.vitepress/dist
pnpm docs:preview   # sert le build sur http://localhost:4173/jellystics/
```

- Les pages anglaises sont à la racine de `site/`, les françaises dans `site/fr/`, les espagnoles dans `site/es/`, avec les mêmes noms de fichiers.
- Les captures d'écran sont dans `site/public/screenshots/<lang>/`. Elles sont prises depuis le frontend de Jellystics avec ses mocks d'API e2e, et ne contiennent donc jamais de données réelles.
- Un push sur `main` déploie le site avec GitHub Actions (`.github/workflows/pages.yml`).
