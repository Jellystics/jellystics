# Resumen, en directo e historial

## Navegación {#navigation}

- La barra lateral agrupa las páginas en **Observar** (Resumen, En directo, Historial), **Explorar** (Bibliotecas, Usuarios) y **Comprender** (Análisis, Descubrir, Reproducción). En directo muestra un punto rojo cuando hay algo reproduciéndose.
- El **selector de periodo** (barra superior) ofrece Hoy, Últimos 7/30/90 días, Mes en curso, Año en curso, Todo y un rango personalizado. *Comparar con el período anterior* añade variaciones a cada KPI (% relativo para los recuentos, puntos para las proporciones).
- El periodo, los filtros, las pestañas y el orden se guardan en la URL: cualquier vista se puede añadir a marcadores o compartir.
- **`⌘K` / `Ctrl+K`** abre la paleta de comandos: busca películas, series, episodios, álbumes, artistas, pistas y usuarios, salta a cualquier página, lanza una sincronización o cambia el tema.
- El **menú de usuario** contiene el tema, el idioma, los ajustes y *Cerrar sesión*.
- Cada gráfico tiene un botón **CSV** que descarga sus datos.

## Resumen {#overview}

![Resumen](/screenshots/es/overview.jpg){.shot}

- **KPI**: Tiempo de reproducción, Vistas, Clics (cada uno con un minigráfico; haz clic en uno para elegir lo que muestra el gráfico principal), Usuarios activos, Finalización media.
- **{Métrica} por día**: la métrica seleccionada a lo largo del periodo; con la comparación, también se dibuja el periodo anterior.
- **Todas las reproducciones**: un punto por reproducción (hora de inicio × minutos vistos, coloreado por tipo); haz clic en un punto para abrir el título.
- **Reproducciones por biblioteca**: vistas por día (o semana) y biblioteca.
- **Contenido más visto** (todo, películas, series, música), **En directo** (dos sesiones actuales), **Usuarios** (los 8 primeros por tiempo de reproducción).
- **Por tipo**: parte del tiempo de reproducción por tipo y después por método de reproducción.
- **Cuándo**: mapa de calor día de la semana × hora. **Pico de flujos simultáneos** por día. **Tal día como hoy**: lo que se vio en esta fecha en años anteriores.

Con *Comparar con el período anterior*:

![Resumen con comparación](/screenshots/es/overview-compare.jpg){.shot}

## En directo {#live}

![En directo](/screenshots/es/live.jpg){.shot}

Se actualiza en tiempo real por WebSocket.

- **En reproducción**, **Transcodificaciones** (con el número de aceleradas por hardware), **Tasa de salida** (suma de los flujos activos), **Pico del día**.
- Una tarjeta por sesión: carátula, título, usuario, dispositivo, método de reproducción, etiqueta de pausa y progreso.
- **Tasa de salida en directo** por flujo, **Reproducciones simultáneas** del día en franjas de 15 minutos y **Cronología del día**: quién vio qué y cuándo se solaparon los flujos.

## Historial {#history}

![Tabla del historial](/screenshots/es/history.jpg){.shot}

- **Filtros**: texto libre, tipo (películas, series, música), *Solo vistas*, usuario, cliente, dispositivo, método de reproducción. Haz clic en una celda de cliente o de dispositivo para filtrar por ella.
- **Tabla**: fecha, usuario, contenido, visto, finalización, dispositivo, método de reproducción y etiqueta Vista/Clic. Ordena por fecha, tiempo visto o finalización; de 25 a 200 filas por página.
- **Exportar** descarga todo el historial filtrado en CSV, generado por el servidor.

La vista **Calendario** muestra cada día del mes con su número de vistas y los primeros títulos:

![Calendario del historial](/screenshots/es/history-calendar.jpg){.shot}
