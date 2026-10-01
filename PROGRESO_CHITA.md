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
