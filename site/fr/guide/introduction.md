# Introduction

Jellystics est un outil de statistiques pour [Jellyfin](https://jellyfin.org). Il répond à des questions comme *qui regarde quoi, quand, sur quel appareil, jusqu'où, et avec quelle qualité de diffusion*, et conserve la réponse pendant des années.

![Page Vue d'ensemble](/screenshots/fr/overview.jpg){.shot}

## Les deux composants {#the-two-components}

| Composant | Où il tourne | Rôle |
|---|---|---|
| **Jellystics** | Conteneur Docker (API Go, tableau de bord React) + PostgreSQL | Stocke l'historique, calcule les statistiques, sert le tableau de bord, l'API, `/metrics`, les webhooks et les sauvegardes. |
| **Plugin Jellystics** | Dans Jellyfin 10.11 | Enregistre chaque lecture à partir des événements du serveur dans sa propre base SQLite, et relaie la page « Mes statistiques » à chaque utilisateur Jellyfin. |

Jellystics fonctionne seul. Le plugin apporte davantage de précision et une sauvegarde :

| | Sans le plugin | Avec le plugin |
|---|---|---|
| Source | Jellystics interroge `/Sessions` de Jellyfin toutes les 5 s | Jellyfin envoie les événements au plugin en temps réel |
| Temps de lecture | Très bon, limité par l'interrogation toutes les 5 s | Exact, issu des événements de lecture |
| Redémarrage de Jellyfin | La lecture en cours est clôturée quand elle disparaît | Au plus 15 s perdues (points de sauvegarde) |
| Sauts en avant / génériques passés | – | Comptés, avec le temps sauté |
| « Lu jusqu'à la fin » | Estimé (≥ 90 % de la durée) | Fourni par Jellyfin |
| Ajouts et suppressions dans les bibliothèques | Dates du catalogue (approximatives) | Événements réels |
| Connexions, « marqué comme vu » à la main | – | Enregistrés |
| Sauvegarde de l'historique dans Jellyfin | – | Historique complet dans `jellystics.db` |
| « Mes statistiques » pour les utilisateurs | – | Oui |

Quand le plugin est en bonne santé, il fait foi ; s'il cesse de répondre, Jellystics repasse automatiquement sur son propre suivi, sans trou ni doublon. Voir [le fonctionnement](/fr/guide/how-it-works).

## Ce que vous obtenez {#what-you-get}

- **Tableau de bord** : vue d'ensemble avec comparaison de périodes, sessions en direct, historique (tableau et calendrier), [bibliothèques et pages de médias](/fr/platform/libraries), [utilisateurs et récaps](/fr/platform/users), [analyses, découverte et santé de la lecture](/fr/platform/analysis).
- **Des définitions précises** : clics, vues (≥ 30 s), complétion, binges, reprises, revisionnages. Voir le [glossaire](/fr/guide/glossary).
- **Intégrations** : [métriques Prometheus et Grafana](/fr/prometheus/), webhooks Discord, une API HTTP avec clés d'API.
- **Exploitation** : tâches planifiées, sauvegardes et restauration, import de l'historique du plugin officiel Playback Reporting.
- **Trois langues** : anglais, français et espagnol, dans le tableau de bord comme dans cette documentation.

## Prérequis {#requirements}

- Jellyfin **10.11** (pour le plugin ; Jellystics lui-même n'a besoin que de l'API HTTP de Jellyfin).
- Docker et Docker Compose pour Jellystics.
- Une **clé d'API** Jellyfin et un compte **administrateur** Jellyfin pour se connecter au tableau de bord.

## Étapes suivantes {#next-steps}

1. [Installer Jellystics](/fr/guide/installation).
2. [Installer le plugin](/fr/plugin/).
3. [Brancher Prometheus et Grafana](/fr/prometheus/) si vous les utilisez.
