import { defineConfig, type DefaultTheme } from 'vitepress'

// GitHub Pages serves the site under /<repo>/. Override with SITE_BASE for a custom domain ("/").
const base = process.env.SITE_BASE ?? '/jellystics/'
const repo = 'https://github.com/Jellystics/jellystics'
const pluginRepo = 'https://github.com/Jellystics/jellystics-plugin-playbackreporting'

type Labels = {
  guide: string; platform: string; plugin: string; prometheus: string; reference: string
  intro: string; install: string; config: string; how: string; glossary: string
  dashboard: string; libraries: string; users: string; analysis: string; settings: string
  pOverview: string; pConfig: string; pTracking: string; pData: string; pSync: string; pMe: string; pApi: string
  promSetup: string; promMetrics: string
  api: string; dev: string
}

function sidebar(prefix: string, l: Labels): DefaultTheme.SidebarItem[] {
  return [
    {
      text: l.guide,
      items: [
        { text: l.intro, link: `${prefix}/guide/introduction` },
        { text: l.install, link: `${prefix}/guide/installation` },
        { text: l.config, link: `${prefix}/guide/configuration` },
        { text: l.how, link: `${prefix}/guide/how-it-works` },
        { text: l.glossary, link: `${prefix}/guide/glossary` },
      ],
    },
    {
      text: l.platform,
      items: [
        { text: l.dashboard, link: `${prefix}/platform/dashboard` },
        { text: l.libraries, link: `${prefix}/platform/libraries` },
        { text: l.users, link: `${prefix}/platform/users` },
        { text: l.analysis, link: `${prefix}/platform/analysis` },
        { text: l.settings, link: `${prefix}/platform/settings` },
      ],
    },
    {
      text: l.plugin,
      items: [
        { text: l.pOverview, link: `${prefix}/plugin/` },
        { text: l.pConfig, link: `${prefix}/plugin/configuration` },
        { text: l.pTracking, link: `${prefix}/plugin/tracking` },
        { text: l.pData, link: `${prefix}/plugin/data` },
        { text: l.pSync, link: `${prefix}/plugin/sync` },
        { text: l.pMe, link: `${prefix}/plugin/my-statistics` },
        { text: l.pApi, link: `${prefix}/plugin/api` },
      ],
    },
    {
      text: l.prometheus,
      items: [
        { text: l.promSetup, link: `${prefix}/prometheus/` },
        { text: l.promMetrics, link: `${prefix}/prometheus/metrics` },
      ],
    },
    {
      text: l.reference,
      items: [
        { text: l.api, link: `${prefix}/reference/api` },
        { text: l.dev, link: `${prefix}/reference/development` },
      ],
    },
  ]
}

const en: Labels = {
  guide: 'Guide', platform: 'Platform tour', plugin: 'Jellyfin plugin', prometheus: 'Prometheus', reference: 'Reference',
  intro: 'Introduction', install: 'Installation', config: 'Configuration', how: 'How it works', glossary: 'Glossary',
  dashboard: 'Overview, live, history', libraries: 'Libraries and media', users: 'Users and recaps', analysis: 'Insights, discover, playback', settings: 'Settings',
  pOverview: 'Overview and install', pConfig: 'Plugin settings', pTracking: 'Playback tracking', pData: 'Collected data', pSync: 'Sync with Jellystics', pMe: 'My statistics page', pApi: 'Plugin API',
  promSetup: 'Setup and Grafana', promMetrics: 'Metrics reference',
  api: 'Jellystics HTTP API', dev: 'Development',
}
const fr: Labels = {
  guide: 'Guide', platform: 'Visite de la plateforme', plugin: 'Plugin Jellyfin', prometheus: 'Prometheus', reference: 'Référence',
  intro: 'Introduction', install: 'Installation', config: 'Configuration', how: 'Fonctionnement', glossary: 'Glossaire',
  dashboard: 'Vue d\'ensemble, direct, historique', libraries: 'Bibliothèques et médias', users: 'Utilisateurs et récaps', analysis: 'Analyses, découverte, lecture', settings: 'Paramètres',
  pOverview: 'Présentation et installation', pConfig: 'Configuration du plugin', pTracking: 'Suivi des lectures', pData: 'Données collectées', pSync: 'Synchronisation avec Jellystics', pMe: 'Page « Mes statistiques »', pApi: 'API du plugin',
  promSetup: 'Mise en place et Grafana', promMetrics: 'Référence des métriques',
  api: 'API HTTP de Jellystics', dev: 'Développement',
}
const es: Labels = {
  guide: 'Guía', platform: 'Recorrido por la plataforma', plugin: 'Plugin de Jellyfin', prometheus: 'Prometheus', reference: 'Referencia',
  intro: 'Introducción', install: 'Instalación', config: 'Configuración', how: 'Funcionamiento', glossary: 'Glosario',
  dashboard: 'Resumen, en vivo, historial', libraries: 'Bibliotecas y medios', users: 'Usuarios y resúmenes', analysis: 'Análisis, descubrir, reproducción', settings: 'Ajustes',
  pOverview: 'Presentación e instalación', pConfig: 'Ajustes del plugin', pTracking: 'Seguimiento de reproducciones', pData: 'Datos recogidos', pSync: 'Sincronización con Jellystics', pMe: 'Página «Mis estadísticas»', pApi: 'API del plugin',
  promSetup: 'Puesta en marcha y Grafana', promMetrics: 'Referencia de métricas',
  api: 'API HTTP de Jellystics', dev: 'Desarrollo',
}

function nav(prefix: string, l: Labels): DefaultTheme.NavItem[] {
  return [
    { text: l.guide, link: `${prefix}/guide/introduction`, activeMatch: `${prefix}/guide/` },
    { text: l.platform, link: `${prefix}/platform/dashboard`, activeMatch: `${prefix}/platform/` },
    { text: l.plugin, link: `${prefix}/plugin/`, activeMatch: `${prefix}/plugin/` },
    { text: l.prometheus, link: `${prefix}/prometheus/`, activeMatch: `${prefix}/prometheus/` },
    {
      text: 'GitHub',
      items: [
        { text: 'Jellystics', link: repo },
        { text: 'Jellystics plugin', link: pluginRepo },
        { text: 'Jellystics-Grafana', link: 'https://github.com/Jellystics/Jellystics-Grafana' },
      ],
    },
  ]
}

export default defineConfig({
  base,
  title: 'Jellystics',
  cleanUrls: true,
  ignoreDeadLinks: [/^https?:\/\/localhost/],
  lastUpdated: false,
  head: [
    ['link', { rel: 'icon', type: 'image/png', href: `${base}logo.png` }],
    ['meta', { name: 'theme-color', content: '#3987e5' }],
    ['meta', { property: 'og:image', content: `${base}screenshots/en/overview.jpg` }],
  ],
  appearance: 'dark',
  themeConfig: {
    logo: { src: '/logo.png', alt: 'Jellystics' },
    socialLinks: [{ icon: 'github', link: repo }],
    search: {
      provider: 'local',
      options: {
        locales: {
          fr: {
            translations: {
              button: { buttonText: 'Rechercher', buttonAriaLabel: 'Rechercher' },
              modal: {
                displayDetails: 'Afficher la liste détaillée', resetButtonTitle: 'Effacer la recherche', backButtonTitle: 'Fermer la recherche',
                noResultsText: 'Aucun résultat pour',
                footer: { selectText: 'choisir', navigateText: 'naviguer', closeText: 'fermer' },
              },
            },
          },
          es: {
            translations: {
              button: { buttonText: 'Buscar', buttonAriaLabel: 'Buscar' },
              modal: {
                displayDetails: 'Mostrar lista detallada', resetButtonTitle: 'Borrar búsqueda', backButtonTitle: 'Cerrar búsqueda',
                noResultsText: 'Sin resultados para',
                footer: { selectText: 'elegir', navigateText: 'navegar', closeText: 'cerrar' },
              },
            },
          },
        },
      },
    },
  },
  locales: {
    root: {
      label: 'English',
      lang: 'en',
      description: 'Precise statistics for your Jellyfin server: dashboard, Jellyfin plugin and Prometheus metrics.',
      themeConfig: {
        nav: nav('', en),
        sidebar: sidebar('', en),
        footer: { message: 'Jellystics is open source. The Jellyfin plugin is released under the GPL-3 license.', copyright: 'Jellystics' },
        outline: { level: [2, 3], label: 'On this page' },
      },
    },
    fr: {
      label: 'Français',
      lang: 'fr',
      link: '/fr/',
      description: 'Des statistiques précises pour votre serveur Jellyfin : tableau de bord, plugin Jellyfin et métriques Prometheus.',
      themeConfig: {
        nav: nav('/fr', fr),
        sidebar: sidebar('/fr', fr),
        footer: { message: 'Jellystics est open source. Le plugin Jellyfin est publié sous licence GPL-3.', copyright: 'Jellystics' },
        outline: { level: [2, 3], label: 'Sur cette page' },
        docFooter: { prev: 'Page précédente', next: 'Page suivante' },
        darkModeSwitchLabel: 'Apparence',
        lightModeSwitchTitle: 'Passer au thème clair',
        darkModeSwitchTitle: 'Passer au thème sombre',
        sidebarMenuLabel: 'Menu',
        returnToTopLabel: 'Retour en haut',
        langMenuLabel: 'Changer de langue',
        notFound: { title: 'Page introuvable', quote: 'Cette page n\'existe pas ou a été déplacée.', linkText: 'Retour à l\'accueil' },
      },
    },
    es: {
      label: 'Español',
      lang: 'es',
      link: '/es/',
      description: 'Estadísticas precisas para tu servidor Jellyfin: panel, plugin de Jellyfin y métricas de Prometheus.',
      themeConfig: {
        nav: nav('/es', es),
        sidebar: sidebar('/es', es),
        footer: { message: 'Jellystics es de código abierto. El plugin de Jellyfin se publica bajo licencia GPL-3.', copyright: 'Jellystics' },
        outline: { level: [2, 3], label: 'En esta página' },
        docFooter: { prev: 'Página anterior', next: 'Página siguiente' },
        darkModeSwitchLabel: 'Apariencia',
        lightModeSwitchTitle: 'Cambiar al tema claro',
        darkModeSwitchTitle: 'Cambiar al tema oscuro',
        sidebarMenuLabel: 'Menú',
        returnToTopLabel: 'Volver arriba',
        langMenuLabel: 'Cambiar idioma',
        notFound: { title: 'Página no encontrada', quote: 'Esta página no existe o se ha movido.', linkText: 'Volver al inicio' },
      },
    },
  },
})
