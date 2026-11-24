---
aside: false
---

# Fonctionnement

Cette page suit une lecture depuis le moment où quelqu'un appuie sur *Lecture* jusqu'à son apparition dans un graphique et dans Prometheus.

<Workflow />

## 1. Enregistrer une lecture {#_1-recording-a-playback}

Il existe deux enregistreurs. Le plugin est utilisé lorsqu'il est en bonne santé ; sinon, le suivi intégré prend le relais.

### Avec le plugin (piloté par les événements) {#with-the-plugin-event-driven}

Le plugin écoute les événements de Jellyfin lui-même (`PlaybackStart`, `PlaybackProgress`, `PlaybackStopped`) et tient une machine à états par lecture. Chaque événement ajoute :

- du **temps regardé** = le plus petit entre le temps réel écoulé et la progression de la position (+ 2 s de marge). Les pauses, la mise en mémoire tampon et les retours en arrière n'ajoutent rien ;
- du **temps de pause** tant que le lecteur est en pause, et une **pause** à chaque passage de lecture → pause ;
- un **saut** quand la position fait un bond de plus de 5 s au-delà de ce qu'explique la lecture, avec le **temps sauté** pour les sauts en avant.

Les lectures ouvertes sont sauvegardées toutes les 15 s ; après un plantage de Jellyfin, une lecture est clôturée à son dernier point de sauvegarde. Tous les détails : [Mesure des lectures](/fr/plugin/tracking).

### Sans le plugin (interrogation) {#without-the-plugin-polling}

Jellystics appelle `/Sessions` de Jellyfin toutes les **5 secondes** et applique la même comptabilité :

- chaque interrogation ajoute la progression vers l'avant de la position, plafonnée au temps écoulé + 2 s ;
- si un client qui se manifeste normalement toutes les ≤ 30 s reste muet plus de 45 s, le temps cesse d'être compté ;
- à l'arrêt de la lecture, le temps écoulé depuis la dernière interrogation est récupéré à partir de la position enregistrée par Jellyfin (55 s au plus) ;
- une nouvelle lecture commence quand le contenu ou l'utilisateur change, ou quand la position revient au début après la fin de la précédente (relecture).

Seuls les films, les épisodes et les pistes musicales sont suivis ; les bandes-annonces, les bonus et les médias de thème sont ignorés.

## 2. Faire entrer les données du plugin dans Jellystics {#_2-getting-plugin-data-into-jellystics}

Toutes les **30 secondes**, Jellystics demande au plugin ce qui a changé depuis la dernière fois, avec un curseur par flux (lectures, événements de bibliothèque, événements de données utilisateur, connexions), à raison de 500 lignes par page. Un curseur est un numéro de version global : chaque insertion ou mise à jour dans le plugin en reçoit un nouveau, si bien que les lectures en cours sont renvoyées au fur et à mesure qu'elles s'allongent.

- **En bonne santé** = le plugin a répondu avec succès il y a moins de **2 minutes**.
- Tant que le plugin est en bonne santé, le suivi n'écrit pas de lectures : c'est le plugin qui en a la charge.
- Quand une lecture du plugin arrive, toute ligne du suivi pour le même utilisateur, contenu et appareil qu'elle recouvre est supprimée (10 s de tolérance), et le même nettoyage est appliqué aux dernières 24 heures après chaque synchronisation.
- Quand le plugin est hors service, les lectures du suivi sont conservées, sauf si une lecture clôturée du plugin les recouvre déjà. Quand le plugin revient, sa version clôturée remplace la ligne du suivi.
- Si la base du plugin a été recréée (nouvel identifiant de base, ou curseur qui recule), Jellystics réinitialise ses curseurs et relit tout ; les insertions/mises à jour sont idempotentes.

Détails : [Synchronisation avec Jellystics](/fr/plugin/sync).

## 3. Des lectures aux statistiques {#_3-from-plays-to-statistics}

Toutes les lectures, quelle que soit leur source, aboutissent dans une même table. Les statistiques sont calculées en SQL au moment de la requête, dans le fuseau horaire `TZ` :

| Notion | Règle |
|---|---|
| Clic | Toute lecture enregistrée. |
| Vue | Une lecture avec **≥ 30 s** réellement regardées. |
| Temps de lecture | Somme du temps regardé de **toutes** les lectures (clics compris). |
| Complétion | `max(stop position, watched time) / runtime`, plafonnée à 100 %, par utilisateur, titre et jour local. |
| Vue terminée | Complétion ≥ **90 %**, ou, pour les films et les épisodes de 5 min ou plus enregistrés par le plugin, l'indicateur « lu jusqu'à la fin » de Jellyfin. |
| Binge | ≥ **3 épisodes distincts** d'une série par un utilisateur sur un même jour local. |

La liste complète se trouve dans le [glossaire](/fr/guide/glossary).

## 4. Sources et précision {#_4-sources-and-precision}

Chaque lecture garde sa source, visible dans *Réglages → Configuration → Qualité des données* :

| Source | Précision |
|---|---|
| Plugin Jellystics | Événements exacts, pauses, sauts, pistes, transcodage. |
| Suivi Jellystics | Interrogation toutes les 5 s. |
| Imports Playback Reporting | Heure de début estimée (heure enregistrée − 20 s), pas de pauses ; durées impossibles réparées (négative → 0, > 1,5 × la durée → la durée, > 12 h avec une durée inconnue → 12 h). |
| Autres imports, sauvegardes restaurées | Tels qu'enregistrés par l'outil d'origine. |

Les imports ne créent jamais de doublons : une lecture déjà enregistrée par une autre source (même utilisateur et même contenu, début à moins de 2 min, durée similaire) est ignorée.

## 5. Le reste des données {#_5-the-rest-of-the-data}

- **Bibliothèques, contenus, utilisateurs** proviennent de l'API Jellyfin (synchronisation complète, synchronisation des ajouts récents, et à la demande quand un nouveau contenu est lu).
- **Les événements de bibliothèque, les connexions et les marques vu/favori** proviennent uniquement du plugin.
- **À distance ou local** : d'après l'IP du client ; les adresses privées, de bouclage et link-local sont locales. Les IP distantes peuvent être géolocalisées avec MaxMind GeoLite (facultatif).
- **Les sessions en direct** sont poussées vers le tableau de bord par WebSocket toutes les 5 s.

## 6. Exposer les données {#_6-exposing-the-data}

- Le **tableau de bord** lit `/api/v2/*`.
- **Prometheus** lit [`/metrics`](/fr/prometheus/metrics), calculé en direct à chaque collecte.
- **Les utilisateurs Jellyfin** consultent leurs propres chiffres via le plugin : [« Mes statistiques »](/fr/plugin/my-statistics).
