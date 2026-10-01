# Verificación de CHITA — 1 de octubre de 2026

Esta es una primera versión funcional, comprobada en el entorno de desarrollo. Los resultados describen los escenarios ejecutados; no son una garantía de ausencia de fallos ni una certificación de producción.

| Comprobación | Resultado |
|---|---|
| `go test ./...` | Correcto; la integración optativa se omite sin su variable explícita |
| `go vet ./...` | Correcto |
| Pruebas reales con `go test -race ./app/server ./app/services` | Correcto, sin carreras detectadas en los casos ejecutados |
| Flujo y seguridad HTTP | 106 peticiones comprobadas, cuatro cuentas CHITA, dos repartidores compitiendo por una asignación |
| Contrato real con HALCON aislado | Registro de cuenta de prueba, login, identidad personal, WebSocket, persistencia de coordenadas y desconexión correctos |
| Vinculación CHITA–HALCON | Sesión cifrada, identidad no compartida entre repartidores, envío y consulta de coordenadas autorizados, desvinculación y bloqueo tras reporte comprobados |
| Redis | Claves con prefijo propio, lectura/borrado contextual y prohibición de reset global comprobados con Redis real |
| PostgreSQL | Doce migraciones aplicadas, revertidas y reaplicadas en `chita_validation`; posteriormente aplicadas a la base local de CHITA previamente vacía |
| React | Pruebas de tarifas, estados y renovación del GPS; build TypeScript/Vite correcto |
| Chromium / Playwright | Dos escenarios completos aprobados: formularios accesibles y recorrido empresa–repartidor |
| Responsive | Sin desbordamiento horizontal en las vistas comprobadas a 320, 390, 768 y 1440 píxeles |
| Accesibilidad automática | Sin incidencias de axe en registro y detalles de entrega comprobados |
| Arranque local | Servicio `chita.service` activo; interfaz y sesión responden HTTP 200 en `http://localhost:3330` |

## Escenarios de negocio y acceso

Se comprobó que un repartidor no puede publicar trabajos; una empresa ajena no puede consultar ni cancelar publicaciones de otra; un repartidor no ve trabajos de una empresa hasta aceptar la invitación; tampoco puede aceptar una invitación ajena ni modificar una entrega no asignada a él.

Dos repartidores aceptaron simultáneamente el mismo trabajo: sólo uno obtuvo la asignación. Una segunda aceptación, los saltos de estado, la confirmación prematura, la cancelación después de recoger y la aceptación vencida fueron rechazados.

Retirar a un repartidor de la red impidió consultar publicaciones nuevas y mantuvo su acceso a trabajos ya aceptados. Se probaron reporte, solicitud de revisión con motivo, nuevo reporte y confirmación. Los eventos, avisos y fechas de confirmación/entrega quedaron persistidos.

Se rechazaron peticiones sin CSRF válido, campos de asignación enviados por el cliente, tarifas inválidas, coordenadas fuera de rango, lectura de avisos ajenos y solicitudes de seguimiento de terceros. Los DTO de trabajos excluyen hashes y sesiones cifradas. Se verificaron cookies HttpOnly, SameSite y Secure en modo producción, cabeceras de seguridad y respuestas 429 después del límite de autenticación.

La vinculación se hizo contra HALCON en 3331 y bases de datos de pruebas separadas en PostgreSQL 55439. CHITA envió un punto mediante su API; la empresa recibió ese mismo punto desde la persistencia real de HALCON. No se sustituyó la integración por un simulador de HALCON. Las imágenes de mapa en los tests repetidos de navegador sí son fixtures: permiten verificar el selector y la carga sin descargar mosaicos comunitarios repetidamente.

## Límites de la verificación

No se ejecutó una carga de 1000 usuarios en CHITA. Esa prueba correspondió al trabajo anterior de HALCON y no se reutiliza como evidencia de este sistema. No se probaron dispositivos móviles físicos, GPS en segundo plano ni un despliegue público de CHITA. La prueba automática de accesibilidad cubre las vistas indicadas, no sustituye pruebas con lectores de pantalla o personas usuarias.

La versión móvil actual comparte el cliente React adaptable. Las sesiones vinculadas de HALCON duran como máximo 25 minutos y deben renovarse. El broker requiere una sola instancia de CHITA. Pagos, push, verificación de correo y aplicaciones nativas quedan fuera de esta versión.

El rollback completo del esquema se verificó en una base descartable. El rollback aislado de la migración de flujo conserva la mayor precisión de coordenadas y la posibilidad de tax_id nulo para evitar redondear coordenadas o inventar identificadores fiscales.

Los comandos y las precauciones para reproducir las pruebas están en [README.md](README.md). Las pruebas de integración truncan usuarios únicamente cuando se proporciona el indicador explícito y una base cuyo nombre termina en `_validation`.

Se aprobaron 17 pruebas de React en total. Cinco comprueban el GPS: caché reciente, renovación sin movimiento, primera lectura, error de permisos/señal y rechazo de una posición antigua. La integración real también verifica que un halcón sin ninguna posición previa no aparezca artificialmente en 0,0.

## Actualización: trabajos públicos, cercanía y navegación

Resultados posteriores a la primera fase descrita arriba:

- `go test ./...` y `go vet ./...`: aprobados.
- Integración con PostgreSQL aislado y `-race`: 163 peticiones, cinco cuentas CHITA, pruebas de ambos tipos de publicación y aceptación simultánea pública y exclusiva. Sin carreras detectadas.
- Público sin invitación visible/aceptable; exclusivo oculto sin consentimiento; empresa ajena sin acceso; trabajo público asignado oculto para otro repartidor; seguimiento no abierto por la visibilidad pública. Cancelados y vencidos excluidos, CSRF inválido rechazado.
- Ranking global antes de paginar probado con 32 publicaciones públicas, página de 30 y segunda página de dos; un trabajo activo lejano precede al disponible cercano. Coordenadas negativas, cero, no finitas, incompletas y fuera de rango comprobadas. Distancias unitarias verificadas en origen, antimeridiano y puntos opuestos.
- Migración 13: rollback/reaplicación con trabajos existentes conserva `network`; rollback con publicaciones públicas se rechaza para no borrar su clasificación. Migración aplicada al servicio tras copia privada de PostgreSQL.
- React: 17 pruebas existentes y compilación TypeScript/Vite aprobadas. Nueve escenarios Playwright verificados contra CHITA aislado: seis de UX/UI y tres de formulario/recorrido, incluyendo entrega exclusiva completa y aceptación pública sin invitación. El fallo inicial del nombre accesible del menú se corrigió; el escenario afectado se repitió y pasó.
- Publicación HTTPS: los seis escenarios de UX pasaron en el dominio público sin crear datos de producción. Menú modal con cierre, Escape/foco, acceso a HALCON y cambio de ancho; GPS de ranking; carga del logo; ambos temas y axe. Paneles privados y GPS usan respuestas/permisos controlados en estas pruebas de UI.
- CHITA, SVG y HALCON público respondieron 200. Capturas de navegación superior y lateral revisadas.

El contrato remoto real de HALCON no se repitió en esta ronda: el broker y sus reglas permanecen como en las pruebas de la primera fase. No se hizo una nueva carga de 1000 usuarios ni pruebas con dispositivos físicos.

## Actualización: mapas, propuestas y calificaciones (2026-10-01)

- `go test ./...` y `go vet ./...` aprobados. Pruebas de servidor y servicios con PostgreSQL aislado y `-race` aprobadas, sin carreras detectadas. Se conservaron los flujos anteriores de trabajos públicos y exclusivos y el CRUD administrativo.
- Nueva integración: ubicación del perfil, validación de coordenadas y campos, roles, GPS con consentimiento revocable, actualizaciones atrasadas, orden por distancia, privacidad de DTO, reserva única concurrente, rechazo/retirada/caducidad de propuestas, y tres entregas confirmadas antes de calificar. Se rechazaron calificaciones prematuras, de empresas sin las entregas requeridas y valores fuera de rango; actualizar una calificación conserva una sola contribución al promedio.
- React: 17 pruebas unitarias aprobadas y compilación TypeScript/Vite aprobada. Navegador local: cuatro escenarios de mapas/disponibilidad/propuestas y un flujo real de tres entregas con calificación, además de los recorridos anteriores de entrega y el CRUD administrativo real. El CRUD verifica ahora selección desde el mapa en cuentas y trabajos.
- Se corrigieron selectores antiguos de pruebas que eran ambiguos al añadir regiones de mapa y mensajes de estado. Los recorridos afectados se repitieron y pasaron.
- Versión pública: 11 escenarios Playwright aprobados (mapas, paneles, navegación, temas, responsive y comprobaciones axe). Los dos escenarios que crean registros reales se omitieron deliberadamente en producción; sí se ejecutaron en el servidor aislado. Las respuestas privadas y los mosaicos repetidos se controlan en estas pruebas de UI.
- Respaldo privado de PostgreSQL y de la versión anterior antes de publicar. Migraciones 15 y 16 aplicadas. CHITA y HALCON público responden 200; búsqueda cercana sin sesión responde 401. Una consulta real del nuevo endpoint devolvió la dirección del proveedor Photon.

Esta ronda no repitió el contrato remoto completo de HALCON ni una carga de 1000 usuarios. No verifica dispositivos físicos, GPS en segundo plano ni disponibilidad continua del proveedor externo. La posición cercana requiere consentimiento y expira tras cinco minutos sin actualización; una dirección sugerida puede corregirse manualmente.

## Actualización: edición directa de cuenta y paneles neón

- El editor de empresa y repartidor usa un único mapa abierto, con selección directa, coordenadas internas y dirección editable. Guardar espera a que termine la búsqueda y muestra confirmación. Los errores conservan la selección; descartar restaura la ubicación guardada.
- Panel de cuenta con identidad y ubicación en columnas adaptables. Disponibilidad sigue montada al cambiar de sección para conservar el consentimiento, pero su control sólo aparece en trabajos/propuestas. La selección de un trabajo en móvil desplaza y enfoca sus detalles; respeta la preferencia de movimiento reducido.
- Compilación TypeScript/Vite y 17 pruebas unitarias aprobadas. Trece escenarios locales de navegador aprobados, con dos pruebas de escritura real omitidas porque esta ronda utiliza una vista previa estática con API simulada. Una prueba adicional de foco y visibilidad de detalles en móvil aprobada.
- Los nuevos escenarios de cuenta comprueban empresa y repartidor, espera de dirección, guardado, errores recuperables, descarte, ausencia de campos numéricos visibles, anchos de 320/390/768/1440 y comprobaciones axe en temas claro/oscuro.
- Publicado después de respaldar el frontend anterior. CHITA responde 200. Ocho escenarios públicos de cuenta y temas aprobados con API controlada, sin modificar registros de producción.

Esta actualización sólo modifica el cliente React; no cambia las reglas de permisos, propuestas, calificaciones ni el esquema. Las comprobaciones automáticas de accesibilidad no sustituyen pruebas con dispositivos físicos o personas usuarias.
