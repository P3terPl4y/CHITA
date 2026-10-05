# Plan de optimización para cargas altas

## Objetivo y alcance

Reducir trabajo repetido en las rutas consultadas con frecuencia y permitir ajustar el pool de PostgreSQL según la capacidad real del servidor, manteniendo permisos, visibilidad de publicaciones, orden del feed y contratos de API. Los cambios son incrementales y no requieren migraciones. No se desplegaron ni reiniciaron servicios.

## Hallazgos

- El sondeo del feed pedía el total paginado en cada actualización, aunque la interfaz ya conocía ese total.
- El feed cargaba entidades relacionadas completas y luego consultaba por separado ofertas pendientes; además contaba incluso cuando sólo se necesitaban los elementos.
- La autenticación consultaba la concesión del token y después el usuario, haciendo dos viajes a la base de datos en cada petición protegida.
- La interfaz consultaba trabajos cada 15 segundos y refrescaba también datos secundarios con demasiada frecuencia; los intervalos podían solaparse y las pestañas ocultas seguían generando tráfico.
- El pool estaba fijado en código, sin parametrización validada por límites.

## Cambios aplicados

1. El feed de trabajos usa una consulta acotada con joins y selecciona sólo las columnas usadas por el DTO. Mantiene los filtros de rol, red, estado y visibilidad. El recuento se omite cuando `total=false`; la carga inicial y las peticiones existentes conservan el total por defecto.
2. La validación del token y la carga del usuario activo se resuelven en una sola consulta indexada, respetando expiración, revocación y borrado lógico.
3. El frontend realiza sondeos seriales: trabajos cada 15 segundos, datos de red y avisos cada 30, estado HALCON cada 60. Añade variación aleatoria al intervalo, evita solapamientos, pausa el tráfico con la pestaña oculta y reintenta mostrando feedback de conexión. Las consultas periódicas omiten el total.
4. Para el orden geográfico del feed, el cliente reutiliza las coordenadas del perfil ya cargado; evita volver a buscar el perfil en cada sondeo. La ubicación elegida para ordenar sigue prevaleciendo sobre la ubicación guardada.
5. El pool se configura con `DB_MAX_OPEN_CONNS` (por defecto 32, máximo 96) y `DB_MAX_IDLE_CONNS` (por defecto 10, limitado a conexiones abiertas). Valores no positivos o inválidos usan el valor seguro predeterminado. No se elevó el valor por defecto: la prueba de 48 conexiones no mejoró el resultado frente a 32.
6. Se añadieron pruebas unitarias de límites del pool, pruebas de integración para la respuesta sin total y una prueba de navegador para comprobar cadencia y ausencia de solapamiento.

## Mediciones locales

Pruebas realizadas contra una base PostgreSQL aislada (`chita_validation`) y servicios locales de prueba; no contra producción. La carga incluyó un burst de 1.000 cuentas sintéticas con 3.000 lecturas y 500 pares de flujos empresa/repartidor con entregas y calificaciones.

| Escenario | p50 feed | p95 feed | Burst de 3.000 lecturas |
|---|---:|---:|---:|
| Antes, pool 32, respuesta antigua | 3,481 s | 5,011 s | 7,557 s |
| Tras consulta acotada y sin total en sondeo, pool 32 | 1,408 s | 3,016 s | 5,692 s |
| Mismo código, pool 48/16 | 1,660 s | 3,282 s | 6,018 s |

La mejora del feed medido fue aproximadamente 60% en p50 y 40% en p95 frente a la línea base local. Son tiempos de una prueba sintética en esta máquina, no una garantía de capacidad en producción. El benchmark evalúa una ráfaga con solicitudes concurrentes; no representa 1.000 usuarios permanentemente activos ni una prueba distribuida desde redes reales.

## Verificación y límites

- `go test ./...` pasó con las pruebas de integración aisladas.
- La suite de integración seleccionada también pasó con `-race`, incluidos despacho, ubicaciones, calificaciones y controles de acceso.
- Suite E2E de Firefox: 24 pasaron y 5 quedaron omitidas por requerir el servidor aislado. La prueba específica del sondeo pasó.
- `npm test` pasó (26 pruebas) y `npm run build` terminó correctamente antes del ajuste final de coordenadas en el sondeo; el build se vuelve a ejecutar como verificación de cierre.
- No se añadieron índices especulativos: las mediciones disponibles no demostraron que un índice concreto fuera la causa dominante, y añadirlos sin revisar planes de consulta podría penalizar escrituras.
- El p95 local sigue alrededor de 3 segundos en el feed bajo esa carga, así que esto no demuestra que el sistema esté listo para cualquier volumen. Para dimensionar capacidad real faltan objetivo de concurrencia/SLO, perfil de hardware de producción y pruebas con una copia anonimizada de datos representativos.

## Siguiente fase recomendada

Antes de aumentar límites del pool, medir `EXPLAIN (ANALYZE, BUFFERS)` con cardinalidades realistas, revisar métricas de espera/conexiones y fijar objetivos de latencia y concurrencia. Luego probar escalado horizontal, límites de tasa y comportamiento del canal de ubicación en tiempo real. Incorporar cache o colas sólo si los perfiles muestran presión concreta; la consistencia de ofertas, aceptación y finalización debe mantenerse transaccional.
