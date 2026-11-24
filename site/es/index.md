---
layout: home
title: Jellystics
titleTemplate: Estadísticas para Jellyfin

hero:
  name: Jellystics
  text: Estadísticas precisas para tu servidor Jellyfin
  tagline: Cada reproducción registrada en origen, el historial completo conservado y todo disponible en tu panel y en Grafana.
  image:
    src: /logo.png
    alt: Jellystics
  actions:
    - theme: brand
      text: Empezar
      link: /es/guide/introduction
    - theme: alt
      text: Instalar el plugin
      link: /es/plugin/
    - theme: alt
      text: GitHub
      link: https://github.com/Jellystics/jellystics

features:
  - icon: 🎯
    title: Tiempo de reproducción real
    details: El plugin de Jellyfin registra las reproducciones a partir de los eventos del servidor. Las pausas, el búfer y los retrocesos nunca cuentan; un reinicio pierde como máximo 15 segundos.
  - icon: 👆
    title: Clics y vistas
    details: Cada inicio es un clic; una vista exige 30 segundos realmente vistos. Finalización, maratones, reanudaciones y revisionados se calculan por usuario y por día.
  - icon: 📊
    title: Un panel completo
    details: Resumen con comparativas, sesiones en directo, historial en tabla y calendario, bibliotecas, películas, series, artistas, álbumes, usuarios y resúmenes anuales.
  - icon: 🔎
    title: Análisis y salud de la reproducción
    details: Distribución de finalización, ritmo semanal, títulos abandonados, almacenamiento nunca visto, motivos de transcodificación, resolución servida, HDR y tasa de bits.
  - icon: 📈
    title: Prometheus y Grafana
    details: Un endpoint /metrics protegido por clave API con 21 familias de métricas, y un panel de Grafana de 31 paneles listo para usar.
  - icon: 🙋
    title: Estadísticas para cada usuario
    details: Cada usuario de Jellyfin tiene una página «Mis estadísticas» en el menú de su avatar, servida desde el origen de Jellyfin y que solo muestra sus propios datos.
---

<script setup>
import { withBase } from 'vitepress'
</script>

<div class="home-section">

## Échale un vistazo {#see-it}

<div class="home-gallery">
  <figure class="wide">
    <img :src="withBase('/screenshots/es/overview.jpg')" alt="Página Resumen">
    <figcaption>Resumen: tiempo de reproducción, vistas, clics, cada reproducción y reproducciones por biblioteca.</figcaption>
  </figure>
  <figure>
    <img :src="withBase('/screenshots/es/live.jpg')" alt="Página En directo">
    <figcaption>Sesiones en directo, tasa de salida y reproducciones simultáneas.</figcaption>
  </figure>
  <figure>
    <img :src="withBase('/screenshots/es/series.jpg')" alt="Página de una serie">
    <figcaption>Una serie: retención por episodio, maratones y saltos hacia delante.</figcaption>
  </figure>
  <figure>
    <img :src="withBase('/screenshots/es/insights.jpg')" alt="Página Análisis">
    <figcaption>Análisis: finalización, maratones, pausas y ritmo semanal.</figcaption>
  </figure>
  <figure>
    <img :src="withBase('/screenshots/es/playback.jpg')" alt="Página Reproducción">
    <figcaption>Salud de la reproducción: direct play, transcodificaciones y sus motivos.</figcaption>
  </figure>
  <figure>
    <img :src="withBase('/screenshots/es/recap.jpg')" alt="Página de resumen anual">
    <figcaption>Resumen anual y mensual de cada usuario.</figcaption>
  </figure>
  <figure>
    <img :src="withBase('/screenshots/es/me.jpg')" alt="Página Mis estadísticas">
    <figcaption>«Mis estadísticas», dentro de Jellyfin, para cada usuario.</figcaption>
  </figure>
</div>

## Cómo encajan las piezas {#how-the-pieces-fit}

<Workflow />

Lee [cómo funciona](/es/guide/how-it-works) para conocer los detalles.

## Empieza en dos minutos {#start-in-two-minutes}

1. Copia el `docker-compose.yaml` de la [guía de instalación](/es/guide/installation) y define un `JWT_SECRET` largo y aleatorio.
2. Ejecuta `docker compose up -d`.
3. Abre `http://localhost:3000`, conecta tu servidor Jellyfin e inicia sesión con una cuenta de administrador de Jellyfin.
4. [Instala el plugin](/es/plugin/) en Jellyfin para obtener los datos más precisos.

</div>
