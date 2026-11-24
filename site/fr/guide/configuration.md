# Configuration

Jellystics se configure à deux endroits : les **variables d'environnement** au démarrage (voir [Installation](/fr/guide/installation#environment-variables)) et les **Réglages** du tableau de bord (menu utilisateur → *Réglages*). Cette page détaille chaque réglage ; la [visite des réglages](/fr/platform/settings) montre les écrans.

## Serveur Jellyfin {#jellyfin-server}

*Réglages → Configuration → Serveur Jellyfin*

| Champ | Description |
|---|---|
| URL Jellyfin | Doit commencer par `http://` ou `https://`. C'est l'adresse qu'utilise Jellystics pour lire les bibliothèques, les utilisateurs et les sessions. |
| Clé d'API | Laissez vide pour conserver la clé enregistrée. Requise pour tester la connexion. Créez-en une dans Jellyfin : *Tableau de bord → Clés API*. |
| URL publique de Jellystics | Adresse publique de Jellystics. Sert d'avatar par défaut du bot (`<url>/logo.png`) dans les notifications Discord. Ce n'est **pas** l'adresse transmise au plugin (voir `JELLYSTICS_PUBLIC_URL`). |

*Tester la connexion* vérifie l'URL et la clé, et répond « Connecté à *nom du serveur* ».

## Carte du plugin Jellystics {#jellystics-plugin-card}

*Réglages → Configuration → Plugin Jellystics* affiche l'état du [plugin](/fr/plugin/) :

- **Actif** : le plugin a répondu il y a moins de 2 minutes. **Ne synchronise plus** : aucune synchronisation réussie depuis plus de 2 minutes (le suivi intégré enregistre les lectures en attendant). **Non détecté** : installez-le avec l'URL du manifeste affichée dans la carte.
- Version, schéma d'API, dernière synchro, lectures synchronisées, dernière erreur.
- **Page « Mes statistiques »** : état de la liaison (Active, Jellystics injoignable depuis Jellyfin, Reconnexion en cours, Pas encore reliée) et adresse transmise au plugin.
- **Sauvegarder l'historique dans le plugin** : envoie au plugin les lectures que Jellystics a enregistrées lui-même (suivi, imports, restaurations), pour que le plugin détienne l'historique complet. Peut être relancé sans risque ; reprend après une erreur ou un redémarrage.
- **Resynchroniser depuis le plugin** : relit tout depuis le plugin. Les lectures enregistrées en double sont remplacées par la version du plugin.

Détails : [Synchronisation avec Jellystics](/fr/plugin/sync).

## Carte Qualité des données {#data-quality-card}

Indique d'où viennent les lectures d'une période (plugin, suivi, imports Playback Reporting, autres imports, sauvegardes restaurées) et quels détails chaque source a enregistrés (position d'arrêt, piste audio/sous-titres, détails de lecture, détails du transcodage). Elle compte aussi les durées importées qui ont été réparées et les lectures sans date exploitable (exclues de toutes les statistiques).

## Tâches planifiées {#scheduled-tasks}

*Réglages → Tâches*. Les expressions cron comportent 5 champs (`minute hour day month weekday`) et suivent `TZ`.

| Tâche | Planification par défaut | Ce qu'elle fait |
|---|---|---|
| Full Jellyfin Sync | `0 0 * * *` (toujours active) | Synchronise toutes les bibliothèques, contenus, saisons, épisodes, pistes et utilisateurs. Déclenche aussi la géolocalisation IP. |
| Recently Added Sync | `0 * * * *` (désactivée par défaut) | Synchronise les 100 derniers contenus de chaque bibliothèque. |
| Backup | `0 3 * * *` (désactivée par défaut) | Crée une sauvegarde dans `/app/data/backups`. |
| Jellyfin Playback Reporting Plugin Sync | `0 4 * * *` (désactivée par défaut) | Importe l'historique du plugin officiel Playback Reporting. Ignorée quand le plugin Jellystics est détecté (le plugin l'importe lui-même). |

Tâches de fond non listées : la synchronisation du plugin (toutes les 30 s), le suivi (toutes les 5 s), le nettoyage des sessions (quotidien) et la géolocalisation (2 min après le démarrage, puis quotidienne).

::: warning
Une seule tâche s'exécute à la fois : une tâche lancée pendant qu'une autre tourne est ignorée. Sur une nouvelle installation, enregistrez une fois la planification de Full Jellyfin Sync pour qu'elle soit programmée.
:::

Le même onglet importe un **export TSV/TXT de Playback Reporting** (le fichier exporté par le plugin officiel) : chaque ligne devient une lecture, les durées négatives sont réparées et les lectures déjà enregistrées par une autre source sont écartées.

## Sauvegardes {#backups}

*Réglages → Sauvegardes*

- **Créer une sauvegarde** écrit `backup_YYYY-MM-DD HH-mm-ss.json` dans `/app/data/backups`. Les 5 sauvegardes les plus récentes sont conservées.
- Une sauvegarde contient : bibliothèques, contenus, saisons, épisodes, utilisateurs, toutes les lectures et les lignes Playback Reporting pas encore importées. Elle ne contient **pas** les réglages, les clés d'API, les webhooks ni les pistes musicales : ceux-ci sont recréés par une synchronisation ou doivent être redéfinis.
- **Restaurer** insère ou met à jour la sauvegarde dans la base (les lectures existantes sont mises à jour, celles déjà enregistrées par une autre source sont ignorées), puis réinitialise les curseurs du plugin. Les sauvegardes Jellystat sont aussi acceptées.
- **Téléverser une sauvegarde** accepte les fichiers `.json`.

## Webhooks {#webhooks}

*Réglages → Webhooks* envoie des notifications **Discord**.

| Champ | Description |
|---|---|
| Nom | Texte libre. |
| URL du webhook Discord | `https://discord.com/api/webhooks/…` (Discord : *paramètres du salon → Intégrations → Webhooks*). |
| Nom du bot | Par défaut `jellystics_bot`. |
| URL de l'avatar du bot | Facultatif ; par défaut, le logo de l'URL publique de Jellystics. |
| Notifications | Au moins une parmi : tâche démarrée, tâche terminée, tâche en échec, sauvegarde terminée, clé API créée, clé API supprimée. |

Chaque webhook peut être activé/désactivé, modifié, supprimé ou testé. L'envoi a un délai d'expiration de 10 s et n'est pas réessayé.

## Clés d'API {#api-keys}

*Réglages → Clés API*. Une clé n'est affichée en entier qu'une seule fois, à sa création. Les clés authentifient [`/metrics`](/fr/prometheus/) avec `Authorization: Bearer <key>`. La création ou la suppression d'une clé peut déclencher un webhook.

## Journaux {#logs}

*Réglages → Journaux* liste les 50 dernières exécutions de tâches (niveau, date, tâche, message), avec une recherche et un filtre par niveau. *Réglages → Tâches* diffuse aussi en direct la sortie de la tâche en cours.

## À propos {#about}

Version installée et dernière version, lien vers la release, « Rechercher des mises à jour ».

## Langue et thème du tableau de bord {#dashboard-language-and-theme}

- **Langue** : menu utilisateur → *Langue* (English, Français, Español). Mémorisée dans le navigateur (`i18nextLng`) ; `?lng=fr` dans l'URL fonctionne aussi.
- **Thème** : menu utilisateur → *Thème clair / Thème sombre*, ou `⌘K` → *Basculer le thème*. Sans choix explicite, il suit le système.
