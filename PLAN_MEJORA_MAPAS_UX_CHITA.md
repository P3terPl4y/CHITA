# Diagnóstico y plan de mejora de mapas de CHITA

## Diagnóstico observado

Revisé el selector de direcciones, el mapa del trabajo, el mapa de repartidores, el panel de cuenta y la administración. Ejecuté los escenarios de Firefox que verifican contención en escritorio y móvil, zoom, desplazamiento, cambio de tamaño, selección de punto, dirección devuelta y respuestas de geocodificación fuera de orden. El mapa de edición de cuenta se inspeccionó también visualmente en Firefox.

La base funcional es correcta: el pin puede colocarse y arrastrarse, la petición inversa no permite que una respuesta antigua reemplace al último punto, los mapas no desbordan el panel y guardar la ubicación está separado de elegirla. Quedan estos problemas de experiencia:

1. Los controles de zoom que añade Leaflet se muestran en inglés, aunque el producto y el resto de la interfaz están en español.
2. El selector no explica con suficiente claridad la diferencia entre mover el mapa, colocar el pin, usar GPS y guardar. El botón para usar el centro tiene el mismo peso visual que el GPS y no se explica qué centro usa.
3. El mapa no comunica si sus calles no cargaron. El pin y las coordenadas siguen funcionando, pero el usuario puede confundir un fondo vacío con un mapa roto.
4. El modo oscuro cambia el fondo y la atribución, pero deja los mosaicos claros; el mapa rompe la continuidad visual y deslumbra en baja luz.
5. El mapa de repartidores usa círculos similares para el origen y los repartidores, no ofrece leyenda, y los marcadores de repartidor no son controles accesibles por teclado. Al seleccionarlos sólo se destaca la tarjeta, sin llevarla a la vista.
6. `PointMap` destruye y vuelve a crear Leaflet y sus mosaicos al cambiar cualquier dato de puntos. En mapas que se refrescan, esto puede restablecer el zoom/centro y provocar parpadeo o tráfico de mosaicos.
7. No había búsqueda de calles/lugares; había que acercarse y mover el mapa manualmente. El selector depende además de geocodificación externa con limitación global. Cuando no hay respuesta inversa, el usuario podía escribir la dirección, pero el error quedaba lejos del campo.

## Plan detallado y ejecutado

### P0 — Orientación y lectura del mapa

- Traducir los controles nativos de Leaflet y ampliar sus objetivos táctiles sin ocultar la atribución.
- Añadir búsqueda explícita de calle/lugar por el endpoint `/api` de Photon y seleccionar uno de hasta cinco resultados sin volver a geocodificar las mismas coordenadas. El backend valida entrada, limita peticiones, cachea resultados y comparte la cadencia de upstream con la búsqueda inversa.
- Añadir instrucciones breves y persistentes: tocar coloca el pin, arrastrarlo afina el punto, GPS usa la ubicación del dispositivo y el centro sirve como alternativa manual.
- Mostrar estado de carga/fallo de mosaicos por separado del estado de geocodificación; permitir continuar con el mapa interactivo cuando falla el proveedor de calles.
- Dar formato consistente a la leyenda y distinguir origen, recogida, entrega y repartidores.
- En el despacho, mostrar a la vez los marcadores de recogida y entrega del trabajo seleccionado; búsqueda y selección de repartidor no deben convertir el mapa en un punto sin contexto de ruta.

### P1 — Selección accesible y feedback

- Convertir los marcadores seleccionables en controles de mapa enfocables por teclado y con nombres accesibles.
- Al seleccionar un repartidor en el mapa, reflejarlo en la lista y enfocar su tarjeta sin desplazar el documento de forma inesperada.
- Dar al usuario una única jerarquía de acciones y feedback: punto activo, dirección aproximada, coordenadas, búsqueda pendiente/error y acción de guardado.
- Mantener el consentimiento GPS explícito y no activar seguimiento continuo desde el selector de direcciones.

### P2 — Continuidad visual y estabilidad

- Adaptar visualmente los mosaicos al tema oscuro sin cambiar la paleta de CHITA ni invertir pines, controles o atribución.
- Actualizar las capas de `PointMap` sin recrear el mapa en cada refresh, preservando zoom y posición cuando el usuario interactúa.
- Comprobar visualmente escritorio, móvil, Firefox, temas claro/oscuro, zoom, resize, mapa sin mosaicos y geocoder fallido.

## Criterios de aceptación

- Ningún mapa sale del contenedor ni provoca scroll horizontal entre 320 px y escritorio.
- Todos los controles tienen etiqueta española, foco visible y tamaño táctil suficiente.
- Seleccionar un punto actualiza coordenadas; resolver dirección nunca bloquea corregirla manualmente ni guardar una ubicación válida.
- El fallo de mosaicos informa del problema sin desactivar el pin; el fallo de geocodificación ofrece una instrucción junto al campo de dirección.
- La búsqueda manual nunca es obligatoria: con proveedor lento, sin coincidencias o tras el límite, siguen disponibles mapa, GPS y escritura manual. Se envía al geocodificador sólo el texto de búsqueda, no el nombre, cuenta ni teléfono.
- Los puntos de origen/destino/repartidor se distinguen sin depender sólo del color; el mapa se puede recorrer y seleccionar con teclado.
- Actualizar los resultados de repartidores no reinicia el mapa ni cambia el zoom que eligió el usuario.
- Los tests existentes de persistencia real, privacidad/CSRF, Firefox y mapa en móvil/escritorio siguen pasando.

## Riesgos y límites

El mapa y la geocodificación dependen de servicios externos; la aplicación debe conservar selección manual y explicar fallos. La capa oscura es un tratamiento visual de los mosaicos actuales, no una nueva fuente de cartografía. No se transmite la dirección del usuario a un proveedor de mosaicos; sólo se hace geocodificación cuando el usuario selecciona un punto.

La consulta directa de producción detectó que Photon rechaza `lang=es` (HTTP 400). Se quitó ese parámetro para usar el idioma predeterminado del proveedor y conservar las etiquetas de interfaz en español; debe comprobarse la disponibilidad externa y no atribuir un 503 de proveedor a un fallo del mapa. Photon documenta los parámetros y el formato GeoJSON en su [API v1](https://github.com/komoot/photon/blob/master/docs/api-v1.md).
