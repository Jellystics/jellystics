# Glossaire

Définitions utilisées partout dans Jellystics (tableau de bord, « Mes statistiques » et Prometheus). Les jours, heures et jours de la semaine sont exprimés dans le fuseau horaire `TZ`.

| Terme | Définition |
|---|---|
| **Clic** | Une lecture démarrée, quelle que soit sa durée. Un clic qui n'a jamais atteint 30 s porte un badge *Clic*. |
| **Vue** | Une lecture d'au moins **30 secondes** réellement regardées (pauses exclues). |
| **Temps de lecture** | Somme du temps regardé de toutes les lectures, clics compris. |
| **Utilisateur actif** | Un utilisateur ayant au moins une lecture (clic) sur la période. |
| **Complétion** | `max(stop position, watched time) ÷ runtime`, plafonnée à 100 %. Comptée une fois par utilisateur, titre et jour : regarder un épisode en deux fois le même jour compte pour une seule unité. La *complétion moyenne* est la moyenne de ces unités. |
| **Vue terminée** | Complétion ≥ **90 %**. Pour les films et les épisodes de 5 min ou plus enregistrés par le plugin, c'est l'indicateur « lu jusqu'à la fin » de Jellyfin qui décide. Les lectures encore en cours ne comptent jamais. |
| **Abandonné** (complétion) | Une unité de complétion sous 90 % (Prometheus `jellyfin_abandoned_plays_total`, « abandons » sur les épisodes). |
| **Abandonné** (Découverte) | Un titre lu pour la dernière fois il y a plus de **60 jours** et laissé entre **10 % et 80 %** (films), ou dont 10 à 80 % des épisodes ont été vus (séries). |
| **Moitié atteinte** | Étape de l'entonnoir : complétion ≥ 50 %. |
| **Faux départ** | Un clic qui n'est jamais devenu une vue (arrêté avant 30 s). |
| **Taux de skip** (musique) | Clics jamais devenus une écoute ÷ clics. Affiché en rouge au-delà de 30 %. |
| **Binge** | Au moins **3 épisodes distincts** d'une même série, par le même utilisateur, le même jour. |
| **Reprise** | Une vue démarrée à **30 s ou plus** du début du titre. |
| **Revisionnage** | Le même titre vu par la même personne sur au moins deux jours différents (pistes : au moins deux vues). |
| **Rétention** (séries) | Spectateurs du dernier épisode régulier ÷ spectateurs du premier (épisodes spéciaux exclus). |
| **Rétention** (utilisateurs) | Cohortes hebdomadaires : utilisateurs actifs une semaine donnée qui le sont de nouveau 1 à 4 semaines plus tard (8 dernières semaines complètes). |
| **Série en cours** | Jours consécutifs avec au moins une lecture. |
| **Lectures simultanées** | Nombre maximal de lectures en même temps, par tranches de 15 minutes. |
| **Temps jusqu'au visionnage** | Jours entre la date d'ajout d'un contenu et sa première vue. |
| **Saut court en début de lecture** (générique) | Plugin uniquement : un saut en avant de 10 s à 15 % de la durée (3 min au plus) dans une lecture démarrée depuis le début. Il s'agit d'une estimation. |
| **Méthode de lecture** | *Direct play* (fichier envoyé tel quel), *Direct stream* (remuxé, codecs inchangés), *Transcodage* (réencodé). |
| **Local / à distance** | D'après l'IP du client : les adresses privées (RFC 1918, ULA), de bouclage et link-local sont locales. Le CGNAT (100.64.0.0/10, par ex. Tailscale) compte comme à distance. |
| **VO ou VF** | *VO* = la première piste audio du fichier a été lue. |
| **Appareil obsolète** | Une version d'application plus ancienne que la plus récente observée pour le même client. |
| **Scores du profil** | Régularité (jours actifs), Découverte (premières vues), Concentration (part du top 3 des titres). |
