# Refonte du frontend Jellystics — spec

Date : 2026-11-22 · Branche : `feat/frontend-redesign` · Référence visuelle : `design/jellystics-design.html` (fait foi pour tout rendu).

## 1. Objectif

Reconstruire entièrement le frontend selon la direction artistique B « Instrument » validée, corriger les incohérences et bugs relevés par l'audit, refondre l'architecture des pages et des URL, et ajouter les nouvelles vues (section « Nouveautés » de la page de design), y compris les endpoints backend et la capture tracker qu'elles nécessitent.

Critère de réussite : chaque page de l'app correspond à sa maquette dans `design/jellystics-design.html` (tokens, composants, graphiques, navigation), en clair et en sombre, de 390 px à 1440 px, sans régression fonctionnelle (auth, réglages, sauvegardes, tâches, webhooks, clés API).

Hors périmètre : la direction C « story » (export d'images du récap), une app mobile, le multi-serveur.

## 2. Décisions techniques

| Sujet | Décision |
|---|---|
| Framework | React 19 + Vite + TypeScript strict (inchangés) |
| UI kit | **Aucun kit**. Composants maison sur tokens CSS. MUI, MUI X (charts, date pickers), Emotion, recharts, `@tabler/icons-react`, `@fluentui/react-icons`, `@mui/icons-material`, `@mui/lab`, `react-image-crop`, `crypto-js` sont **retirés** |
| Accessibilité des surfaces flottantes | `@radix-ui/react-*` (dialog, dropdown-menu, popover, tooltip, tabs, toggle-group, switch, checkbox, select, scroll-area), non stylés, habillés par nos classes |
| Styles | `src/styles/tokens.css` (copie des tokens de la page de design, clair/sombre via `[data-theme]` + `prefers-color-scheme`), CSS Modules par composant (`*.module.css`). Pas de CSS-in-JS |
| Icônes | `lucide-react` (trait 1,6 px, 16 px UI / 20 px nav), même famille que la maquette |
| Graphiques | Module maison `src/charts/` en SVG, porté de la page de design (ligne/aire + comparaison, colonnes, colonnes empilées, aire empilée, heatmap jour×heure, calendrier, barre segmentée, bar list, entonnoir, rétention par épisode, histogramme, matrice, nuage de points, frise de sessions, petits multiples, sparkline). Infobulle et réticule communs, vue tableau pour chaque graphique |
| Données | `@tanstack/react-query` (cache, refetch, `placeholderData` pour garder l'ancien rendu au rafraîchissement, pas de flash de squelette) ; client `fetch` typé (remplace axios) |
| Tableaux | `@tanstack/react-table` (conservé), pagination serveur pour l'historique |
| Routage | React Router 7, routes déclarées dans `src/app/routes.tsx`, état d'URL via un hook `useUrlState` (période, comparaison, filtres, onglets) |
| Dates | `date-fns` + `date-fns-tz`, fuseau = TZ du serveur exposé par l'API (`GET /api/config/public` → `timezone`) ; les dates envoyées à l'API sont des dates **locales** `YYYY-MM-DD`, jamais `toISOString()` |
| i18n | i18next conservé (fr, en, es). Toutes les chaînes passent par `t()`, aucune valeur par défaut inline ; `lang` du document suit la langue |
| Temps réel | socket.io-client conservé (événement `sessions`, token envoyé à la connexion) |
| Tests | Vitest + Testing Library (formatters, hooks d'URL, composants, graphiques), Playwright (parcours de chaque route, clair/sombre, 390/1360 px, absence d'erreur console et de débordement horizontal) ; tests Go pour chaque nouvel endpoint |

## 3. Tokens (source : page de design, onglet Fondations)

- Neutres : `--bg --surface --surface-2 --surface-3 --border --border-strong --grid --axis --ink --ink-2 --ink-3`.
- Accent unique : `--accent --accent-hover --accent-wash --accent-ring`.
- Types de média : `--t-movie --t-series --t-music` (validés daltonisme, toujours avec libellé).
- Statut : `--good --warning --serious --critical` + `*-ink`. Réservés aux badges et alertes : **seule l'icône porte la couleur** (fond neutre, texte encre).
- Méthode de lecture (graphiques) : rampe ordinale bleue `--m-direct --m-stream --m-transcode` + `*-ink`.
- Séquentiel : `--seq-0` … `--seq-6`.
- Forme : `--r-sm 4px · --r-md 6px · --r-lg 8px · --r-full`. Aucune autre valeur.
- Espacement : grille 4 px `--s-1` … `--s-12`. Contrôle standard 32 px (dense 26, grand 38).
- Typo : police système ; `--fs-xs 11` … `--fs-hero 48` ; rien sous 11 px ; `tabular-nums` dans colonnes et axes uniquement.
- Élévation : bordure 1 px ; ombre `--shadow-pop` uniquement pour menus, popovers, infobulles, dialogues.
- Mouvement : 120 ms ease-out ; `prefers-reduced-motion` respecté.

## 4. Formats (module `src/lib/format.ts`, testé)

Durée depuis des secondes : `45 s · 12 min · 1 h 05 · 1 248 h`. Position média : `1:05:12`. Nombre : `Intl` locale, compact au-delà de 10 000 sur axes/tuiles. Pourcentage : 0 décimale sauf < 10 %. Date : `22 nov.` (+ année si ≠ courante), relatif dans les listes récentes. Heure : `21 h`. Semaine : lundi → dimanche partout. Delta : flèche selon le signe, couleur selon « bien/mal », période comparée nommée.

## 5. Composants (`src/ui/`)

Button (primary/secondary/ghost/danger × sm/md/lg, icon-only avec `aria-label`), Input, SearchInput, Select, Field, SegmentedControl, Tabs (synchronisés `?tab=`), Switch, Checkbox, Badge (type, méthode, vue/clic, statut, NEW), Chip (filtre retirable), Card (un seul en-tête : titre 13/600, sous-titre, action, pied), StatTile + StatStrip (sélectionnable), BarList, DataTable (tri, pagination serveur, ligne = lien), Art (poster 2:3 / thumb 16:9 / square, 3 tailles, repli dégradé + initiales), Avatar (20/28/56, couleur stable par id), AvatarStack, Meter, SegBar, Breadcrumbs, PageHeader, EntityHeader, EmptyState, ErrorState, Skeleton, Menu, Popover, Tooltip, Dialog/ConfirmDialog, Toast, PeriodPicker (seul contrôle de temps, dans la barre du haut), FilterBar, CommandPalette (⌘K), SidePanel, SessionCard, UserCompare, StreakCard, ChartFrame (titre, légende, bascule graphique/tableau, export CSV).

Règles : tout lien est un vrai `<a href>` (via `<Link>`), conservant `?range`/`?from&to` ; focus visible partout ; aucun élément cliquable sans rôle ni clavier.

## 6. Architecture & URL

Navigation (sidebar) : **Observer** — Vue d'ensemble `/`, En direct `/live`, Historique `/history` · **Explorer** — Bibliothèques `/libraries`, Utilisateurs `/users` · **Comprendre** — Insights `/insights`, Découverte `/discover`, Lecture & transcodage `/playback` · pied : Réglages `/settings/:section`, utilisateur, thème, langue. Recherche globale ⌘K.

| Route | Page | Paramètres |
|---|---|---|
| `/` | Vue d'ensemble | `range`, `compare`, `metric=time|views|clicks` |
| `/live` | En direct | — |
| `/history` | Historique | `view=table|calendar`, `user`, `type`, `client`, `device`, `method`, `views=1`, `q`, `page`, `month` |
| `/libraries`, `/libraries/:libraryId` | Bibliothèques / Bibliothèque | `tab=items|stats|activity`, `q`, `layout=grid|table` |
| `/movies/:id` | Film | `tab` |
| `/series/:id` | Série | `season`, `tab=episodes|viewers|history` |
| `/series/:id/episodes/:episodeId` | Épisode | — |
| `/artists/:id`, `/albums/:id` | Artiste / Album | `track` (panneau latéral) |
| `/users`, `/users/:id` | Utilisateurs / Utilisateur | `tab=overview|history|taste`, `compare=:otherId` |
| `/users/:id/recap/:period` | Récap (`2026`, `2026-11`) | — |
| `/insights`, `/discover`, `/playback` | Analyses | `range` |
| `/settings/:section` | Réglages (config, tasks, backup, logs, webhooks, apikeys, about) | — |
| `/login` | Connexion + assistant de configuration | — |

Redirections permanentes : `/statistics → /insights`, `/activity → /history`, `/activity/timeline → /history?view=calendar`, `/items/:id` et `/libraries/:lib/items/:id` → route typée (résolution du type via `GET /api/items/:id/route`), `/libraries/:lib/albums/:id → /albums/:id`, `/settings?view=x → /settings/x`, `/about → /settings/about`, `/setup → /login`. Route 404 dédiée.

Paramètres globaux conservés d'une page à l'autre : `range=7d|30d|90d|mtd|ytd|all` (défaut 30d) ou `from`/`to`, `compare=1`. Toutes les pages, y compris les pages de détail, respectent la période. Fil d'Ariane sur toutes les pages hors niveau 1 ; plus aucun `navigate(-1)`.

Décisions d'entités : épisode = page légère ; piste = lignes + panneau latéral (`?track=`) ; saison = filtre `?season=` ; artiste = vraie page ; album = une seule page ; client/appareil = filtres de l'historique ; Statistiques supprimée (contenu propre → Insights) ; Activité + Timeline fusionnées.

## 7. Pages et widgets

Chaque widget indique l'endpoint (existant ou **nouveau**, cf. §8). Toutes les requêtes portent `from`, `to` (dates locales, fin exclusive) et les filtres de page.

- **Vue d'ensemble** : StatStrip sélectionnable (temps, vues, clics, utilisateurs actifs, complétion) pilotant le graphique principal avec période précédente ; Top contenus (onglets type) ; En direct (2 sessions) ; Utilisateurs ; Par type + méthode (SegBar) ; Jour × heure ; **NEW** pic de flux simultanés ; **NEW** « Ce jour-là ».
- **En direct** : tuiles (lectures, transcodages, débit, pic du jour) ; SessionCards liées ; lectures simultanées du jour ; **NEW** frise de la soirée ; **NEW** débit sortant.
- **Historique** : FilterBar (URL) ; vue tableau paginée côté serveur (date, utilisateur, contenu, regardé, complétion, appareil, méthode, vue/clic) ; vue calendrier (mois dans l'URL) ; export CSV.
- **Bibliothèques / Bibliothèque** : cartes ; tuiles ; contenus (grille/tableau, sparkline, rétention pour les séries) ; stats (genres, méthode, temps jusqu'au visionnage, jamais vus) ; activité ; **NEW** décennies vs catalogue ; **NEW** croissance vs consommation.
- **Film** : en-tête, tuiles, entonnoir de complétion, positions d'arrêt, qui a regardé, historique ; **NEW** séances pour finir.
- **Série** : en-tête, tuiles (dont rétention, binges), rétention par épisode, onglets saisons, tableau d'épisodes (vues, spectateurs, complétion, abandons) ; spectateurs ; historique (avec nom d'épisode) ; **NEW** cadence entre épisodes ; **NEW** délai sortie → visionnage.
- **Épisode** : fil d'Ariane série › saison › épisode, précédent/suivant, tuiles, entonnoir, positions d'arrêt, qui a regardé.
- **Artiste** : tuiles (écoutes, temps, auditeurs, skip), top pistes (vues, clics, skip, auditeurs), albums ; **NEW** skip selon l'heure.
- **Album** : pistes avec skip, panneau latéral de piste ; **NEW** albums écoutés en entier.
- **Utilisateurs** : tableau (temps, vues, répartition par type, dernière activité, tendance) ; **NEW** rétention ; **NEW** part de chacun dans le temps.
- **Utilisateur** : onglets aperçu/historique/goûts ; tuiles ; calendrier 12 mois ; en cours ; horloge ; profil (régularité, découverte, concentration, complétion) ; goûts (genres par type) ; **NEW** comparaison avec un autre utilisateur ; **NEW** appareils & versions ; **NEW** VO/VF et sous-titres ; série en cours & records.
- **Récap** : chiffre héros, top séries/films/artistes, par mois, records ; **NEW** découverte d'artistes ; **NEW** classement avec évolution.
- **Insights** : distribution de complétion, complétion par type, binges, pauses, rythme hebdo ; **NEW** durée des séances, reprises vs début, revisionnages, genres dans le temps, moments de la journée par type.
- **Découverte** : jamais vus, abandonnés, ajoutés ce mois ; **NEW** stockage vs visionnage.
- **Lecture & transcodage** : tuiles, méthode par jour (rampe ordinale), clients (% transcodé), conversions ; **NEW** faux départs par client, résolution servie, HDR, raisons du transcodage, matrice codecs, vidéo vs audio seul, local/distant + lieux.
- **Réglages** : mêmes fonctions qu'aujourd'hui (config Jellyfin, tâches + cron, sauvegardes, journaux, webhooks Discord/HTTP, clés API, à propos), restylées ; le choix de palette disparaît (un seul accent) ; thème et langue dans le menu utilisateur.
- **Connexion** : écran unique + assistant de configuration (host, clé API), `deviceId` par navigateur conservé.

## 8. Backend

### 8.1 Contrat commun des endpoints de stats
Paramètres : `from`, `to` (dates locales, fin exclusive, interprétées dans `TZ`), `compare=1` (ajoute la période précédente de même durée), `userId`, `libraryId`, `type=movie|series|music`. Réponses JSON typées, chiffres bruts (secondes, comptes) — le formatage est côté front. Les endpoints existants utilisés gardent leur forme ; les nouveaux vivent sous `/api/v2/…` avec ce contrat, et le front n'utilise que `/api/v2` + les endpoints de configuration/tâches existants.

### 8.2 Nouveaux endpoints (`/api/v2`)
`overview` (tuiles + séries quotidiennes + précédente) · `concurrency` (pic quotidien, frise d'une journée) · `on-this-day` · `history` (paginé, filtres, tri, CSV) · `history/calendar` · `items/:id/route` · `movies/:id` · `series/:id` (+ `episodes`, rétention, cadence, délai sortie→vue) · `episodes/:id` · `artists/:id` · `albums/:id` (+ pistes, écoutes complètes) · `tracks/:id` (panneau) · `users/:id` (profil, calendrier, en cours, horloge, appareils, langues) · `users/compare` · `users/retention` · `users/share` · `recap/:userId/:period` · `insights/*` (complétion, sessions, reprises, revisionnages, genres dans le temps, moments par type, pauses, binges) · `discover/*` (jamais vus, abandonnés, stockage vs vues, décennies, croissance) · `playback/*` (méthode par jour, clients, faux départs, résolution, HDR, raisons, matrice codecs, vidéo/audio, local/distant, lieux) · `funnel` et `stop-positions` (film/épisode) · `search` (⌘K).

### 8.3 Capture tracker (migration 011)
Colonnes sur `jf_activity_watchdog` et `jf_playback_activity` : `AudioStreamIndex`, `AudioLanguage`, `SubtitleStreamIndex`, `SubtitleLanguage`, `TranscodeReasons` (jsonb), `Bitrate`, `HwAccel`, `VideoRange`, `PauseCount`, `IsRemote`. Le tracker les renseigne depuis `PlayState`, `TranscodingInfo`, `NowPlayingItem.MediaStreams` et `RemoteEndPoint` (privé RFC1918 = local) ; `PauseCount` compte les transitions vers pause. Géolocalisation : table `ip_geo_cache` alimentée en tâche de fond via l'endpoint existant. Tests unitaires et d'intégration comme pour le tracker actuel.

## 9. Bugs et incohérences à corriger (issus des audits)

Palette non appliquée à l'UI ; 8 radius ; 4 en-têtes de carte ; sélecteurs de période dupliqués et non reflétés dans l'URL ; tailles de texte hors échelle et police jamais chargée ; mode clair cassé (fonds `rgba(255,255,255,…)`, item actif sidebar) ; 8+ tailles de posters ; 4 formats de durée, 3 de date, semaine dimanche/lundi ; états vides ad hoc ; tuiles et lignes de classement hétérogènes ; 4 avatars ; contrôles de tailles variées ; élévations incohérentes ; ~20 éléments cliquables non accessibles ; double tranche Musique/Audio dans le camembert ; Historique qui recharge tout l'historique toutes les 15 s ; tuiles « tout temps » au-dessus de graphiques filtrés ; pages de détail qui ignorent la période ; culs-de-sac (sessions live, « qui a regardé », graphiques, appareils) ; `navigate(-1)` ; deux pages album ; série sans liste d'épisodes ; épisode sans lien vers sa série ; dates envoyées en UTC (`toISOString`) ; 20 clés i18n manquantes et ~44 valeurs par défaut françaises inline ; chaînes en dur ; `lang="en"` statique ; `alt="Jellyfin"` sur le logo ; `/setup` inatteignable ; code mort (`GenreRadarChart`, `stats.go`, `useFetchWithDays`).

## 10. Accessibilité & responsive

Contraste AA pour le texte ; focus visible ; navigation clavier complète (menus Radix) ; `aria-current` dans la navigation ; graphiques avec `role="img"`, nom accessible et vue tableau ; infobulles aussi au focus. Responsive : sidebar repliable sous 900 px (tiroir), grilles 1 colonne sous 640 px, tableaux défilables dans leur carte, aucun débordement horizontal de page.

## 11. Découpage

1. **Fondations** : MUI isolé dans un enveloppeur « legacy » (les pages pas encore réécrites restent fonctionnelles dans le nouveau shell), tokens, `ui/`, `charts/`, `format.ts`, `useUrlState`, client API + React Query, AppShell (sidebar, barre du haut, PeriodPicker, fil d'Ariane, ⌘K squelette), routes + redirections + 404, connexion. Tests.
2. **Backend v2 (socle)** : contrat commun, `overview`, `history`, `items/:id/route`, `search`, entités (`movies`, `series`, `episodes`, `artists`, `albums`, `tracks`, `users`). Tests Go.
3. **Pages cœur** : Vue d'ensemble, En direct, Historique, Utilisateurs, Utilisateur.
4. **Pages d'entité** : Bibliothèques, Bibliothèque, Film, Série, Épisode, Artiste, Album.
5. **Analyses** : Insights, Découverte, Lecture & transcodage, Récap + endpoints `insights/discover/playback/recap/concurrency/funnel/stop-positions`.
6. **Capture tracker** (migration 011) + vues qui en dépendent (VO/VF, raisons, débit, HDR, local/distant, pauses).
7. **Réglages** restylés + retrait complet de MUI et nettoyage final (dépendances, code mort, i18n complet) + passe Playwright complète.

Chaque étape laisse l'app fonctionnelle et testée ; les commits suivent les Conventional Commits sans attribution.
