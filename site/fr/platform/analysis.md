# Analyses, découverte et lecture

## Analyses {#insights}

![Analyses](/screenshots/fr/insights.jpg){.shot}

Comment les gens regardent :

- **Distribution de complétion** : vues par tranches de 10 % ; le pic à 100 % correspond aux vues terminées. **Vues terminées par type**.
- **Binges** par série, **pauses** (temps de pause ÷ temps regardé par titre), heatmap du **rythme hebdomadaire**.
- **Durée des séances** (médiane, faux départs exclus), **reprises vs depuis le début**, **revisionnages**.
- **Génériques les plus passés** (plugin), **genres dans le temps**, **moments de la journée par type**.

## Découverte {#discover}

![Découverte](/screenshots/fr/discover.jpg){.shot}

Ce qui dort dans les bibliothèques :

- **Jamais vus** : nombre, part du catalogue et taille sur disque, ajouts les plus anciens en premier.
- **Abandonnés** : titres arrêtés entre 10 et 80 % et non lus depuis plus de 60 jours.
- **Ajoutés sur la période**, et combien ont déjà été regardés.
- **Stockage vs visionnage** : taille des fichiers face aux vues ; les gros fichiers jamais regardés ressortent en rouge.

## Lecture {#playback}

![Lecture et transcodage](/screenshots/fr/playback.jpg){.shot}

Santé de la lecture :

- Parts de **direct play** et de **transcodage**, **cause n°1** de transcodage, **client le plus transcodé**.
- **Méthode de lecture par jour**, **clients** (% de transcodage), **faux départs par client**.
- **Résolution réellement servie** (4K, 1080p, 720p, SD) et fichiers 4K servis en dessous de la 4K ; **HDR** (natif, tone-mappé, SDR).
- **Pourquoi ça transcode** : les raisons fournies par Jellyfin (conteneur, codec vidéo/audio/sous-titres, débit, résolution, profondeur de bits, plage HDR…).
- **Vidéo ou audio seul**, accélération matérielle, **conversions de codec**.
- **Local ou à distance** avec les lieux (géolocalisation IP), distribution du **débit** (moyenne et 95e percentile).

Les cartes qui dépendent d'une capture détaillée (raisons, réseau, débit) indiquent « Données collectées à partir du… » lorsque les données commencent au cours de la période.
