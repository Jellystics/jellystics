# Glosario

Definiciones usadas en todo Jellystics (panel, «Mis estadísticas» y Prometheus). Los días, las horas y los días de la semana son locales a la zona horaria `TZ`.

| Término | Definición |
|---|---|
| **Clic** | Una reproducción iniciada, sea cual sea su duración. Un clic que nunca llegó a 30 s muestra la etiqueta *Clic*. |
| **Vista** | Una reproducción con al menos **30 segundos** realmente vistos (pausas excluidas). |
| **Tiempo de reproducción** | Suma del tiempo visto de todas las reproducciones, clics incluidos. |
| **Usuario activo** | Un usuario con al menos una reproducción (clic) en el periodo. |
| **Finalización** | `max(posición de parada, tiempo visto) ÷ duración`, con un máximo del 100 %. Se cuenta una vez por usuario, título y día: ver un episodio en dos veces el mismo día cuenta como una unidad. La *finalización media* es la media de estas unidades. |
| **Vista terminada** | Finalización ≥ **90 %**. Para películas y episodios de 5 min o más registrados por el plugin, decide en su lugar el indicador «reproducido hasta el final» de Jellyfin. Las reproducciones aún en curso nunca cuentan. |
| **Abandonado** (finalización) | Una unidad de finalización por debajo del 90 % (Prometheus `jellyfin_abandoned_plays_total`, «abandonos» en los episodios). |
| **Abandonado** (Descubrir) | Un título reproducido por última vez hace más de **60 días** y dejado entre el **10 % y el 80 %** (películas), o con el 10–80 % de sus episodios vistos (series). |
| **Mitad alcanzada** | Etapa del embudo: finalización ≥ 50 %. |
| **Falso inicio** | Un clic que nunca llegó a ser una vista (detenido antes de 30 s). |
| **Tasa de saltos** (música) | Clics que nunca llegaron a ser una escucha ÷ clics. Se muestra en rojo por encima del 30 %. |
| **Maratón** | Al menos **3 episodios distintos** de la misma serie por el mismo usuario el mismo día. |
| **Reanudada** | Una vista que empezó a **30 s o más** del inicio del título. |
| **Revisionado** | El mismo título visto por la misma persona en dos o más días distintos (pistas: dos o más vistas). |
| **Retención** (series) | Espectadores del último episodio normal ÷ espectadores del primero (especiales excluidos). |
| **Retención** (usuarios) | Cohortes semanales: usuarios activos en una semana que vuelven a estar activos de 1 a 4 semanas después (últimas 8 semanas completas). |
| **Racha** | Días consecutivos con al menos una reproducción. |
| **Reproducciones simultáneas** | Número máximo de reproducciones a la vez, en franjas de 15 minutos. |
| **Tiempo hasta verlo** | Días entre la fecha en que se añadió un elemento y su primera vista. |
| **Salto corto al inicio** (intro) | Solo con el plugin: un salto hacia delante de 10 s al 15 % de la duración (como máximo 3 min) en una reproducción empezada desde el principio. Es una estimación. |
| **Método de reproducción** | *Direct play* (archivo enviado tal cual), *Direct stream* (remultiplexado, códecs sin cambios), *Transcodificación* (recodificado). |
| **Local / remoto** | Según la IP del cliente: las direcciones privadas (RFC 1918, ULA), de loopback y de enlace local son locales. CGNAT (100.64.0.0/10, p. ej. Tailscale) cuenta como remoto. |
| **VO o doblaje** | *VO* = se reprodujo la primera pista de audio del archivo. |
| **Dispositivo desactualizado** | Una versión de la aplicación más antigua que la versión más reciente vista para el mismo cliente. |
| **Puntuaciones de perfil** | Regularidad (días activos), Descubrimiento (primeras vistas), Concentración (parte de los 3 títulos principales). |
