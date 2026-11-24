# Vue d'ensemble, direct et historique

## Navigation {#navigation}

- La barre latérale regroupe les pages en **Observer** (Vue d'ensemble, En direct, Historique), **Explorer** (Bibliothèques, Utilisateurs) et **Comprendre** (Analyses, Découverte, Lecture). En direct affiche un point rouge quand une lecture est en cours.
- Le **sélecteur de période** (barre du haut) propose Aujourd'hui, 7/30/90 derniers jours, Mois en cours, Année en cours, Tout et une période personnalisée. *Comparer à la période précédente* ajoute des écarts à chaque KPI (% relatif pour les comptages, points pour les ratios).
- La période, les filtres, les onglets et le tri sont conservés dans l'URL : chaque vue peut être mise en favori ou partagée.
- **`⌘K` / `Ctrl+K`** ouvre la palette de commandes : recherchez des films, séries, épisodes, albums, artistes, pistes et utilisateurs, allez à n'importe quelle page, lancez une synchronisation ou basculez le thème.
- Le **menu utilisateur** regroupe le thème, la langue, les réglages et *Se déconnecter*.
- Chaque graphique dispose d'un bouton **CSV** qui télécharge ses données.

## Vue d'ensemble {#overview}

![Vue d'ensemble](/screenshots/fr/overview.jpg){.shot}

- **KPI** : Temps de lecture, Vues, Clics (chacun avec une sparkline ; cliquez sur l'un d'eux pour choisir ce qu'affiche le graphique principal), Utilisateurs actifs, Complétion moyenne.
- **{Métrique} par jour** : la métrique sélectionnée sur la période ; avec la comparaison, la période précédente est tracée aussi.
- **Toutes les lectures** : un point par lecture (heure de début × minutes regardées, coloré par type) ; cliquez sur un point pour ouvrir le titre.
- **Lectures par bibliothèque** : vues par jour (ou par semaine) et par bibliothèque.
- **Top contenus** (tout, films, séries, musique), **En direct** (deux sessions en cours), **Utilisateurs** (top 8 par temps de lecture).
- **Par type** : part du temps de lecture par type, puis méthode de lecture.
- **Quand** : heatmap jour de la semaine × heure. **Pic de flux simultanés** par jour. **Ce jour-là** : ce qui a été regardé à cette date les années précédentes.

Avec *Comparer à la période précédente* :

![Vue d'ensemble avec comparaison](/screenshots/fr/overview-compare.jpg){.shot}

## En direct {#live}

![En direct](/screenshots/fr/live.jpg){.shot}

Mis à jour en temps réel par WebSocket.

- **Lectures en cours**, **Transcodages** (avec le nombre accéléré matériellement), **Débit sortant** (somme des flux actifs), **Pic du jour**.
- Une carte par session : visuel, titre, utilisateur, appareil, méthode de lecture, badge de pause et progression.
- **Débit sortant en direct** par flux, **Lectures simultanées** du jour par tranches de 15 minutes, et **Frise de la journée** : qui a regardé quoi, et quand les flux se sont chevauchés.

## Historique {#history}

![Tableau de l'historique](/screenshots/fr/history.jpg){.shot}

- **Filtres** : texte libre, type (films, séries, musique), *Vues seulement*, utilisateur, client, appareil, méthode de lecture. Cliquez sur une cellule client ou appareil pour filtrer dessus.
- **Tableau** : date, utilisateur, contenu, temps regardé, complétion, appareil, méthode de lecture et badge Vue/Clic. Tri par date, temps regardé ou complétion ; 25 à 200 lignes par page.
- **Exporter** télécharge tout l'historique filtré au format CSV, généré par le serveur.

La vue **Calendrier** affiche chaque jour du mois avec son nombre de vues et les premiers titres :

![Calendrier de l'historique](/screenshots/fr/history-calendar.jpg){.shot}
