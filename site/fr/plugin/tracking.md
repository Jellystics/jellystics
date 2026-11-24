# Mesure des lectures

Le plugin transforme les événements de lecture de Jellyfin en un enregistrement par lecture. Cette page décrit les règles exactes.

## Quels événements comptent {#which-events-count}

Le plugin écoute `PlaybackStart`, `PlaybackProgress` et `PlaybackStopped`. Un événement est ignoré quand :

- il n'a pas de contenu ou pas d'utilisateur ;
- il est **automatique** (progression générée par le serveur, non signalée par un client) ;
- le contenu est un **bonus** (bande-annonce, featurette…) ou un **média de thème**.

## Une lecture, une clé {#one-play-one-key}

Une lecture est identifiée par son `PlaySessionId`, ou par `DeviceId | UserId | ItemId` quand le client n'en envoie pas. Un *Stop* sans identifiant de session est rattaché à la lecture ouverte du même appareil, utilisateur et contenu.

| Situation | Résultat |
|---|---|
| *Start* pour une nouvelle clé | Une nouvelle lecture s'ouvre immédiatement (sans délai d'attente). |
| *Start* sur un appareil où le même utilisateur a une autre lecture ouverte | La lecture précédente est clôturée à cet instant (lecture automatique de l'épisode suivant, par exemple). |
| Nouveau *Start* pour la même clé et le même contenu | Traité comme une progression : la lecture n'est pas scindée. |
| Événement pour une clé clôturée il y a moins de 2 min | Ignoré (événements tardifs). |
| Aucun événement pendant **5 min** en lecture | La lecture est clôturée à son dernier événement. |
| Aucun événement pendant **60 min** en pause | La lecture est clôturée à son dernier événement. |

## La machine à états {#the-state-machine}

Chaque événement est appliqué dans l'ordre d'arrivée. Avec `dt` = temps réel depuis l'événement précédent et `dp` = variation de la position :

```txt
if the player was playing:
    WatchedMs += dp unknown ? dt : min(dt, max(dp, 0) + 2 s)
if the player was paused:
    PausedMs  += dt

expected = real time since the last known position while playing, 0 while paused
if dp > expected + 5 s:   SeekCount += 1, SkippedMs += dp - expected   # saut en avant
if dp < -5 s:             SeekCount += 1                               # saut en arrière

playing -> paused (not on Stop): PauseCount += 1
WatchedMs is always capped at the real time since the start.
```

Conséquences :

- **le temps regardé ne dépasse jamais le temps réel**, ni la distance parcourue par la position (+2 s de marge) ;
- une pause, une mise en mémoire tampon ou un retour en arrière n'ajoutent rien ;
- un saut en avant est compté avec le temps sauté (c'est ainsi que les génériques passés sont détectés) ;
- un client qui se met en pause juste avant de s'arrêter n'ajoute pas de pause.

## Pistes et transcodage {#tracks-and-transcoding}

- **Les pistes audio et sous-titres** sont lues au démarrage et à chaque événement de progression. Chaque changement est consigné dans `StreamChanges` (`at`, `kind`, `index`, `language`). Le temps regardé est aussi compté par piste audio : à la fin, la piste regardée le plus longtemps devient la piste audio et la langue de la lecture.
- **Le transcodage** (issu de `TranscodingInfo` de Jellyfin), rafraîchi à chaque progression : méthode de lecture, codecs vidéo source et cible, codecs audio source et cible, indicateurs de vidéo et d'audio directs, largeur, hauteur, débit et fréquence d'images en sortie, accélération matérielle, raisons du transcodage, plage vidéo (SDR, HDR10, Dolby Vision…) et résolution source.
- **Le réseau** : `IsRemote` d'après l'IP du client. Les adresses de bouclage, les plages IPv4 privées (10/8, 172.16/12, 192.168/16), le link-local, l'IPv6 ULA et le site-local sont locaux ; tout le reste est à distance ; inconnu en l'absence d'adresse.

## Fin d'une lecture {#end-of-a-play}

- Sur *Stop* : l'heure de fin, la position finale et l'indicateur `PlayedToCompletion` de Jellyfin sont enregistrés.
- Quand la lecture est clôturée par les délais d'expiration ci-dessus ou au démarrage, `PlayedToCompletion` reste inconnu et la fin correspond au dernier événement.

## Sauvegarde et plantages {#saving-and-crashes}

- Les lectures ouvertes sont sauvegardées toutes les **15 secondes** (point de sauvegarde) et les lectures clôturées immédiatement.
- Lors d'un arrêt propre, les lectures en cours sont sauvegardées comme ouvertes.
- Au démarrage, chaque lecture encore ouverte est clôturée à son dernier point de sauvegarde : **environ 15 secondes au plus sont perdues** après un plantage.
- Les écritures passent par une file unique en arrière-plan ; une action est retentée une fois si elle échoue.

## Bibliothèque, marques et connexions {#library-marks-and-logins}

Voir [Données collectées](/fr/plugin/data#library-events).
