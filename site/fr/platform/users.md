# Utilisateurs et récaps

## Utilisateurs {#users}

![Utilisateurs](/screenshots/fr/users.jpg){.shot}

- Tableau des utilisateurs : temps de lecture, vues, répartition par type, dernière activité et une sparkline sur 14 jours.
- **Rétention des utilisateurs** : cohortes hebdomadaires, et combien reviennent les semaines suivantes.
- **Part de chacun dans le temps** : la part de chaque utilisateur dans le temps de lecture, semaine après semaine.

## Un utilisateur {#a-user}

![Page utilisateur](/screenshots/fr/user.jpg){.shot}

L'onglet **Aperçu** :

- Temps de lecture, vues, **jours actifs**, **série en cours** (et record), complétion moyenne.
- **Activité sur 12 mois** (heatmap calendaire), **En cours** (séries commencées, avec l'épisode suivant), **Horloge** (vues par heure).
- Scores du **Profil** : régularité, découverte, concentration, vues terminées.
- **Série en cours & records**, **Appareils & versions** (les versions d'application obsolètes sont signalées), **VO ou VF** (langues audio et sous-titres).
- **Connexions** (plugin) : nombre, dernière connexion et tentatives échouées par appareil.
- **Comparaison** avec un autre utilisateur.

L'onglet **Historique** est la page d'historique filtrée sur cet utilisateur. L'onglet **Goûts** affiche les genres favoris pour les séries, les films et la musique, côte à côte avec un autre utilisateur si vous en choisissez un :

![Comparaison des goûts](/screenshots/fr/user-taste.jpg){.shot}

## Récap {#recap}

![Récap annuel](/screenshots/fr/recap.jpg){.shot}

`/users/<id>/recap/2026` (une année) ou `/users/<id>/recap/2026-11` (un mois) :

- total d'heures, équivalent en jours, vues et jours actifs ;
- top séries, films et artistes ;
- heures par mois ou par jour, meilleur mois ou jour, **binge record**, **plus longue série** ;
- artistes découverts, et un **classement avec évolution** par rapport à la période précédente.

## Mes statistiques (pour chaque utilisateur Jellyfin) {#my-statistics-for-every-jellyfin-user}

L'accès au tableau de bord est réservé aux administrateurs Jellyfin. Tous les autres utilisateurs disposent de leur propre page dans Jellyfin grâce au plugin :

![Mes statistiques](/screenshots/fr/me.jpg){.shot}

Voir [la page « Mes statistiques »](/fr/plugin/my-statistics).
