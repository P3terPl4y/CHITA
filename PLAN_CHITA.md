# Primera versión de CHITA

CHITA es un sistema independiente de HALCON. Usa Go, Fiber v3 y Goravel para el backend, PostgreSQL y Redis; React para web y una interfaz móvil adaptable.

## Ejecución

1. Corregir el arranque existente, separar las sesiones de HALCON y completar las migraciones.
2. Crear perfiles dedicados para empresa y repartidor, con dirección y coordenadas validadas.
3. Implementar redes de repartidores. La empresa invita por correo; el repartidor acepta antes de recibir sus publicaciones.
4. Publicar trabajos con direcciones, coordenadas, ventanas de recogida/entrega y tarifa en unidades menores de moneda, sin operaciones de pago.
5. Aceptar un trabajo de forma transaccional y exclusiva. Registrar recogida, llegada al destino, entrega reportada y confirmación de la empresa.
6. Mantener eventos y notificaciones persistentes. Reportar entrega no equivale a confirmarla; una empresa puede rechazar el reporte con motivo y solicitar revisión.
7. Integrar HALCON mediante sus API y WebSocket existentes, sin compartir tablas ni asumir que los IDs de ambas aplicaciones coinciden. El repartidor vincula una cuenta HALCON, con contraseña transitoria y sesión remota cifrada; sólo los participantes del trabajo activo consultan su ubicación desde CHITA.
8. Construir los paneles React y comprobar permisos, concurrencia, estados, integración y funcionamiento en móvil/escritorio con datos aislados.

## Decisiones iniciales

- Una cuenta tiene un rol inicial: empresa o repartidor. El rol de login se deriva del registro guardado; no se cambia con un parámetro del navegador.
- Una cuenta empresa gestiona una empresa en esta versión. Un repartidor puede pertenecer a varias redes.
- Las publicaciones sólo se ofrecen a repartidores que aceptaron la invitación. Los participantes conservan acceso a sus trabajos aceptados aunque después se retire la membresía.
- Los trabajos usan estados explícitos: `published → accepted → picked_up → arrived → delivery_reported → completed`. El rechazo de un reporte vuelve a `arrived`; la cancelación sólo está disponible antes de la recogida.
- Las fechas incluyen zona horaria y se guardan en UTC. La tarifa se registra con moneda; la app no procesa ni garantiza cobros.
- La ubicación de registro no demuestra presencia física. Informar llegada es una declaración del repartidor, no una prueba de entrega.
- El GPS del navegador requiere permiso, conexión y página abierta. Una app nativa y el rastreo con pantalla bloqueada quedan fuera de esta primera versión.
- HALCON conserva sus permisos propios, incluido su administrador y el destinatario que el repartidor hubiera configurado allí. La vinculación debe explicarlo antes de transmitir.

## Referencias contrastadas

- [Transacciones y bloqueos de Goravel](https://docs.goravel.dev/orm/getting-started.html).
- [Sesiones Fiber v3](https://docs.gofiber.io/middleware/session/) y [CSRF](https://docs.gofiber.io/middleware/csrf/).
- [Limpieza de efectos de React](https://react.dev/reference/react/useEffect).

Se verifican las firmas con las dependencias fijadas en `go.mod`; la guía local es `.agents/skills/goravel-development/SKILL.md`.

## Ejecución de la primera versión — 1 de octubre de 2026

- [x] Estudiar el esquema existente, las versiones y las API reales de HALCON.
- [x] Definir autorización, consentimiento de red y estados de entrega.
- [x] Implementar identidad, perfiles dedicados, migraciones, servicios y API Fiber.
- [x] Implementar vinculación cifrada, envío WebSocket y consultas autorizadas de ubicación.
- [x] Construir React adaptable a web y móvil con formularios y mapas.
- [x] Verificar reglas, aislamiento entre empresas, concurrencia, integración real, accesibilidad y tamaños de pantalla.
- [x] Verificar rollback y reinstalación de las doce migraciones en una base de datos aislada.

Límites deliberados de esta versión: cliente móvil web, GPS con la página abierta, una instancia de broker y sesiones HALCON renovables de 25 minutos. Pagos, push, verificación de identidad/correo y binarios móviles nativos necesitan su propio alcance y validación; no se prometen en esta entrega.
