# Rediseño CHITA — progreso del 2 de octubre de 2026

Trabajo reanudado después de la pausa solicitada. Implementación, verificación local y publicación completadas. CHITA y HALCON responden 200; directorio sin sesión 401. No se ha solicitado push en esta ronda.

## Implementación

- Inicio organizado por tareas, navegación lateral en escritorio y menú modal móvil.
- Publicación en tres bloques, borrador conservado entre secciones; lista y detalle separados, filtros claramente limitados a la página cargada.
- Guías específicas para empresa y repartidor y landing con instrucciones y límites reales del servicio.
- Directorio de repartidores para empresas: búsqueda, paginación, foto, vehículo, puntuación y afiliación. No expone correo, teléfono, domicilio ni coordenadas.
- Invitación por ID con aceptación obligatoria del destinatario, conservando invitación por correo y requisito de tres entregas confirmadas para calificar.
- Fotos recortadas a 128×128; servidor valida tamaño, formato y dimensiones y recodifica sin metadatos. CSRF, sesión vigente, límites y protección frente a respuestas tardías de otra sesión.
- Temas claro/oscuro, rojo y amarillo intensos, feedback visible, foco y transiciones que respetan movimiento reducido.
- Mapas con callbacks protegidos tras desmontaje; publicación desde página posterior vuelve a página uno sin filtros.

## Verificación

Compilación TypeScript/Vite, 17 pruebas unitarias React, go vet y suite Go con PostgreSQL aislado y detector de carreras aprobados. Las pruebas incluyen privacidad del directorio, roles, SQL parametrizado, afiliación por empresa, aceptación, CSRF, fotos propias, formatos inválidos, cuentas inactivas y límite HTTP 413 real.

Chromium: ronda completa de 31 escenarios aprobada; casos adicionales de perfiles comprobados por separado. Firefox: ronda completa con 30 de 32 aprobados; seis casos aprobados al repetir y el caso final de sesión aprobado por separado en ambos navegadores. Un fallo correspondía a una simulación que omitía iniciar sesión en la cuenta nueva y se corrigió. El clic inicial del directorio falló una vez y pasó al repetir con traza; no se atribuye una causa definitiva.

Consultar PRUEBAS_CHITA.md para los resultados finales. No se ejecutó una carga nueva de 1000 usuarios ni se probó la laptop física. No hay migraciones nuevas.

## Cierre

Las dos comprobaciones públicas de landing y temas pasaron en Firefox. Se respaldaron binario y frontend, se publicó sin migraciones y se detuvieron la vista previa y PostgreSQL aislado. El servicio de producción permanece activo. Cambios guardados en commit local, sin push.
