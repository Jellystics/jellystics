<script setup lang="ts">
import { computed } from 'vue'
import { useData } from 'vitepress'

// Data flow of Jellystics, drawn with the app's design tokens (see custom.css, --wf-*).
// Technical strings (routes, file names) stay in English; prose follows the page language.

type Dict = Record<string, string> & { steps: string[] } & Record<'stepsTitle' | 'legendMain' | 'legendAlt', string>
const DICTS: Record<string, Dict> = {
  en: {
    hJellyfin: 'Jellyfin server', hJellystics: 'Jellystics (Docker)', hOut: 'Consumers',
    events: 'Server events', plugin: 'Jellystics plugin', pluginL1: 'one state machine per play', pluginL2: 'checkpoint every 15 s',
    db: 'jellystics.db (SQLite)', dbL1: 'global version cursor', api: 'Jellyfin HTTP API',
    sync: 'Plugin sync', syncL1: 'every 30 s', tracker: 'Tracker (fallback)', trackerL1: 'plugin silent > 2 min',
    lib: 'Library sync', libL1: 'nightly · recent hourly',
    pg: 'PostgreSQL', pgL1: 'one table of plays', pgL2: 'plugin replaces tracker',
    stats: 'Statistics', statsL1: 'SQL at query time', statsL2: 'TZ · views ≥ 30 s', http: 'HTTP API',
    dash: 'Dashboard', dashL1: 'Jellyfin administrators', prom: 'Prometheus → Grafana', promL1: 'Bearer API key',
    discord: 'Discord webhooks', discordL1: 'tasks · backups · keys',
    users: 'Jellyfin users', usersL1: 'avatar menu → My statistics', usersL2: 'relayed by the plugin',
    push: 'history', sched: 'Scheduler', schedL1: 'cron expressions in the TZ time zone',
    legendMain: 'main path (plugin)', legendAlt: 'fallback or optional',
    stepsTitle: 'Step by step',
    steps: [
      'Jellyfin raises an event for every playback start, progress, pause and stop, every library change, every played or favourite mark and every sign-in. The plugin receives them in real time.',
      'The plugin keeps one state machine per play (watched time, pauses, seeks, tracks, transcoding) and saves it to jellystics.db every 15 s. Each change gets a new version number.',
      'Every 30 s, Jellystics asks the plugin for everything after its last cursor (GET /Jellystics/Plays?since=…), 500 rows per page, and the same for library events, marks and logins.',
      'If the plugin has not answered for 2 minutes, the built-in tracker takes over and polls /Sessions every 5 s.',
      'The library sync reads items and users from the Jellyfin API: a full sync every night, recently added items every hour.',
      'Everything lands in PostgreSQL, in one table of plays whatever the source. Plugin plays replace the tracker rows they cover, so nothing is counted twice.',
      'Statistics are computed in SQL when they are requested, in the TZ time zone: views (≥ 30 s), completion (≥ 90 %), binges, retention…',
      'They are served to the dashboard (/api/v2 and the live WebSocket), to Prometheus (/metrics), to Discord webhooks (task, backup and API key events), and to every Jellyfin user through the plugin relay (/Jellystics/Me → /api/me with the shared secret).',
    ],
  },
  fr: {
    hJellyfin: 'Serveur Jellyfin', hJellystics: 'Jellystics (Docker)', hOut: 'Consommateurs',
    events: 'Événements du serveur', plugin: 'Plugin Jellystics', pluginL1: 'une machine à états par lecture', pluginL2: 'sauvegarde toutes les 15 s',
    db: 'jellystics.db (SQLite)', dbL1: 'curseur de version global', api: 'API HTTP de Jellyfin',
    sync: 'Synchro du plugin', syncL1: 'toutes les 30 s', tracker: 'Tracker (secours)', trackerL1: 'plugin muet > 2 min',
    lib: 'Synchro bibliothèques', libL1: 'la nuit · récents chaque heure',
    pg: 'PostgreSQL', pgL1: 'une table de lectures', pgL2: 'le plugin remplace le tracker',
    stats: 'Statistiques', statsL1: 'SQL à la demande', statsL2: 'TZ · vues ≥ 30 s', http: 'API HTTP',
    dash: 'Tableau de bord', dashL1: 'administrateurs Jellyfin', prom: 'Prometheus → Grafana', promL1: 'clé d\'API Bearer',
    discord: 'Webhooks Discord', discordL1: 'tâches · sauvegardes · clés',
    users: 'Utilisateurs Jellyfin', usersL1: 'menu avatar → Mes statistiques', usersL2: 'relayé par le plugin',
    push: 'historique', sched: 'Planificateur', schedL1: 'expressions cron dans le fuseau TZ',
    legendMain: 'chemin principal (plugin)', legendAlt: 'secours ou optionnel',
    stepsTitle: 'Étape par étape',
    steps: [
      'Jellyfin émet un événement à chaque début, progression, pause et arrêt de lecture, à chaque changement de bibliothèque, à chaque marque « vu » ou favori et à chaque connexion. Le plugin les reçoit en temps réel.',
      'Le plugin tient une machine à états par lecture (temps regardé, pauses, sauts, pistes, transcodage) et l\'enregistre dans jellystics.db toutes les 15 s. Chaque modification reçoit un nouveau numéro de version.',
      'Toutes les 30 s, Jellystics demande au plugin tout ce qui suit son dernier curseur (GET /Jellystics/Plays?since=…), 500 lignes par page, et de même pour les événements de bibliothèque, les marques et les connexions.',
      'Si le plugin ne répond plus depuis 2 minutes, le tracker intégré prend le relais et interroge /Sessions toutes les 5 s.',
      'La synchro des bibliothèques lit les éléments et les utilisateurs via l\'API Jellyfin : synchro complète chaque nuit, ajouts récents chaque heure.',
      'Tout arrive dans PostgreSQL, dans une seule table de lectures quelle que soit la source. Les lectures du plugin remplacent les lignes du tracker qu\'elles couvrent : rien n\'est compté deux fois.',
      'Les statistiques sont calculées en SQL au moment de la demande, dans le fuseau TZ : vues (≥ 30 s), complétion (≥ 90 %), binges, rétention…',
      'Elles sont servies au tableau de bord (/api/v2 et le WebSocket du direct), à Prometheus (/metrics), aux webhooks Discord (événements de tâches, de sauvegardes et de clés d\'API) et à chaque utilisateur Jellyfin via le relais du plugin (/Jellystics/Me → /api/me avec le secret partagé).',
    ],
  },
  es: {
    hJellyfin: 'Servidor Jellyfin', hJellystics: 'Jellystics (Docker)', hOut: 'Consumidores',
    events: 'Eventos del servidor', plugin: 'Plugin de Jellystics', pluginL1: 'una máquina de estados por lectura', pluginL2: 'guardado cada 15 s',
    db: 'jellystics.db (SQLite)', dbL1: 'cursor de versión global', api: 'API HTTP de Jellyfin',
    sync: 'Sync del plugin', syncL1: 'cada 30 s', tracker: 'Tracker (respaldo)', trackerL1: 'plugin callado > 2 min',
    lib: 'Sync de bibliotecas', libL1: 'de noche · recientes cada hora',
    pg: 'PostgreSQL', pgL1: 'una tabla de lecturas', pgL2: 'el plugin sustituye al tracker',
    stats: 'Estadísticas', statsL1: 'SQL bajo demanda', statsL2: 'TZ · vistas ≥ 30 s', http: 'API HTTP',
    dash: 'Panel', dashL1: 'administradores de Jellyfin', prom: 'Prometheus → Grafana', promL1: 'clave API Bearer',
    discord: 'Webhooks de Discord', discordL1: 'tareas · copias · claves',
    users: 'Usuarios de Jellyfin', usersL1: 'menú avatar → Mis estadísticas', usersL2: 'a través del plugin',
    push: 'historial', sched: 'Planificador', schedL1: 'expresiones cron en la zona horaria TZ',
    legendMain: 'camino principal (plugin)', legendAlt: 'respaldo u opcional',
    stepsTitle: 'Paso a paso',
    steps: [
      'Jellyfin emite un evento en cada inicio, progreso, pausa y parada de reproducción, en cada cambio de biblioteca, en cada marca de visto o favorito y en cada inicio de sesión. El plugin los recibe en tiempo real.',
      'El plugin mantiene una máquina de estados por reproducción (tiempo visto, pausas, saltos, pistas, transcodificación) y la guarda en jellystics.db cada 15 s. Cada cambio recibe un nuevo número de versión.',
      'Cada 30 s, Jellystics pide al plugin todo lo posterior a su último cursor (GET /Jellystics/Plays?since=…), 500 filas por página, y lo mismo para los eventos de biblioteca, las marcas y los inicios de sesión.',
      'Si el plugin no responde desde hace 2 minutos, el tracker integrado toma el relevo y consulta /Sessions cada 5 s.',
      'La sincronización de bibliotecas lee elementos y usuarios desde la API de Jellyfin: sincronización completa cada noche, elementos recientes cada hora.',
      'Todo llega a PostgreSQL, a una única tabla de reproducciones sea cual sea la fuente. Las reproducciones del plugin sustituyen a las filas del tracker que cubren: nada se cuenta dos veces.',
      'Las estadísticas se calculan en SQL en el momento de la consulta, en la zona horaria TZ: vistas (≥ 30 s), finalización (≥ 90 %), maratones, retención…',
      'Se sirven al panel (/api/v2 y el WebSocket en vivo), a Prometheus (/metrics), a los webhooks de Discord (eventos de tareas, copias de seguridad y claves API) y a cada usuario de Jellyfin a través del relé del plugin (/Jellystics/Me → /api/me con el secreto compartido).',
    ],
  },
}

const { lang } = useData()
const t = computed<Dict>(() => DICTS[lang.value.slice(0, 2)] ?? DICTS.en)
const pushNote = computed(() => ({
  en: 'Dashed line from Jellystics to the plugin: “Save the history to the plugin” sends the plays Jellystics recorded itself (POST /Jellystics/Import/Plays, batches of 500), so the plugin holds the complete history.',
  fr: 'Ligne pointillée de Jellystics vers le plugin : « Sauvegarder l\'historique dans le plugin » envoie les lectures enregistrées par Jellystics lui-même (POST /Jellystics/Import/Plays, par lots de 500), pour que le plugin détienne l\'historique complet.',
  es: 'Línea discontinua de Jellystics al plugin: «Guardar el historial en el plugin» envía las reproducciones que Jellystics registró por sí mismo (POST /Jellystics/Import/Plays, lotes de 500), para que el plugin tenga el historial completo.',
}[lang.value.slice(0, 2)] ?? ''))
</script>

<template>
  <figure class="wf">
    <div class="wf-scroll">
      <svg class="wf-svg" viewBox="0 0 1120 652" role="img" :aria-label="t.stepsTitle">
        <defs>
          <marker id="wf-arrow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse">
            <path d="M0,0 L10,5 L0,10 z" class="wf-head-fill" />
          </marker>
          <marker id="wf-arrow-accent" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse">
            <path d="M0,0 L10,5 L0,10 z" class="wf-head-fill-accent" />
          </marker>
        </defs>

        <!-- column headers -->
        <text x="16" y="26" class="wf-col">{{ t.hJellyfin.toUpperCase() }}</text>
        <text x="336" y="26" class="wf-col">{{ t.hJellystics.toUpperCase() }}</text>
        <text x="852" y="26" class="wf-col">{{ t.hOut.toUpperCase() }}</text>

        <!-- hosts -->
        <rect x="16" y="40" width="280" height="560" rx="8" class="wf-host" />
        <rect x="336" y="40" width="460" height="560" rx="8" class="wf-host" />

        <!-- Jellyfin column -->
        <rect x="32" y="60" width="248" height="112" rx="8" class="wf-box" />
        <text x="48" y="84" class="wf-title">{{ t.events }}</text>
        <text x="48" y="108" class="wf-mono">PlaybackStart · Progress · Stopped</text>
        <text x="48" y="128" class="wf-mono">ItemAdded · ItemRemoved</text>
        <text x="48" y="148" class="wf-mono">UserDataSaved · Authentication</text>

        <rect x="32" y="204" width="248" height="92" rx="8" class="wf-box wf-box--accent" />
        <text x="48" y="228" class="wf-title">{{ t.plugin }}</text>
        <text x="48" y="252" class="wf-text">{{ t.pluginL1 }}</text>
        <text x="48" y="274" class="wf-text">{{ t.pluginL2 }}</text>

        <rect x="32" y="328" width="248" height="72" rx="8" class="wf-box" />
        <text x="48" y="352" class="wf-title">{{ t.db }}</text>
        <text x="48" y="376" class="wf-text">{{ t.dbL1 }}</text>

        <rect x="32" y="440" width="248" height="120" rx="8" class="wf-box" />
        <text x="48" y="464" class="wf-title">{{ t.api }}</text>
        <text x="48" y="488" class="wf-mono">/Sessions</text>
        <text x="48" y="508" class="wf-mono">/Items · /Users</text>
        <text x="48" y="528" class="wf-mono">/Jellystics/* (plugin routes)</text>

        <path d="M156 172 V202" class="wf-line wf-line--accent" marker-end="url(#wf-arrow-accent)" />
        <path d="M156 296 V326" class="wf-line wf-line--accent" marker-end="url(#wf-arrow-accent)" />

        <!-- Jellystics: scheduler -->
        <rect x="352" y="60" width="428" height="112" rx="8" class="wf-box" />
        <text x="368" y="84" class="wf-title">{{ t.sched }}</text>
        <text x="368" y="104" class="wf-text">{{ t.schedL1 }}</text>
        <text x="368" y="126" class="wf-mono">Plugin sync · every 30 s</text>
        <text x="368" y="143" class="wf-mono">Tracker · every 5 s</text>
        <text x="368" y="160" class="wf-mono">Geolocation · every 24 h</text>
        <text x="592" y="126" class="wf-mono">Full sync · 0 0 * * *</text>
        <text x="592" y="143" class="wf-mono">Recently added · 0 * * * *</text>
        <text x="592" y="160" class="wf-mono">Backup · 0 3 * * *</text>

        <!-- Jellystics: ingest -->
        <rect x="352" y="204" width="200" height="124" rx="8" class="wf-box wf-box--accent" />
        <text x="368" y="228" class="wf-title">{{ t.sync }}</text>
        <text x="368" y="250" class="wf-text">{{ t.syncL1 }}</text>
        <text x="368" y="272" class="wf-mono">GET /Jellystics/Plays</text>
        <text x="368" y="290" class="wf-mono">?since=cursor</text>
        <text x="368" y="308" class="wf-mono">&amp;limit=500</text>

        <rect x="352" y="360" width="200" height="80" rx="8" class="wf-box wf-box--dashed" />
        <text x="368" y="384" class="wf-title">{{ t.tracker }}</text>
        <text x="368" y="406" class="wf-text">{{ t.trackerL1 }}</text>
        <text x="368" y="426" class="wf-mono">GET /Sessions · 5 s</text>

        <rect x="352" y="472" width="200" height="88" rx="8" class="wf-box" />
        <text x="368" y="496" class="wf-title">{{ t.lib }}</text>
        <text x="368" y="518" class="wf-text">{{ t.libL1 }}</text>
        <text x="368" y="540" class="wf-mono">/Items · /Users</text>

        <!-- 3: plugin db -> plugin sync -->
        <path d="M280 364 H316 V300 H350" class="wf-line wf-line--accent" marker-end="url(#wf-arrow-accent)" />
        <!-- history push (Jellystics -> plugin) -->
        <path d="M352 226 H282" class="wf-line wf-line--dashed" marker-end="url(#wf-arrow)" />
        <text x="316" y="218" text-anchor="middle" class="wf-small">{{ t.push }}</text>
        <!-- 4: API -> tracker -->
        <path d="M280 400 H350" class="wf-line wf-line--dashed" marker-end="url(#wf-arrow)" />
        <!-- 5: API -> library sync -->
        <path d="M280 516 H350" class="wf-line" marker-end="url(#wf-arrow)" />

        <!-- Jellystics: store and serve -->
        <rect x="576" y="204" width="204" height="110" rx="8" class="wf-box" />
        <text x="592" y="228" class="wf-title">{{ t.pg }}</text>
        <text x="592" y="250" class="wf-text">{{ t.pgL1 }}</text>
        <text x="592" y="270" class="wf-text">{{ t.pgL2 }}</text>
        <text x="592" y="294" class="wf-mono">jf_playback_activity</text>

        <rect x="576" y="356" width="204" height="84" rx="8" class="wf-box" />
        <text x="592" y="380" class="wf-title">{{ t.stats }}</text>
        <text x="592" y="402" class="wf-text">{{ t.statsL1 }}</text>
        <text x="592" y="422" class="wf-text">{{ t.statsL2 }}</text>

        <rect x="576" y="472" width="204" height="96" rx="8" class="wf-box" />
        <text x="592" y="496" class="wf-title">{{ t.http }}</text>
        <text x="592" y="518" class="wf-mono">/api/v2 · /socket.io</text>
        <text x="592" y="538" class="wf-mono">/metrics</text>
        <text x="592" y="558" class="wf-mono">/api/me</text>

        <!-- 6: into PostgreSQL -->
        <path d="M552 236 H574" class="wf-line wf-line--accent" marker-end="url(#wf-arrow-accent)" />
        <path d="M552 400 H564 M552 516 H564 M564 516 V290 H574" class="wf-line" marker-end="url(#wf-arrow)" />
        <!-- 7, then to the HTTP API -->
        <path d="M678 314 V354" class="wf-line" marker-end="url(#wf-arrow)" />
        <path d="M678 440 V470" class="wf-line" marker-end="url(#wf-arrow)" />

        <!-- consumers -->
        <rect x="852" y="60" width="236" height="84" rx="8" class="wf-box" />
        <text x="868" y="84" class="wf-title">{{ t.dash }}</text>
        <text x="868" y="106" class="wf-text">{{ t.dashL1 }}</text>
        <text x="868" y="126" class="wf-mono">/api/v2 · live 5 s</text>

        <rect x="852" y="172" width="236" height="84" rx="8" class="wf-box" />
        <text x="868" y="196" class="wf-title">{{ t.prom }}</text>
        <text x="868" y="218" class="wf-mono">GET /metrics</text>
        <text x="868" y="238" class="wf-text">{{ t.promL1 }}</text>

        <rect x="852" y="284" width="236" height="76" rx="8" class="wf-box" />
        <text x="868" y="308" class="wf-title">{{ t.discord }}</text>
        <text x="868" y="330" class="wf-text">{{ t.discordL1 }}</text>

        <rect x="852" y="388" width="236" height="100" rx="8" class="wf-box" />
        <text x="868" y="412" class="wf-title">{{ t.users }}</text>
        <text x="868" y="434" class="wf-text">{{ t.usersL1 }}</text>
        <text x="868" y="454" class="wf-text">{{ t.usersL2 }}</text>
        <text x="868" y="474" class="wf-mono">/Jellystics/Me → /api/me</text>

        <!-- 8: HTTP API -> consumers -->
        <path d="M780 520 H816 V102" class="wf-line" />
        <path d="M816 102 H850 M816 214 H850 M816 322 H850 M816 438 H850" class="wf-line" />
        <path d="M840 102 H850" class="wf-line" marker-end="url(#wf-arrow)" />
        <path d="M840 214 H850" class="wf-line" marker-end="url(#wf-arrow)" />
        <path d="M840 322 H850" class="wf-line" marker-end="url(#wf-arrow)" />
        <path d="M840 438 H850" class="wf-line" marker-end="url(#wf-arrow)" />

        <!-- step badges -->
        <g class="wf-badge"><circle cx="176" cy="187" r="10" /><text x="176" y="191">1</text></g>
        <g class="wf-badge"><circle cx="176" cy="311" r="10" /><text x="176" y="315">2</text></g>
        <g class="wf-badge"><circle cx="316" cy="334" r="10" /><text x="316" y="338">3</text></g>
        <g class="wf-badge wf-badge--neutral"><circle cx="316" cy="400" r="10" /><text x="316" y="404">4</text></g>
        <g class="wf-badge wf-badge--neutral"><circle cx="316" cy="516" r="10" /><text x="316" y="520">5</text></g>
        <g class="wf-badge"><circle cx="564" cy="236" r="10" /><text x="564" y="240">6</text></g>
        <g class="wf-badge wf-badge--neutral"><circle cx="698" cy="334" r="10" /><text x="698" y="338">7</text></g>
        <g class="wf-badge wf-badge--neutral"><circle cx="816" cy="470" r="10" /><text x="816" y="474">8</text></g>

        <!-- legend -->
        <path d="M16 628 H52" class="wf-line wf-line--accent" />
        <text x="62" y="632" class="wf-small">{{ t.legendMain }}</text>
        <path d="M268 628 H304" class="wf-line wf-line--dashed" />
        <text x="314" y="632" class="wf-small">{{ t.legendAlt }}</text>
      </svg>
    </div>

    <figcaption class="wf-steps">
      <p class="wf-steps-title">{{ t.stepsTitle }}</p>
      <ol>
        <li v-for="(step, i) in t.steps" :key="i"><span class="wf-num" :class="{ 'wf-num--neutral': ![0, 1, 2, 5].includes(i) }">{{ i + 1 }}</span><span>{{ step }}</span></li>
      </ol>
      <p class="wf-note">{{ pushNote }}</p>
    </figcaption>
  </figure>
</template>

<style scoped>
/* Wider than the text column: centred on it, up to the width of the main area. */
.wf {
  margin: 24px 0;
  position: relative;
  left: 50%;
  transform: translateX(-50%);
  width: min(1104px, calc(100vw - 48px));
}
@media (min-width: 960px) {
  .wf { width: min(1104px, calc(100vw - var(--vp-sidebar-width) - 96px)); }
}
.VPHome .wf { width: min(1104px, calc(100vw - 48px)); }
.wf-scroll { overflow-x: auto; }
.wf-svg { display: block; width: 100%; min-width: 760px; height: auto; font-family: var(--vp-font-family-base); }

.wf-col { font-size: 11px; font-weight: 600; letter-spacing: 0.08em; fill: var(--wf-ink-3); }
.wf-host { fill: var(--wf-surface); stroke: var(--wf-border); stroke-width: 1; }
.wf-box { fill: var(--wf-surface-2); stroke: var(--wf-border-strong); stroke-width: 1; }
.wf-box--accent { fill: var(--wf-accent-wash); stroke: var(--wf-accent); }
.wf-box--dashed { stroke-dasharray: 4 3; }
.wf-title { font-size: 13px; font-weight: 600; fill: var(--wf-ink); }
.wf-text { font-size: 11.5px; fill: var(--wf-ink-2); }
.wf-mono { font-size: 10.5px; font-family: var(--vp-font-family-mono); fill: var(--wf-ink-3); }
.wf-small { font-size: 10.5px; fill: var(--wf-ink-3); }

.wf-line { fill: none; stroke: var(--wf-line); stroke-width: 1.5; }
.wf-line--accent { stroke: var(--wf-accent); }
.wf-line--dashed { stroke-dasharray: 5 4; }
.wf-head-fill { fill: var(--wf-line); }
.wf-head-fill-accent { fill: var(--wf-accent); }

.wf-badge circle { fill: var(--wf-accent); }
.wf-badge text { font-size: 11px; font-weight: 700; text-anchor: middle; fill: var(--wf-on-accent); }
.wf-badge--neutral circle { fill: var(--wf-surface); stroke: var(--wf-line); stroke-width: 1.5; }
.wf-badge--neutral text { fill: var(--wf-ink); }

.wf-steps { margin-top: 20px; }
.wf-steps-title { font-weight: 600; color: var(--vp-c-text-1); margin: 0 0 8px; }
.wf-steps ol { list-style: none; padding: 0; margin: 0; }
.wf-steps li { display: flex; gap: 12px; align-items: flex-start; margin: 0 0 10px; color: var(--vp-c-text-2); font-size: 14.5px; line-height: 1.55; }
.wf-num {
  flex: none; width: 22px; height: 22px; border-radius: 999px; margin-top: 1px;
  display: inline-grid; place-items: center; font-size: 11.5px; font-weight: 700;
  background: var(--wf-accent); color: var(--wf-on-accent);
}
.wf-num--neutral { background: var(--wf-surface); color: var(--wf-ink); box-shadow: inset 0 0 0 1.5px var(--wf-line); }
.wf-note { font-size: 14px; color: var(--vp-c-text-2); margin: 12px 0 0; padding-left: 12px; border-left: 2px solid var(--wf-border-strong); }
</style>
