# Cómo se miden las reproducciones

El plugin convierte los eventos de reproducción de Jellyfin en un registro por reproducción. Esta página describe las reglas exactas.

## Qué eventos cuentan {#which-events-count}

El plugin escucha `PlaybackStart`, `PlaybackProgress` y `PlaybackStopped`. Un evento se ignora cuando:

- no tiene elemento o no tiene usuario;
- es **automático** (progreso generado por el servidor, no comunicado por un cliente);
- el elemento es un **extra** (tráiler, featurette…) o un **medio temático**.

## Una reproducción, una clave {#one-play-one-key}

Una reproducción se identifica por su `PlaySessionId`, o por `DeviceId | UserId | ItemId` cuando el cliente no envía ninguno. Un *Stop* sin id de sesión se asocia a la reproducción abierta del mismo dispositivo, usuario y elemento.

| Situación | Resultado |
|---|---|
| *Start* para una clave nueva | Se abre una nueva reproducción de inmediato (sin tiempo de espera). |
| *Start* en un dispositivo donde el mismo usuario tiene otra reproducción abierta | La reproducción anterior se cierra en ese momento (reproducción automática del siguiente episodio, por ejemplo). |
| *Start* de nuevo para la misma clave y el mismo elemento | Se trata como progreso: la reproducción no se divide. |
| Evento para una clave cerrada hace menos de 2 min | Se ignora (eventos tardíos). |
| Ningún evento durante **5 min** mientras se reproduce | La reproducción se cierra en su último evento. |
| Ningún evento durante **60 min** mientras está en pausa | La reproducción se cierra en su último evento. |

## La máquina de estados {#the-state-machine}

Cada evento se aplica en orden de llegada. Con `dt` = tiempo real desde el evento anterior y `dp` = cambio de posición:

```txt
if the player was playing:
    WatchedMs += dp unknown ? dt : min(dt, max(dp, 0) + 2 s)
if the player was paused:
    PausedMs  += dt

expected = real time since the last known position while playing, 0 while paused
if dp > expected + 5 s:   SeekCount += 1, SkippedMs += dp - expected   # salto hacia delante
if dp < -5 s:             SeekCount += 1                               # salto hacia atrás

playing -> paused (not on Stop): PauseCount += 1
WatchedMs is always capped at the real time since the start.
```

Consecuencias:

- **el tiempo visto nunca supera el tiempo real**, ni lo que ha avanzado la posición (+2 s de margen);
- una pausa, el búfer o un salto hacia atrás no suman nada;
- un salto hacia delante se cuenta con el tiempo saltado (así se detecta que se salta la intro);
- un cliente que pausa justo antes de detenerse no suma una pausa.

## Pistas y transcodificación {#tracks-and-transcoding}

- **Las pistas de audio y subtítulos** se leen al inicio y en cada evento de progreso. Cada cambio se registra en `StreamChanges` (`at`, `kind`, `index`, `language`). El tiempo visto también se cuenta por pista de audio: al final, la pista vista durante más tiempo pasa a ser la pista de audio y el idioma de la reproducción.
- **La transcodificación** (a partir de `TranscodingInfo` de Jellyfin), actualizada en cada progreso: método de reproducción, códecs de vídeo de origen y destino, códecs de audio de origen y destino, indicadores de vídeo y audio directos, anchura, altura, tasa de bits y fotogramas por segundo de salida, aceleración por hardware, motivos de transcodificación, rango de vídeo (SDR, HDR10, Dolby Vision…) y resolución de origen.
- **La red**: `IsRemote` según la IP del cliente. Loopback, rangos IPv4 privados (10/8, 172.16/12, 192.168/16), enlace local, IPv6 ULA y site-local son locales; todo lo demás es remoto; desconocido cuando no hay dirección.

## Fin de una reproducción {#end-of-a-play}

- En un *Stop*: se registran la hora de fin, la posición final y el indicador `PlayedToCompletion` de Jellyfin.
- Cuando la reproducción se cierra por los tiempos de espera anteriores o al arrancar, `PlayedToCompletion` queda desconocido y el fin es el último evento.

## Guardado y fallos {#saving-and-crashes}

- Las reproducciones abiertas se guardan cada **15 segundos** (punto de control) y las cerradas de inmediato.
- En un apagado limpio, las reproducciones en curso se guardan como abiertas.
- Al arrancar, cada reproducción aún abierta se cierra en su último punto de control: **como máximo se pierden unos 15 segundos** tras un fallo.
- Las escrituras pasan por una única cola en segundo plano; una acción se reintenta una vez si falla.

## Biblioteca, marcas e inicios de sesión {#library-marks-and-logins}

Consulta [Datos recopilados](/es/plugin/data#library-events).
