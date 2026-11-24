# Réglages du plugin

Ouvrez **Jellyfin → Tableau de bord → Extensions → Jellystics** (ou *Jellystics* dans la section serveur du menu du tableau de bord). La page comporte quatre blocs.

## Statut {#status}

Lu depuis `GET /Jellystics/Info` :

```txt
Version 0.2.1.0
1 840 plays, 3 512 library events, 412 marks, 96 logins
First play 2024-03-02
Sync cursor 58 231
```

Le curseur est le numéro de version global qu'utilise Jellystics pour lire ce qui a changé (voir [Synchronisation](/fr/plugin/sync)).

## Page My statistics {#my-statistics-page}

Chaque utilisateur dispose d'une entrée *Mes statistiques* dans le menu de son avatar. Jellystics se relie automatiquement lors de sa prochaine synchronisation : il transmet son URL et un secret partagé au plugin. Ce bloc affiche le résultat :

| Message | Signification | Que faire |
|---|---|---|
| Not linked | Jellystics n'a pas encore transmis son adresse. | Attendez la prochaine synchronisation (30 s). Vérifiez que la carte du plugin est *Actif* dans Jellystics. |
| Linked to `<url>` · reachable | Tout va bien. | – |
| UNREACHABLE from Jellyfin | Jellyfin n'arrive pas à joindre cette adresse. | Renseignez le **URL override** avec une adresse de Jellystics que le serveur Jellyfin peut joindre, ou définissez `JELLYSTICS_PUBLIC_URL` dans Jellystics. |
| Jellystics rejected the shared secret | Le secret a changé. | Jellystics se relie à nouveau en moins d'une minute. Si le problème persiste, assurez-vous qu'une seule instance de Jellystics cible ce Jellyfin. |

**Jellystics URL override** : laissez vide pour utiliser l'URL transmise. Renseignez-le quand Jellystics est joignable depuis Jellyfin à une autre adresse, par exemple `http://192.168.1.10:3000` ou `http://jellystics:3000` sur un réseau Docker partagé. Ce doit être une URL `http` ou `https` absolue, sans paramètres de requête, fragment ni identifiants.

Le secret n'est stocké que dans la configuration XML du plugin ; il n'est jamais renvoyé par aucune API (seule une empreinte SHA-256 l'est).

## Import history {#import-history}

Importe l'historique du plugin officiel **Playback Reporting**.

- **Path of playback_reporting.db** : laissez vide pour utiliser `<jellyfin data>/playback_reporting.db`, ou indiquez une copie.
- **Import history** enregistre le chemin, puis importe en arrière-plan. La ligne d'état affiche `read / imported / skipped / repaired`.

Ce que fait l'import :

- il ouvre le fichier en **lecture seule**, 500 lignes à la fois ;
- il donne à chaque ligne l'identifiant `pbr-<rowid>` : relancer l'import ne crée jamais de doublon ;
- il fixe le début à l'heure enregistrée − 20 s (Playback Reporting enregistre la fin des 20 premières secondes), lue dans le fuseau horaire du serveur ;
- il répare les durées impossibles : négative → 0 ; plus de 1,5 × la durée → la durée ; plus de 12 h quand la durée est inconnue → 12 h. Les lignes réparées sont signalées ;
- il ignore les lignes qui chevauchent une lecture enregistrée par le plugin lui-même ;
- il complète la série, la saison, l'album, la durée et la bibliothèque à partir de la bibliothèque Jellyfin actuelle quand le contenu existe encore.

## Purge plays before {#purge-plays-before}

Supprime définitivement les lectures **clôturées** qui ont commencé avant la date choisie (minuit dans le fuseau horaire de votre navigateur).

::: warning
La purge est locale au plugin. Elle n'est **pas** répercutée sur Jellystics, et réimporter Playback Reporting restaure les lignes purgées qui en provenaient.
:::
