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

## Corrección: mapa de cuenta en Firefox, persistencia y seguridad (2026-10-02)

- Fallo real reproducido: después de elegir un punto y guardar, la dirección nueva se persistía pero las coordenadas enviadas seguían siendo las anteriores. Los campos ocultos con `defaultValue` podían restablecerse durante el render. Se corrigió con estado de React y valores controlados; la prueba exige cambios de ambas coordenadas, comprueba el cuerpo PUT, lee PostgreSQL y recarga.
- Todos los mapas tienen marco de contención y recorte (`contain: layout paint`, aislamiento, tamaño explícito, ancho máximo). Se prueba Firefox al hacer zoom desde Mi cuenta, desplazamiento, escala CSS de 125% y anchos de 390 a 1920. Las capas quedan dentro del marco y no reciben eventos fuera. El mapa de trabajos también observa cambios de tamaño. La captura inicial automatizada no reprodujo constantemente el desbordamiento descrito en la laptop; la contención nueva se verifica en ambos motores, sin afirmar cobertura de todas las GPU/versiones.
- Firefox: 23 escenarios aprobados entre la ronda completa (22 aprobados) y la repetición del test de tema. Chrome: 23 escenarios en la ronda completa, sin omisiones. Se ejecutaron CRUD real, entregas públicas y exclusivas, propuestas/aceptación, tres entregas/calificación y dos guardados de ubicación con recarga. Mosaicos repetidos y algunas consultas de dirección son controlados; las escrituras, autenticación y lecturas reales usan HTTP y PostgreSQL aislado.
- El fallo del test de tema se aisló: Firefox perdía la preferencia emulada al navegar. Aplicar la emulación después de cargar conserva las comprobaciones; no se modificó la aplicación para ocultar ese fallo del entorno.
- Sesión de guardado revalidada y bloqueada dentro de la transacción. La petición con un permiso revocado no modifica la ubicación. Pruebas de CSRF, campos de propietario desconocidos y escritura tras cierre de sesión aprobado. El test de logout espera su respuesta antes de comprobar el rechazo.
- Vulnerabilidad de límites reproducida: un prefijo variable de X-Forwarded-For creaba claves distintas con validación IP desactivada. Se activó EnableIPValidation; el visitante real se determina desde la derecha, con confianza sólo en el proxy local. La petición 21 pasó de 422 a 429 con el mismo visitante y distintos prefijos.
- Última regresión Go con PostgreSQL, HALCON real aislado, `-race` y cobertura aprobada. Contrato de coordenadas real, separación de empresas/repartidores, asignación simultánea, administración, propuestas, calificaciones, sesiones y Redis comprobados. Cobertura combinada de servidor/controladores/servicios: 78,5% de sentencias; no equivale a cobertura completa ni garantiza ausencia de defectos.
- `go test ./...`, `go vet ./...`, compilación TypeScript/Vite y 17 pruebas unitarias React aprobadas. `npm audit --omit=dev --audit-level=high`: cero vulnerabilidades informadas en dependencias de producción.
- Auditoría Go: inicialmente 30 avisos alcanzables. Actualizados Go 1.26.8, pgx 5.9.2, QUIC 0.59.1 y gRPC 1.83.2. El análisis posterior informa un aviso alcanzable (GO-2026-6508) y otro en el módulo x/crypto sin llamadas ni importaciones de OpenPGP detectadas. El exportador afectado de logs gRPC queda deshabilitado al fijar HTTP en configuración; la telemetría de producción tampoco está activada. Su versión corregida 0.21 rompe la API usada por Goravel 1.18 y fue descartada; se documenta la deuda compatible, sin afirmar cero avisos.
- Dependencias finales: repetidos `go test ./...`, `go vet ./...` y la regresión PostgreSQL/HALCON con `-race`, todos aprobados. La cobertura combinada sigue en 78,5%. Una repetición React no llegó a iniciar trabajadores durante las compilaciones; repetida con un trabajador, las 17 pruebas pasaron. El análisis Go final se completó limitando memoria después de que el host terminara un intento anterior.
- Dominio público con Firefox: 15 escenarios aprobados en la ronda y cuatro escrituras reales omitidas por seguridad. El escenario restante de navegación falló una vez tras volver atrás, pasó aislado y en cinco repeticiones consecutivas. No se atribuye una causa definitiva ni se elimina esa observación del informe. Mapas y guardado visual de ambos perfiles sí pasaron.

- Firefox con el backend final: se observaron bloqueos intermitentes de `page.goto` en las pruebas de temas, aun con la interfaz renderizada. Cambiar el hito de espera a DOM/commit no los eliminó y esas modificaciones se retiraron. La sonda aislada completó doce navegaciones HTTP y el escenario de tema pasó por separado. Los proyectos Playwright separan ahora el proceso de temas del utilizado por mapas/capturas/seguimiento, conservando las comprobaciones originales. Las rondas anteriores tuvieron 22 de 23 aprobados; no se presentan como rondas sin fallos.
- Ronda final con proyectos aislados: **23 escenarios Firefox aprobados, sin omisiones ni reintentos** en 3,1 minutos. Incluye todos los recorridos reales anteriores y las pruebas de temas. La separación resuelve la ejecución observada; no se afirma una causa interna definitiva del bloqueo intermitente de Firefox.
- Publicado frontend y binario final tras respaldarlos, sin migraciones. `chita.service` activo; CHITA y HALCON público responden 200, perfil y cercanos sin sesión 401. Cookies `__Host-` con `secure`, `HttpOnly`, `SameSite=Lax`; API `no-store` y CSP presentes. Se comprobó una dirección real del proveedor. Los servicios temporales de CHITA/HALCON y PostgreSQL de validación se detuvieron.

El detalle de diseño y las limitaciones están en [DISENO_IMPLEMENTACION_CHITA.md](DISENO_IMPLEMENTACION_CHITA.md). No se ejecutó una carga de 1000 usuarios ni una prueba en la laptop física. La dirección externa puede fallar y admite edición manual; el GPS no acredita presencia física.

## Corrección y comprobación operativa (2026-10-03)

- Se encontró que el ejecutable principal ignoraba los argumentos `artisan`; por eso los comandos documentados podían arrancar el servidor en lugar de correr `migrate`, `key:generate` o `admin:create`. Ahora la entrada deriva los argumentos a Artisan. `go run . artisan list --no-ansi` y el binario compilado muestran los comandos esperados.
- Se añadió `/healthz` (HTTP 204) para distinguir el proceso HTTP de la ruta SPA comodín. Prueba unitaria y verificación local y pública aprobadas.
- Compose pasó parseo YAML. Define PostgreSQL y Redis con volúmenes, verifica sus healthchecks, ejecuta CHITA como usuario sin privilegios y expone el puerto sólo en loopback. El proxy confiable adicional queda limitado al gateway de su red Docker; la allowlist valida IP y CIDR y por defecto permanece vacía. No fue posible ejecutar `docker compose config` ni construir/probar la imagen: esta máquina no tiene Docker ni Podman.
- `go test ./... -count=1`, `go vet ./...`, `git diff --check`, 26 pruebas React y `npm run build`: aprobados. Siete escenarios Playwright de tema, landing, acceso, conectividad, responsive y accesibilidad pasaron contra el dominio público; las API de esos escenarios fueron simuladas y no escribieron cuentas.
- Comprobación de producción tras respaldar el binario y `react/dist` en `storage/backups/operational-20261003`: servicio CHITA y servicio Cloudflare activos; `/healthz` local y HTTPS público 204, landing 200 y directorio anónimo 401. Sin migraciones de datos. El proceso CHITA usa 56 MiB RSS en la muestra posterior al reinicio.
- El ejecutable Go actualizado está instalado y el servicio se reinició correctamente. No se detuvieron procesos ajenos: el proceso Python observado pertenece a `boti-whatsapp`, no a CHITA, y no mostró uso excesivo de memoria.

Queda pendiente probar el Dockerfile/Compose en una máquina con Docker, así como despliegues de carga y recuperación de respaldos. El servicio nativo actual está activo; estas verificaciones no demuestran disponibilidad continua ni certifican la aplicación como libre de defectos.

## Rediseño por tareas, guías, directorio y fotos (2026-10-02)

- TypeScript/Vite, 17 pruebas unitarias React y `go vet -p2 ./...` aprobados. Suite Go con PostgreSQL aislado y `-race` aprobada; repetición de servicios después de añadir aceptación de JPEG recodificado también aprobada.
- Nuevas pruebas de permisos: directorio sólo para empresas; ausencia de correo, teléfono, dirección y coordenadas; búsqueda parametrizada y paginación; afiliación limitada a cada empresa y aceptación sólo por destinatario; fotos propias, CSRF, concesiones vigentes, campos desconocidos, formatos/dimensiones inválidos, cuentas inactivas y límite HTTP 413 comprobado por socket real.
- Chromium: ronda completa de 31 escenarios aprobada. Se añadieron siete escenarios de perfiles, seis aprobados en la ronda específica y el último, corregido, aprobado por separado. Firefox: ronda completa de 30/32 aprobados; seis escenarios de perfiles aprobados al repetir con traza, y el último aprobado por separado. Todos los escenarios finales tienen ejecución aprobada, pero no se afirma una ronda completa de 32 sin fallos.
- El directorio falló una vez al intentar el primer clic y pasó en la repetición; no se atribuye una causa definitiva. La prueba de foto pendiente omitía el nuevo login y después tenía un selector ambiguo: se corrigió el recorrido y el selector del formulario, conservando la comprobación de que una respuesta tardía no modifica la cuenta nueva. Pasó en ambos navegadores.
- Regresiones nuevas: callbacks de ResizeObserver después de desmontar el mapa, publicación desde página dos con filtros, y respuesta de foto pendiente al cambiar de cuenta. Se corrigieron errores observados de navegación al compositor, landmarks duplicados, desbordamiento del perfil móvil y foco del aviso de error.
- Se probaron CRUD administrativo, trabajos públicos/exclusivos, propuestas, aceptación, entrega y confirmación, tres entregas para calificar, ubicación real con recarga, guías, borrador, filtros, menú móvil, fotos y directorio. Escrituras y sesiones reales contra PostgreSQL aislado; mosaicos y determinados estados de error se simulan explícitamente.
- Las comprobaciones axe pasan en los recorridos probados y ambos temas; no equivalen a certificación WCAG ni a pruebas con personas o la laptop física. No se repitió una carga de 1000 usuarios ni el contrato remoto completo con HALCON en esta ronda; se conserva la integración existente.
- Publicación con respaldo de binario y frontend en `storage/backups/dashboard-20261002-final`, sin migraciones. CHITA y HALCON responden 200, directorio anónimo 401. Se verificaron nombres de assets nuevos, API no-store, CSP y cookies __Host- Secure/HttpOnly/SameSite=Lax. Se reinició tras cambiar el índice para renovar la caché de archivos del servicio.
- Comprobación pública final con Firefox: dos escenarios aprobados de landing/navegación y apariencia persistente, accesible y adaptable, sin escrituras de producción. Vista previa y PostgreSQL de validación detenidos; chita.service permanece activo.

## Carga de usuarios simulados (2026-10-02)

- La prueba optativa nueva corre Fiber por HTTP en un puerto local y PostgreSQL dedicado `chita_validation`; rechaza otras bases y cualquier host DB no loopback. No envía tráfico a CHITA público. El escenario genera 500 empresas y 500 repartidores distintos con direcciones IP loopback únicas para que se apliquen los límites por IP.
- Registro: 1000 cuentas en 56,4 s con 20 trabajadores. Recorridos: 500 parejas, cada una completó invitación/aceptación, tres trabajos aceptados, recogidos, reportados y confirmados, y una calificación tras cumplir el umbral; 10 parejas concurrentes completaron esta fase en 1 min 45,2 s. Una pareja ejecutó además disponibilidad consentida, búsqueda cercana y oferta/aceptación para cubrir esos caminos.
- Seguridad durante la carga: repartidores no pueden crear trabajos; cuentas de empresa no pueden leer administración; otras empresas y repartidores no pueden ver publicaciones/ubicación ni publicar coordenadas ajenas; sólo el destinatario acepta afiliación/oferta. Se comprobaron 500 respuestas del directorio sin email, teléfono, domicilio ni coordenadas; CSRF inválido se rechazó con 403 y la sesión se pudo revalidar.
- Lecturas de 1000 clientes autenticados concurrentes: 3000 solicitudes de sesión, perfil y lista de trabajos en 7,59 s, con cero respuestas inesperadas. Latencias por solicitud: sesión p50 1,13 s / p95 2,39 s / máximo 5,27 s; perfil 1,37 / 2,95 / 5,83 s; trabajos 3,61 / 5,14 / 6,41 s.
- La carga funcional aprobó. El listado de trabajos es el cuello observado en el pico local y su p95 todavía es alto para una interacción fluida; no se fijó un SLO previo, así que estos tiempos son una línea base, no una garantía de producción para 1000 usuarios simultáneos. Conviene perfilar PostgreSQL, consultas y tamaño del pool en el hardware de despliegue antes de estimar capacidad. Esta prueba no simula 1000 escrituras concurrentes, móviles reales ni carga sostenida prolongada.
- Una repetición del escenario con `go test -race` alcanzó el timeout de 10 minutos antes de terminar, por la sobrecarga de la instrumentación a esta escala. No se contó como aprobada ni como medición de capacidad; el log no registró avisos de data race antes del timeout. La carga funcional sin instrumentación sí terminó correctamente.
