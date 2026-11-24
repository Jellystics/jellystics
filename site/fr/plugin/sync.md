# Synchronisation avec Jellystics

Jellystics **récupère** les données du plugin ; le plugin n'appelle jamais Jellystics, sauf pour vérifier la liaison « Mes statistiques » et relayer cette page.

## Détection {#detection}

Toutes les 30 secondes, Jellystics appelle `GET /Jellystics/Info` sur le serveur Jellyfin avec sa clé d'API Jellyfin.

- **404** : le plugin n'est pas installé. Jellystics continue d'utiliser son suivi.
- **schemaVersion** différent de celui que prend en charge Jellystics (1) : le plugin est ignoré et une erreur s'affiche dans la carte du plugin.
- Sinon, le plugin est *détecté* et la synchronisation s'exécute.

## Lecture des changements {#reading-changes}

Pour chaque flux (lectures, événements de bibliothèque, événements de données utilisateur, connexions) :

```http
GET /Jellystics/Plays?since=<cursor>&limit=500
```

```json
{ "items": [ … ], "next": 58231, "hasMore": true }
```

Jellystics enregistre `next` après chaque page et continue tant que `hasMore` vaut true (jusqu'à 1000 pages par exécution ; la suite vient à l'exécution suivante).

## Qui enregistre les lectures {#who-records-the-plays}

| État du plugin | Qui écrit les lectures |
|---|---|
| **En bonne santé** (synchronisation réussie il y a moins de 2 min) | Le plugin. Le suivi continue de surveiller les sessions pour la page En direct mais n'écrit pas de lectures. |
| **En mauvaise santé** | Le suivi. Ses lectures sont conservées, sauf si une lecture clôturée du plugin les recouvre déjà. |
| **De retour en bonne santé** | Le plugin. Ses lectures clôturées remplacent les lignes du suivi qu'elles recouvrent. |

Suppression des doublons :

- après chaque page, les lectures du suivi recouvertes par une nouvelle lecture du plugin (même utilisateur, contenu et appareil, à l'intérieur de la lecture du plugin avec 10 s de tolérance) sont supprimées ;
- après chaque exécution, la même vérification porte sur les dernières 24 heures ;
- quand le plugin est en mauvaise santé depuis plus d'une heure, ses lectures restées ouvertes depuis plus d'une heure sont clôturées.

::: info Horloges
La correspondance compare les heures de début entre Jellyfin et Jellystics : gardez les horloges des deux hôtes synchronisées (NTP).
:::

## Réinitialisation et resynchronisation {#reset-and-resync}

- Si le `databaseId` du plugin change ou si son curseur recule (base supprimée ou restaurée), Jellystics réinitialise ses curseurs et relit tout. Les insertions/mises à jour sont idempotentes : rien n'est dupliqué.
- **Resynchroniser depuis le plugin** (*Réglages → Configuration*) fait de même à la demande. Restaurer une sauvegarde Jellystics le fait aussi.

## Envoyer l'historique de Jellystics au plugin {#pushing-jellystics-history-to-the-plugin}

*Sauvegarder l'historique dans le plugin* envoie au plugin les lectures que Jellystics a enregistrées lui-même (suivi, imports Playback Reporting, imports TSV, sauvegardes restaurées) :

- films, épisodes et pistes ayant un utilisateur, un contenu et une date de début ;
- dans l'ordre, par lots de **500** (`POST /Jellystics/Import/Plays`) ;
- le plugin les insère avec la source `import-jellystics` et ignore les identifiants qu'il possède déjà : relancer l'opération est sans risque ;
- la progression est enregistrée après chaque lot : après une erreur ou un redémarrage, *Reprendre la sauvegarde* repart là où elle s'était arrêtée.

Ces lectures reviennent dans Jellystics par la synchronisation normale et sont rapprochées des lignes existantes.

## Liaison « Mes statistiques » {#my-statistics-link}

Après chaque synchronisation réussie, Jellystics s'assure que le plugin connaît son adresse et son secret (`POST /Jellystics/Link`) :

- **adresse** : `JELLYSTICS_PUBLIC_URL`, ou `http://<IP locale utilisée par Jellystics pour joindre Jellyfin>:<PORT>` ;
- **secret** : 32 octets aléatoires générés une seule fois et conservés par Jellystics ;
- il se relie à nouveau quand le plugin n'est pas relié, quand l'adresse ou l'empreinte du secret diffère, ou quand le plugin signale un secret rejeté. En cas d'échec, les tentatives s'espacent de 30 s jusqu'à 30 min.

Voir [la page « Mes statistiques »](/fr/plugin/my-statistics).
