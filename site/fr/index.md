---
layout: home
title: Jellystics
titleTemplate: Statistiques pour Jellyfin

hero:
  name: Jellystics
  text: Des statistiques précises pour votre serveur Jellyfin
  tagline: Chaque lecture enregistrée à la source, l'historique complet conservé, et tout exposé à votre tableau de bord comme à Grafana.
  image:
    src: /logo.png
    alt: Jellystics
  actions:
    - theme: brand
      text: Commencer
      link: /fr/guide/introduction
    - theme: alt
      text: Installer le plugin
      link: /fr/plugin/
    - theme: alt
      text: GitHub
      link: https://github.com/Jellystics/jellystics

features:
  - icon: 🎯
    title: Le vrai temps de lecture
    details: Le plugin Jellyfin enregistre les lectures à partir des événements du serveur. Les pauses, la mise en mémoire tampon et les retours en arrière ne comptent jamais ; un redémarrage fait perdre au plus 15 secondes.
  - icon: 👆
    title: Clics et vues
    details: Chaque démarrage est un clic, une vue exige 30 secondes réellement regardées. Complétion, binges, reprises et revisionnages sont calculés par utilisateur et par jour.
  - icon: 📊
    title: Un tableau de bord complet
    details: Vue d'ensemble avec comparaisons, sessions en direct, historique en tableau et en calendrier, bibliothèques, films, séries, artistes, albums, utilisateurs et récaps annuels.
  - icon: 🔎
    title: Analyses et santé de la lecture
    details: Distribution de la complétion, rythme hebdomadaire, titres abandonnés, stockage jamais regardé, raisons de transcodage, résolution servie, HDR et débit.
  - icon: 📈
    title: Prometheus et Grafana
    details: Un endpoint /metrics protégé par une clé d'API avec 21 familles de métriques, et un tableau de bord Grafana prêt à l'emploi de 31 panneaux.
  - icon: 🙋
    title: Des statistiques pour chaque utilisateur
    details: Chaque utilisateur Jellyfin dispose d'une page « Mes statistiques » dans le menu de son avatar, servie sur l'origine Jellyfin et limitée à ses propres données.
---

<script setup>
import { withBase } from 'vitepress'
</script>

<div class="home-section">

## Aperçu {#see-it}

<div class="home-gallery">
  <figure class="wide">
    <img :src="withBase('/screenshots/fr/overview.jpg')" alt="Page Vue d'ensemble">
    <figcaption>Vue d'ensemble : temps de lecture, vues, clics, toutes les lectures et lectures par bibliothèque.</figcaption>
  </figure>
  <figure>
    <img :src="withBase('/screenshots/fr/live.jpg')" alt="Page En direct">
    <figcaption>Sessions en direct, débit sortant et flux simultanés.</figcaption>
  </figure>
  <figure>
    <img :src="withBase('/screenshots/fr/series.jpg')" alt="Page d'une série">
    <figcaption>Une série : rétention par épisode, binges et sauts en avant.</figcaption>
  </figure>
  <figure>
    <img :src="withBase('/screenshots/fr/insights.jpg')" alt="Page Analyses">
    <figcaption>Analyses : complétion, binges, pauses et rythme hebdomadaire.</figcaption>
  </figure>
  <figure>
    <img :src="withBase('/screenshots/fr/playback.jpg')" alt="Page Lecture">
    <figcaption>Santé de la lecture : direct play, transcodages et leurs raisons.</figcaption>
  </figure>
  <figure>
    <img :src="withBase('/screenshots/fr/recap.jpg')" alt="Page Récap">
    <figcaption>Récap annuel et mensuel pour chaque utilisateur.</figcaption>
  </figure>
  <figure>
    <img :src="withBase('/screenshots/fr/me.jpg')" alt="Page Mes statistiques">
    <figcaption>« Mes statistiques », dans Jellyfin, pour chaque utilisateur.</figcaption>
  </figure>
</div>

## Comment les briques s'assemblent {#how-the-pieces-fit}

<Workflow />

Lisez [le fonctionnement](/fr/guide/how-it-works) pour les détails.

## Démarrer en deux minutes {#start-in-two-minutes}

1. Copiez le `docker-compose.yaml` du [guide d'installation](/fr/guide/installation) et définissez un `JWT_SECRET` long et aléatoire.
2. Lancez `docker compose up -d`.
3. Ouvrez `http://localhost:3000`, connectez votre serveur Jellyfin et identifiez-vous avec un compte administrateur Jellyfin.
4. [Installez le plugin](/fr/plugin/) dans Jellyfin pour des données plus précises.

</div>
