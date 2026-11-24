# Bibliothèques et médias

## Bibliothèques {#libraries}

![Bibliothèques](/screenshots/fr/libraries.jpg){.shot}

Une carte par bibliothèque Jellyfin (films, séries, musique) avec sa taille, son temps de lecture, ses vues et ses spectateurs.

## Une bibliothèque {#a-library}

![Contenus d'une bibliothèque](/screenshots/fr/library.jpg){.shot}

- KPI : temps de lecture, vues (avec les spectateurs), **jamais vus** (part du catalogue), complétion moyenne.
- Onglet **Contenus** : recherche, tableau ou grille d'affiches, tri par nom, vues, spectateurs ou temps de lecture, avec une sparkline sur 14 jours. Les bibliothèques de séries ajoutent *Rétention S1É1 → fin*.
- Onglet **Activité** : l'historique des lectures de la bibliothèque.

L'onglet **Statistiques** :

![Statistiques d'une bibliothèque](/screenshots/fr/library-stats.jpg){.shot}

- Genres, méthode de lecture, **temps jusqu'au visionnage** (jours entre l'ajout et la première vue).
- **Délai ajout → premier visionnage** (plugin) : histogramme de < 1 h à > 4 semaines.
- **Décennies regardées vs catalogue**, **croissance vs consommation** (ajouts, premiers visionnages et, avec le plugin, suppressions), et la liste des **jamais vus**.

## Film {#movie}

![Film](/screenshots/fr/movie.jpg){.shot}

- Vues (et clics), spectateurs, **vues terminées**, part transcodée.
- **Entonnoir de complétion** : clics → vues (≥ 30 s) → moitié atteinte → terminé (≥ 90 %).
- **Où les gens s'arrêtent** : histogramme des positions d'arrêt.
- **Séances pour finir le film** : une fois, deux fois, trois fois, quatre fois ou plus.
- **Marqué vu** (plugin) : à la main, en fin de lecture ou autre, par utilisateur.
- Onglets : spectateurs et historique.

## Séries et épisodes {#series-and-episodes}

![Série](/screenshots/fr/series.jpg){.shot}

- Temps de lecture, vues, clics (part des clics sans vue), **rétention** (S1É1 → dernier épisode), **binges**.
- **Rétention par épisode** : spectateurs uniques de chaque épisode ; cliquez pour l'ouvrir.
- **Sauts en avant** (plugin) : sauts courts en début de lecture (probablement le générique), lectures avec un saut, temps sauté moyen.
- Onglet **Épisodes** par saison : vues, spectateurs, complétion moyenne, **abandons**, génériques passés.
- **Cadence entre épisodes** et **délai sortie → visionnage**.

![Épisode](/screenshots/fr/episode.jpg){.shot}

Une page d'épisode comporte des boutons précédent/suivant, un entonnoir, les **positions d'arrêt** (le pic final correspond au générique de fin), les pauses par vue, les sauts et les marques.

## Artistes et albums {#artists-and-albums}

![Artiste](/screenshots/fr/artist.jpg){.shot}

- Écoutes, temps d'écoute, auditeurs, **taux de skip** (clics sans écoute de 30 s).
- **Top pistes** avec les skips par piste, vignettes des albums, et **skip selon l'heure**.

![Album avec le panneau de piste](/screenshots/fr/album.jpg){.shot}

Un album affiche ses pistes, ses **écoutes complètes** (toutes les pistes le même jour) et, quand vous cliquez sur une piste, un panneau latéral avec ses écoutes, ses skips, ses auditeurs et ses dernières écoutes.
