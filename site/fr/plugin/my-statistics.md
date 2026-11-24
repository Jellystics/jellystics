# Page « Mes statistiques »

Chaque utilisateur Jellyfin dispose d'une entrée **Mes statistiques** dans le menu de son avatar (la deuxième, après *Profil*). Elle ouvre `/Jellystics/Me/` sur le serveur Jellyfin et n'affiche que les statistiques de cet utilisateur.

![Mes statistiques](/screenshots/fr/me.jpg){.shot}

## Ce que voient les utilisateurs {#what-users-see}

- Temps de visionnage, vues, clics, vues terminées, complétion moyenne et série en cours, comparés à la période précédente.
- Période : 7 jours, 30 jours, 90 jours, 1 an ou depuis le début.
- Temps de visionnage, vues ou clics par jour, contenus préférés, répartition par type de média et méthode de lecture, heatmap jour de la semaine × heure, chaque lecture sous forme de point, appareils et applications, et leur historique complet.

La page suit la langue Jellyfin de l'utilisateur (anglais, français ou espagnol) et son thème. Si Jellystics n'a pas encore synchronisé le compte, elle affiche « Tes statistiques arrivent ».

## Fonctionnement {#how-it-works}

```txt
Browser ──► Jellyfin /Jellystics/Me/…  ──► Jellystics /me/…        (page and assets)
Browser ──► Jellyfin /Jellystics/Me/api/… ──► Jellystics /api/me/… (statistics, with the secret)
```

1. Le plugin ajoute un petit script (`/Jellystics/Me/client.js`) au `index.html` de jellyfin-web. Ce script ajoute l'entrée de menu, avec un libellé dans la langue de la page.
2. Le navigateur **ne parle qu'à Jellyfin**. Les appels de statistiques nécessitent une connexion Jellyfin.
3. Le plugin les transmet à Jellystics avec deux en-têtes : `X-Jellystics-Secret` (le secret partagé) et `X-Jellystics-User` (l'identifiant de l'utilisateur, tiré de l'authentification Jellyfin et jamais du navigateur). Un utilisateur ne peut donc lire que ses propres données.
4. Jellystics ne répond qu'au bon secret et à un utilisateur connu.

Les utilisateurs n'ont jamais besoin d'un compte Jellystics, et Jellystics n'a pas besoin d'être exposé sur Internet : seul le serveur Jellyfin doit pouvoir le joindre.

## Prérequis {#requirements}

- Le plugin est relié : *Tableau de bord → Extensions → Jellystics* indique « Linked … reachable ».
- Jellyfin peut joindre Jellystics. Si l'adresse détectée automatiquement est incorrecte, définissez `JELLYSTICS_PUBLIC_URL` dans Jellystics ou le **URL override** dans les réglages du plugin.
- Le tableau de bord est activé (`DISABLE_DASHBOARD` ne vaut pas `true`).

## Dépannage {#troubleshooting}

| La page affiche | Cause | Solution |
|---|---|---|
| Statistiques indisponibles (pas relié) | Jellystics ne s'est pas encore relié. | Vérifiez la carte du plugin dans Jellystics (*Actif*), attendez 30 s. |
| Statistiques indisponibles (injoignable) | Jellyfin n'arrive pas à joindre l'URL de Jellystics. | Renseignez le URL override ou `JELLYSTICS_PUBLIC_URL`. |
| Statistiques indisponibles (reconnexion) | Le secret a été rejeté ; Jellystics se relie à nouveau en moins d'une minute. | Assurez-vous qu'une seule instance de Jellystics cible ce Jellyfin. |
| Connecte-toi à Jellyfin | Aucune session Jellyfin dans ce navigateur. | Connectez-vous d'abord à Jellyfin. |
| L'entrée de menu est absente | Le navigateur a mis en cache l'ancien `index.html`. | Rechargez sans le cache ; vérifiez que le plugin est activé. |

## Sécurité {#security}

La page et ses scripts proviennent de Jellystics mais s'exécutent sur l'origine Jellyfin, où ils pourraient utiliser la session Jellyfin de l'utilisateur connecté. Jellyfin accorde donc à l'instance Jellystics à laquelle il est relié autant de confiance qu'à son propre client web. Faites tourner Jellystics sur un réseau local ou Docker de confiance, ou donnez-lui une URL `https`.

Protections dans le plugin :

- seuls la page, `assets/**`, `locales/**/*.json` et les images à la racine sont relayés, chacun avec un type de contenu autorisé ;
- Content-Security-Policy stricte sur la page (`script-src 'self'`, `connect-src 'self'`, `frame-ancestors 'self'`), `X-Content-Type-Options: nosniff`, `Referrer-Policy: same-origin` ;
- aucune redirection suivie, délai d'expiration de 15 s, taille limitée à 10 Mo ;
- les statistiques ne sont jamais mises en cache (`Cache-Control: private, no-store`) ;
- les paramètres de requête comme `userId` ou `api_key` sont supprimés avant la transmission ;
- le secret ne se trouve que dans le XML du plugin et n'est jamais renvoyé par une API.
