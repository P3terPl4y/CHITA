# Auditoría y rediseño del panel de CHITA

## Diagnóstico visual y de uso

La revisión cubrió los paneles de empresa y repartidor, navegación fija, logo, apariencia clara/oscura y capturas de escritorio y móvil. Las pruebas de navegador existentes sitúan el panel en 1440 px, 390 px y 320 px, y ejecutan axe en vistas de dashboard. Esta auditoría detectó una brecha de identidad que las comprobaciones anteriores de accesibilidad no medían:

1. **La paleta contradice la marca.** El logo de CHITA es una silueta veloz en blanco y negro. El producto había pedido rojo y amarillo intensos, con superficies negras y blancas. Sin embargo, las últimas reglas globales del CSS restablecían índigo, violeta, celeste y ámbar. El dashboard cambiaba de personalidad según el orden de las hojas de estilo, no según una guía de marca.
2. **La jerarquía estaba bien encaminada, pero desperdiciaba escritorio.** El encabezado daba acceso a publicar o vincular HALCON y la página inicial proponía tareas por rol. Después apilaba “siguiente acción” y la ayuda en columnas de ancho completo; en monitores grandes quedaba una columna larga con espacio vacío.
3. **El header mezclaba demasiadas superficies.** La navegación superior y lateral funcionaban, pero sus estados seleccionados heredaban el violeta y el header usaba una superficie semitransparente. La navegación no transmitía la urgencia y claridad de una operación de reparto.
4. **El color requería medición, no preferencia.** El amarillo brillante sirve para destacar una acción sólo con texto oscuro; el rojo debe reservarse para selección y acciones principales con contraste suficiente. En el modo oscuro el rojo necesita seguir siendo suficientemente oscuro para texto blanco.
5. **El panel móvil ya se adaptaba, pero la composición perdía ritmo.** La barra de marca, el menú y la cuenta ocupan más de una fila; por eso la primera tarea debe quedar visible y cada tarjeta necesita una prioridad consistente, sin sumar adornos que alarguen el recorrido.

## Plan de ejecución

1. **Fijar el lenguaje de marca del dashboard.** Aplicar variables locales al panel para no alterar el resto de la web: carbón/negro y blanco roto como base, rojo como señal de acción y amarillo como acento de alta visibilidad. Mantener el logo monocromo legible sobre una barra superior negra.
2. **Recomponer la pantalla de inicio.** Mantener los pasos y etiquetas reales por rol; usar una tarjeta introductoria nítida con las acciones debajo. En escritorio, colocar resumen de actividad y ayuda en una composición de dos columnas; en móvil, volver a una sola columna.
3. **Unificar navegación y feedback.** Fondo opaco para el header, panel lateral negro, estado activo rojo con borde amarillo, CTA de cabecera amarillo sobre negro y botones principales rojos. Preservar foco visible y `prefers-reduced-motion`.
4. **Tratar claro y oscuro como el mismo sistema.** Conservar geometría, acentos y prioridad de acciones; ajustar sólo superficies y tonos que necesitan contraste en cada tema.
5. **Validar la identidad y la usabilidad.** Automatizar pruebas para ambos roles, ambos temas, contraste de texto/acción, estructura de columnas y desbordamiento a 390 px; ejecutar axe y revisar capturas reales de navegador antes de cerrar.

## Resultado implementado y límites

- El dashboard ahora usa negro y blanco con rojo `#d60024` en claro/`#e00027` en oscuro, y amarillo `#ffdc00`. Los tokens quedan limitados a `.dashboard-shell`, así que landing, login y áreas públicas no reciben un cambio accidental de paleta.
- El header mantiene el logo blanco sobre negro; el menú lateral usa negro, el elemento activo rojo con una guía amarilla y la acción destacada del encabezado usa amarillo con texto negro.
- La prueba de geometría encontró que el menú lateral fijado medía sólo 48 px en Firefox aunque sus reglas indicaban `top` y `bottom`; se fijó su altura respecto del viewport y se comprobó que ahora ocupa el área disponible y muestra todas las opciones.
- La pantalla inicial reorganiza el resumen de actividad y el consejo en dos columnas amplias, y conserva una sola columna en móvil. Los tres accesos rápidos siguen dependiendo del rol y del flujo real.
- Las pruebas cubren empresa y repartidor, temas claro y oscuro, ratios de contraste WCAG AA para texto normal, axe, jerarquía de paneles y ausencia de scroll horizontal a 390 px y 320 px.

El logo no se ha alterado. Se conserva la cartografía y los flujos del panel; esta intervención cambia su presentación, no añade promesas de velocidad, disponibilidad ni seguimiento.
