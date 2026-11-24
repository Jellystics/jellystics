# Installation

Jellystics tourne dans un conteneur Docker, à côté d'une base de données PostgreSQL.

## Docker Compose {#docker-compose}

Créez un dossier et un fichier `docker-compose.yaml` :

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

Puis :

```bash
docker compose up -d
```

et ouvrez `http://<host>:3000`. La section [premier démarrage](#first-start) ci-dessous vous guide dans l'assistant.

::: warning Changez les secrets
Remplacez `mypassword` et `JWT_SECRET`. `JWT_SECRET` signe les sessions du tableau de bord : utilisez une longue chaîne aléatoire, par exemple `openssl rand -hex 32`. Jellystics refuse de démarrer sans elle.
:::

::: tip Fuseau horaire
Réglez `TZ` sur votre fuseau horaire. Chaque « jour », « heure » et « jour de la semaine » des statistiques est calculé dans ce fuseau, et les tâches planifiées le suivent. Sans lui, Jellystics utilise UTC et écrit un avertissement dans les logs.
:::

Le volume `/app/data` contient les sauvegardes (`/app/data/backups`) et les fichiers téléversés à restaurer (`/app/data/imports`).

## Variables d'environnement {#environment-variables}

| Variable | Obligatoire | Défaut | Description |
|---|---|---|---|
| `JWT_SECRET` | **oui** | – | Secret qui signe les sessions du tableau de bord. |
| `POSTGRES_IP` | non | `localhost` | Hôte PostgreSQL. |
| `POSTGRES_PORT` | non | `5432` | Port PostgreSQL. |
| `POSTGRES_USER` | non | `postgres` | Utilisateur PostgreSQL. |
| `POSTGRES_PASSWORD` | non | vide | Mot de passe PostgreSQL. |
| `POSTGRES_DB` | non | `jellystics` | Nom de la base de données. |
| `DATABASE_URL` | non | – | Chaîne de connexion PostgreSQL complète. Prend le pas sur toutes les variables `POSTGRES_*`. |
| `PORT` | non | `3000` | Port HTTP. |
| `TZ` | non | UTC | Fuseau horaire des statistiques et du planificateur, par ex. `Europe/Paris`. |
| `JF_HOST` | non | – | URL de Jellyfin, par ex. `http://jellyfin:8096`. Utilisée uniquement tant qu'aucun serveur n'est configuré : un serveur enregistré dans l'assistant ou dans les réglages l'emporte toujours. |
| `JF_API_KEY` | non | – | Clé d'API Jellyfin, même règle que `JF_HOST` (renseignée seulement si aucune clé n'est stockée et que l'hôte stocké est `JF_HOST`). |
| `JELLYSTICS_PUBLIC_URL` | non | détectée automatiquement | URL de Jellystics **telle que la voit le serveur Jellyfin**. Transmise au plugin pour la page « Mes statistiques ». Détectée automatiquement sous la forme `http://<IP locale utilisée pour joindre Jellyfin>:<PORT>`. |
| `DISABLE_DASHBOARD` | non | `false` | `true` ne sert que l'API et `/metrics` (pas de tableau de bord, pas de « Mes statistiques »). |
| `METRICS_API_KEY` | non | – | Ajoute cette valeur comme clé d'API nommée `metrics` à chaque démarrage. Nécessaire avec `DISABLE_DASHBOARD=true`, puisqu'aucune interface ne permet alors de créer des clés. |
| `JS_GEOLITE_ACCOUNT_ID` | non | – | Identifiant de compte MaxMind GeoLite, pour la géolocalisation IP des lectures à distance. |
| `JS_GEOLITE_LICENSE_KEY` | non | – | Clé de licence MaxMind GeoLite. Sans les deux, les lectures sont classées locales/à distance mais pas géolocalisées. |

### Secrets Docker {#docker-secrets}

Dans l'image Docker, chaque variable peut être lue depuis un fichier grâce au préfixe `FILE__` :

```yaml
environment:
  FILE__JWT_SECRET: /run/secrets/jwt_secret
  FILE__POSTGRES_PASSWORD: /run/secrets/db_password
```

Un fichier manquant arrête le conteneur avec une erreur. Si `X` et `FILE__X` sont tous deux définis, le fichier l'emporte (un avertissement est écrit dans les logs).

### Mode métriques seules {#metrics-only-mode}

Pour utiliser Jellystics uniquement comme exporteur Prometheus :

```yaml
environment:
  DISABLE_DASHBOARD: "true"
  METRICS_API_KEY: "a-long-random-key"
  JF_HOST: http://jellyfin:8096
  JF_API_KEY: your-jellyfin-api-key
```

## Premier démarrage {#first-start}

1. Ouvrez Jellystics. L'étape **Connecter Jellyfin** demande l'URL de Jellyfin (par exemple `http://192.168.1.10:8096`) et une clé d'API (dans Jellyfin : *Tableau de bord → Clés API → +*).
2. L'étape **Se connecter** demande un nom d'utilisateur et un mot de passe Jellyfin. Seuls les **administrateurs Jellyfin** peuvent utiliser le tableau de bord ; les autres comptes obtiennent « Accès administrateur requis ». Les utilisateurs ordinaires disposent à la place de leur propre page [« Mes statistiques »](/fr/plugin/my-statistics) dans Jellyfin.
3. Jellystics lance une première synchronisation complète des bibliothèques et des utilisateurs. Les lectures sont enregistrées immédiatement.

::: tip Planifiez une fois la synchronisation complète
Sur une installation toute neuve, ouvrez *Réglages → Tâches* et enregistrez une fois la planification de **Full Jellyfin Sync** (le cron par défaut `0 0 * * *` convient). Tant qu'aucun réglage de tâche n'a été enregistré, la synchronisation complète nocturne n'est pas planifiée.
:::

Fonctionnement de la connexion :

- Jellystics vous authentifie auprès de Jellyfin, puis conserve une session liée à votre jeton Jellyfin.
- Toutes les 5 minutes, la session est vérifiée auprès de Jellyfin : si le jeton est révoqué ou si vous n'êtes plus administrateur, la session prend fin.
- Les sessions inutilisées pendant 30 jours sont révoquées. *Se déconnecter* révoque aussi le jeton Jellyfin.

## Reverse proxy {#reverse-proxy}

Jellystics écoute sur le port `3000`. La page En direct utilise un WebSocket (Socket.IO) : transmettez donc les en-têtes `Upgrade` et `Connection`. Avec Nginx :

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

Caddy transmet les WebSockets tout seul :

```txt
stats.example.com {
    reverse_proxy localhost:3000
}
```

## Mise à jour {#updating}

```bash
docker compose pull
docker compose up -d
```

Les migrations de la base de données s'exécutent automatiquement au démarrage. *Réglages → À propos* vous indique quand une nouvelle version est disponible.

## Et ensuite {#next}

- [Configurer Jellystics](/fr/guide/configuration)
- [Installer le plugin](/fr/plugin/)
