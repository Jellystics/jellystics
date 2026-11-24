---
layout: home
title: Jellystics
titleTemplate: Statistics for Jellyfin

hero:
  name: Jellystics
  text: Precise statistics for your Jellyfin server
  tagline: Every playback recorded at the source, the full history kept, and everything exposed to your dashboard and to Grafana.
  image:
    src: /logo.png
    alt: Jellystics
  actions:
    - theme: brand
      text: Get started
      link: /guide/introduction
    - theme: alt
      text: Install the plugin
      link: /plugin/
    - theme: alt
      text: GitHub
      link: https://github.com/Jellystics/jellystics

features:
  - icon: 🎯
    title: Real watch time
    details: The Jellyfin plugin records playbacks from server events. Pauses, buffering and backward seeks never count; a restart loses at most 15 seconds.
  - icon: 👆
    title: Clicks and views
    details: Every start is a click, a view needs 30 seconds really watched. Completion, binges, resumes and rewatches are computed per user and per day.
  - icon: 📊
    title: A complete dashboard
    details: Overview with comparisons, live sessions, history table and calendar, libraries, movies, series, artists, albums, users and yearly recaps.
  - icon: 🔎
    title: Insights and playback health
    details: Completion distribution, weekly rhythm, abandoned titles, storage never watched, transcode reasons, resolution served, HDR and bitrate.
  - icon: 📈
    title: Prometheus and Grafana
    details: A /metrics endpoint protected by an API key with 21 metric families, and a ready-made 31-panel Grafana dashboard.
  - icon: 🙋
    title: Statistics for every user
    details: Each Jellyfin user gets a “My statistics” page in their avatar menu, served on the Jellyfin origin and showing only their own data.
---

<script setup>
import { withBase } from 'vitepress'
</script>

<div class="home-section">

## See it

<div class="home-gallery">
  <figure class="wide">
    <img :src="withBase('/screenshots/en/overview.jpg')" alt="Overview page">
    <figcaption>Overview: watch time, views, clicks, every play and plays per library.</figcaption>
  </figure>
  <figure>
    <img :src="withBase('/screenshots/en/live.jpg')" alt="Live page">
    <figcaption>Live sessions, outgoing bitrate and concurrent streams.</figcaption>
  </figure>
  <figure>
    <img :src="withBase('/screenshots/en/series.jpg')" alt="Series page">
    <figcaption>A series: retention per episode, binges and forward skips.</figcaption>
  </figure>
  <figure>
    <img :src="withBase('/screenshots/en/insights.jpg')" alt="Insights page">
    <figcaption>Insights: completion, binges, pauses and weekly rhythm.</figcaption>
  </figure>
  <figure>
    <img :src="withBase('/screenshots/en/playback.jpg')" alt="Playback page">
    <figcaption>Playback health: direct play, transcodes and their reasons.</figcaption>
  </figure>
  <figure>
    <img :src="withBase('/screenshots/en/recap.jpg')" alt="Recap page">
    <figcaption>Yearly and monthly recap for each user.</figcaption>
  </figure>
  <figure>
    <img :src="withBase('/screenshots/en/me.jpg')" alt="My statistics page">
    <figcaption>“My statistics”, inside Jellyfin, for every user.</figcaption>
  </figure>
</div>

## How the pieces fit

<Workflow />

Read [how it works](/guide/how-it-works) for the details.

## Start in two minutes

1. Copy the `docker-compose.yaml` from the [installation guide](/guide/installation) and set a long random `JWT_SECRET`.
2. Run `docker compose up -d`.
3. Open `http://localhost:3000`, connect your Jellyfin server and sign in with a Jellyfin administrator account.
4. [Install the plugin](/plugin/) in Jellyfin for the most precise data.

</div>
