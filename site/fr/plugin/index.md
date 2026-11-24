# Plugin Jellystics pour Jellyfin

Le plugin Jellystics tourne dans Jellyfin. Il enregistre chaque lecture à partir des événements du serveur lui-même, conserve l'historique complet dans sa propre base de données et relaie la page « Mes statistiques » à chaque utilisateur Jellyfin.

- **Précis** : vrai temps de lecture, pauses, sauts et temps sauté, pistes audio et sous-titres et leurs changements, détails complets du transcodage, « lu jusqu'à la fin » fourni par Jellyfin.
- **Complet** : ajouts et suppressions dans les bibliothèques, connexions (y compris échouées), marques vu et favori avec leur raison.
- **Durable** : tout est conservé dans `jellystics.db` sans limite de durée ; un plantage fait perdre au plus 15 secondes de la lecture en cours.
- **Léger** : les événements sont traités en mémoire et écrits par un unique processus d'écriture en arrière-plan, si bien que Jellyfin n'est jamais ralenti.
- **Indépendant** : Jellystics fonctionne sans lui et repasse sur son propre suivi si le plugin cesse de répondre.

Il est basé sur le plugin officiel Playback Reporting (GPL-3), avec une nouvelle identité : les deux plugins peuvent être installés côte à côte, et le plugin Jellystics peut importer l'historique de Playback Reporting.

## Prérequis {#requirements}

- Jellyfin **10.11** (`targetAbi 10.11.0.0`, .NET 9).
- Jellystics joignable depuis Jellyfin en HTTP, si vous voulez la page « Mes statistiques ».

## Installation {#install}

1. Dans Jellyfin, ouvrez **Tableau de bord → Extensions → Dépôts** et ajoutez :

   ```txt
   https://raw.githubusercontent.com/Jellystics/jellystics-plugin-playbackreporting/master/manifest.json
   ```

2. Ouvrez le **Catalogue**, installez **Jellystics**, puis **redémarrez Jellyfin**.
3. Vérifiez que *Jellystics* apparaît dans **Tableau de bord → Extensions** et dans le menu du tableau de bord (section serveur).

Rien d'autre n'est nécessaire côté Jellystics : en moins de 30 secondes, *Réglages → Configuration → Plugin Jellystics* passe à **Actif**, et Jellystics relie automatiquement la page « Mes statistiques ».

::: tip Installation manuelle
Téléchargez `Jellyfin.Plugin.Jellystics-v<version>.zip` depuis les [releases](https://github.com/Jellystics/jellystics-plugin-playbackreporting/releases), décompressez `Jellyfin.Plugin.Jellystics.dll` dans `<jellyfin config>/plugins/Jellystics/` et redémarrez Jellyfin.
:::

## Premiers pas après l'installation {#first-steps-after-installing}

1. **Importer l'ancien historique** (facultatif) : si vous utilisiez le plugin officiel Playback Reporting, lancez *Import history* dans les [réglages du plugin](/fr/plugin/configuration#import-history).
2. **Envoyer l'historique de Jellystics** (facultatif) : dans Jellystics, *Réglages → Configuration → Plugin Jellystics → Sauvegarder l'historique dans le plugin*. Le plugin détient alors une sauvegarde complète de tout.
3. **Vérifier « Mes statistiques »** : ouvrez le menu de l'avatar dans Jellyfin ; *Mes statistiques* est la deuxième entrée.

## Mise à jour {#updating}

Jellyfin affiche les mises à jour du dépôt dans **Tableau de bord → Extensions**. Chaque release ajoute une entrée à `manifest.json` avec sa somme de contrôle ; installez-la et redémarrez Jellyfin.

## Désinstallation {#uninstalling}

Supprimez le plugin depuis **Tableau de bord → Extensions** et redémarrez Jellyfin. Jellystics détecte sa disparition et continue d'enregistrer avec son suivi. La base du plugin (`<jellyfin config>/plugins/Jellyfin.Plugin.Jellystics/jellystics.db`) reste sur le disque.

## En savoir plus {#learn-more}

- [Réglages du plugin](/fr/plugin/configuration)
- [Mesure des lectures](/fr/plugin/tracking)
- [Données collectées](/fr/plugin/data)
- [Synchronisation avec Jellystics](/fr/plugin/sync)
- [Page « Mes statistiques »](/fr/plugin/my-statistics)
- [API du plugin](/fr/plugin/api)
