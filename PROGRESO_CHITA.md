# Punto de continuidad — CHITA

CHITA está en `/home/peter/CHITA`, repositorio independiente de HALCON. Todo el backend nuevo es Go con Goravel 1.18 y Fiber v3; el cliente está en `react/` y usa React 19 y TypeScript.

Implementados: registro y sesión por rol fijo, perfiles de empresa y repartidor con dirección y coordenadas, invitación y aceptación de redes, publicaciones con puntos, horarios y tarifa, aceptación exclusiva, recogida, llegada, reporte y confirmación/revisión de entregas, avisos, API protegida y vinculación cifrada a HALCON con seguimiento autorizado de trabajos activos.

Comprobado: doce migraciones con rollback y reaplicación en PostgreSQL aislado; pruebas Go, vet y detector de carreras; 106 peticiones HTTP con permisos, CSRF, concurrencia, límites y GPS; contrato con HALCON real de pruebas; aislamiento de claves de Redis; doce pruebas de tarifas/estados en React; recorrido E2E de ambos roles y accesibilidad/responsive. Los resultados finales y sus límites se documentan en PRUEBAS_CHITA.md.

La primera versión móvil es web adaptable, sin binarios nativos ni garantía de GPS en segundo plano. HALCON requiere cuentas personales existentes y una sesión renovable de 25 minutos. El broker actual de CHITA requiere una única instancia.

HALCON público sigue administrado por `halcon.service`; las pruebas usan HALCON en 3331 y CHITA en 3340, con PostgreSQL separado en 55439. No compartir tablas de ambos sistemas. Las pruebas de API truncan sólo una base explícitamente terminada en `_validation`.

Se creó una copia de seguridad privada de la base local de CHITA antes de migrarla: `storage/backups/before-chita-workflow-20261001.dump`. No incluir `.env`, backups, binarios, node_modules, dist ni sesiones en Git.

Para retomar: leer README.md, PLAN_CHITA.md, PRUEBAS_CHITA.md y el estado de Git antes de repetir acciones. Comprobar qué servicios están activos y qué verificación quedó pendiente. No publicar CHITA en Internet ni hacer push a otro repositorio sin que la sesión lo autorice.

## Estado al cerrar esta fase

La base local fue migrada y `chita.service` sirve la app en `http://localhost:3330`. El servicio conserva reinicio automático y usa HALCON en 3300. Hay 17 pruebas React, incluyendo renovación del GPS de un dispositivo estacionario. La posición de un halcón sin datos se devuelve vacía y no se dibuja como coordenadas cero. Se informó antes de aceptar un trabajo que la empresa tendrá acceso a la última posición del HALCON vinculado durante el trabajo activo.

El código está guardado en este repositorio local; no se ha configurado un dominio público ni publicado CHITA en GitHub como parte de esta fase. Antes de extenderlo, revisar los límites documentados y elegir explícitamente el alcance móvil nativo y la estrategia para múltiples instancias.

Las instancias temporales de pruebas (3331, 3340 y PostgreSQL 55439) se cerraron después de validar. Sus archivos y bases de datos se conservan para futuras ejecuciones. Los servicios permanentes son `halcon.service` y `chita.service`.

## Publicación y diseño — 1 de octubre de 2026

Por autorización posterior del usuario, CHITA se subió a GitHub (main, b5ed826) y se publicó en https://chita.duohnson.com mediante el túnel web-d. El usuario creó el DNS desde vadmacska y reinició cloudflared. Se verificaron CHITA y HALCON por HTTPS. chita.service usa APP_ENV=production y cookies Secure/HttpOnly.

El cliente adopta la estructura visual de HALCON con rojo, amarillo y blanco sabana (#fffaf0). Incluye selector Apariencia: Sistema, Claro y Oscuro; preferencia local, sincronización entre pestañas, seguimiento del tema del sistema y funcionamiento con almacenamiento bloqueado. Se actualizaron el icono, manifest y color del navegador. El cliente compilado está servido en el dominio público.

Validación: build TypeScript/Vite, 17 pruebas unitarias y dos pruebas Playwright de apariencia contra el dominio público, sin crear usuarios. Estas verifican persistencia tras recarga, adaptación de 320 a 1440 píxeles, cambios del sistema y auditoría axe de entrada/registro en ambos temas sin infracciones. Se revisaron capturas de escritorio y móvil. No se repitió el recorrido transaccional de entregas: no hubo cambios en API ni en su lógica.

### Refinamiento visual dinámico

La siguiente revisión usa la portada real de HALCON (app/views/landing.html) como referencia: fondos radiales, profundidad de tarjetas, acentos vivos y recorrido ilustrado. CHITA conserva identidad roja/amarilla/sabana; símbolo de dirección e icono propios, CTA con enlace al formulario para evitar desplazamiento innecesario en móvil y modos claro/oscuro/sistema. El recorrido decorativo se identifica como ilustración, no muestra datos reales y respeta prefers-reduced-motion. No modifica la lógica de trabajos o ubicación. Validación adicional: build, 17 pruebas unitarias, dos pruebas Playwright de apariencia/accesibilidad en el dominio público y revisión de capturas. Movimiento reducido verificado: animationName=none.

### Landing independiente y UX neón

Nueva landing en /, acceso en /entrar y registro en /registro, con enlaces específicos ?rol=empresa y ?rol=repartidor. Las transiciones internas usan History API sin recargar, respetan clics modificados y el botón Atrás; los enlaces directos también funcionan tras recarga gracias al fallback SPA existente. Separación visual entre promoción y trabajo, secciones de roles/flujo/HALCON y limitaciones reales.

Paleta actual: rojo #df0029, amarillo #ffdc00, blanco #ffffff y negro #050505, con degradados neón y brillos decorativos. Los formularios usan fondos sólidos, selector de apariencia y foco accesible. Mostrar/ocultar contraseña, estado de geolocalización y enlace para saltar al contenido. Puntos del mapa diferenciados por nombres y colores de CHITA.

Verificación: compilación TypeScript/Vite, 17 pruebas unitarias existentes y cinco pruebas Playwright de UX/UI. Auditorías axe de landing, entrada/registro y paneles de ambos roles en claro/oscuro; adaptación 320–1440, navegación/recarga/rol, conservación de credenciales tras rechazo simulado, almacenamiento bloqueado y movimiento reducido. Paneles privados y rechazo de acceso se prueban con respuestas interceptadas en el navegador: no crean cuentas ni alteran trabajos de producción. No equivalen a una nueva ejecución del flujo transaccional de entregas. Se revisaron capturas de landing y acceso móvil.

## Trabajos públicos y navegación de plataforma

Implementados y publicados: `visibility` pública/exclusiva (por defecto network, incluidos trabajos anteriores), disponibilidad pública para repartidores sin invitación y asignación única. Los permisos posteriores a aceptar y el seguimiento siguen restringidos. Migración 20261001153005 con restricción e índice; rollback no borra una clasificación pública existente.

Ranking en PostgreSQL antes de paginar: trabajos propios activos, disponibles por distancia a recogida, cierre de ventana y desempate por ID; perfil por defecto, GPS puntual opcional que no activa HALCON ni persiste la ubicación. La distancia se etiqueta como línea recta. Nuevo navbar superior y drawer móvil modal con foco/escape/resize. HALCON tiene sección propia y botones directos. Logo original public/chita.svg copiado a react/public/chita.svg, conservado y servido en /chita.svg. Landing ampliada con seis preguntas de uso, reglas y límites.

Verificación: Go/vet, 163 peticiones aisladas con -race, migración reversible sin publicaciones públicas y protegida cuando las hay, 17 pruebas React, nueve escenarios web verificados y seis de UX repetidos por HTTPS. Actualizados README y PRUEBAS_CHITA con el alcance exacto. Copia privada antes de migrar: storage/backups/before-public-jobs-1790870024907.dump. Binario anterior conservado en storage/bin/chita.before-public-jobs; no incluir esas rutas ni credenciales en Git.
