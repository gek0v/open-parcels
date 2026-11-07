# Plan Maestro de Issues para GitHub — Open-Parcels

Este documento recoge el análisis técnico exhaustivo del repositorio `gek0v/open-parcels` y define la descomposición estructurada de **Issues listos para su publicación en GitHub**, organizados por hitos (*milestones*), áreas clave y prioridades de desarrollo.

---

## 1. Diagnóstico y Análisis por Aspectos Clave

El proyecto **Open-Parcels** es un motor universal de rastreo de envíos en Go que cuenta con una arquitectura base bien diseñada (CLI, REST API compatible con ParcelsApp v4, servidor MCP y motor de deduplicación/fusión de eventos). Sin embargo, tras la auditoría técnica profunda (`AUDIT_MOCKS_PRODUCCION.md`) y la inspección del código, se identifican 6 áreas críticas que requieren resolución mediante issues específicos:

1. **La Crisis de los Mocks y Veracidad de Datos (Deuda Técnica Mayor)**:
   - Más de 1.570 transportistas operan como stubs en memoria (`time.Now()`), devolviendo siempre HTTP 200 con eventos inventados ("En reparto", progreso 85%).
   - Inconsistencia con el motor: `pkg/engine/merger.go` contiene filtros específicos para eliminar eventos generados después de una entrega simulada.
   - Es imprescindible una transición estricta hacia errores canónicos (`ErrShipmentNotFound`, `ErrCarrierNotImplemented`) para que el sistema sea confiable en producción.

2. **Infraestructura de Adaptadores (Carrier Adapter Framework)**:
   - Actualmente no existe una distinción clara entre adaptadores de acceso público (web scraping, endpoints inversos como InPost o Correos Express) y adaptadores empresariales (APIs que requieren API Keys/OAuth2 como DHL, UPS, FedEx).
   - Ausencia de un sistema unificado de configuración de credenciales y cabeceras de red / proxies.

3. **Estrategia de Testing y CI/CD**:
   - Tests de integración en vivo (como `TestInPostLiveTracking`) llaman a la red real con números de seguimiento fijos, provocando fallos en `go test ./...` por expiración de guías, rate-limiting o falta de conectividad.
   - Necesidad urgente de separar tests unitarios con *golden fixtures* de tests de integración en vivo mediante Go build tags (`//go:build integration`).

4. **Motor de Detección Heurística (`pkg/detector`)**:
   - Cuenta con algoritmos matemáticos sólidos (UPU S10 Modulo 11, UPS 1Z Modulo 10, IATA Modulo 7, ISO 6346), pero las más de 1.500 expresiones regulares de transportistas globales provocan colisiones y ambigüedades en identificadores genéricos.

5. **Interfaces de Consumo (CLI, REST ParcelsApp v4 y MCP)**:
   - Necesidad de reflejar la autenticidad del dato (`[MOCK]` vs `[LIVE]`), soportar sondeo asíncrono y enriquecer la experiencia en agentes IA a través de MCP.

6. **Gobernanza y Transparencia**:
   - El `README.md` y la documentación proclaman soporte de "+1.500 transportistas", lo que no refleja el estado operativo real. Se requiere una matriz de compatibilidad viva.

---

## 2. Mapa de Hitos (Milestones)

- **Milestone 1: Core Engine & Carrier Adapter Standard (Semana 1)**
  - Erradicación de fallbacks falsos, estandarización de la interfaz y testing determinista.
- **Milestone 2: Integraciones Reales Tier 1 — España e Iberia (Semana 2)**
  - Correos Express, Correos España, SEUR, GLS Spain, CTT Express y MRW.
- **Milestone 3: Integraciones Reales Tier 1 — Globales y Cross-Border (Semana 3)**
  - Cainiao real sin bypass, YunExpress, DHL Express, UPS y FedEx.
- **Milestone 4: Optimización del Motor Heurístico y Detección (Semana 4)**
  - Desambiguación, benchmark y pre-filtrado de reglas.
- **Milestone 5: MCP, REST API y Capacidades Asíncronas (Semana 5)**
  - Streaming MCP, webhooks y polling de envíos de larga duración.
- **Milestone 6: Documentación y Transparencia (Transversal)**
  - Badges de estado por transportista y matriz de cobertura verificada.

---

## 3. Catálogo Estructurado de Issues para GitHub

---

### ISSUE #1: refactor(engine): Erradicación de fallbacks simulados silenciosos y gestión estricta de errores canónicos

- **Área:** Core Engine & Models  
- **Milestone:** Milestone 1 — Core Engine & Standard  
- **Prioridad:** 🔴 Alta (Bloqueante para producción)  
- **Etiquetas:** `bug`, `refactor`, `core-engine`

#### Contexto y Justificación
Actualmente, el sistema devuelve datos simulados con `time.Now()` ante guías inexistentes (`000000`, `FAKE123456`), simulando un envío en tránsito con HTTP 200 OK. En `pkg/engine/merger.go` (L40), se filtran eventos posteriores a la entrega para ocultar las anomalías causadas por estos mocks. El motor debe rechazar inventar estados y devolver errores canónicos cuando un envío no existe o el transportista aún no tiene integración de red real.

#### Objetivos y Criterios de Aceptación
- [ ] Definir errores canónicos de dominio en `pkg/models/errors.go`:
  - `ErrShipmentNotFound` (paquete inexistente o no registrado).
  - `ErrCarrierNotImplemented` (transportista registrado en catálogo pero sin adaptador de red).
  - `ErrPostalCodeRequired` (transportistas que exigen código postal para consultar).
  - `ErrCarrierRateLimited` (bloqueo temporal o throttling del operador).
- [ ] Modificar `pkg/engine/engine.go` para que, salvo que `opts.AllowSimulated == true`, nunca devuelva un `*models.Shipment` con datos ficticios.
- [ ] Eliminar en `pkg/engine/merger.go` el parche de descarte de eventos posteriores a entrega una vez que los adaptadores devuelvan datos consistentes.
- [ ] Ajustar `pkg/server/handlers.go` para que `ErrShipmentNotFound` responda con **HTTP 404 Not Found** y `ErrPostalCodeRequired` con **HTTP 422 Unprocessable Entity**.
- [ ] Actualizar tests unitarios del engine para validar que las guías inexistentes devuelven error y no una simulación.

#### Implementación Técnica
- Archivos a modificar: `pkg/models/errors.go`, `pkg/engine/engine.go`, `pkg/engine/merger.go`, `pkg/server/handlers.go`, `pkg/engine/engine_test.go`.

#### Fuera de Alcance
- Implementar los scrapers o APIs de los transportistas (se abordará en los issues de cada carrier).

---

### ISSUE #2: feat(carriers): Especificación e interfaz estándar para adaptadores reales (`CarrierAdapter`), credenciales y fixtures

- **Área:** Carrier Infrastructure  
- **Milestone:** Milestone 1 — Core Engine & Standard  
- **Prioridad:** 🔴 Alta  
- **Etiquetas:** `enhancement`, `architecture`, `carriers`

#### Contexto y Justificación
Los 55 adaptadores dedicados y los 14 catálogos regionales implementan una función básica `Track(ctx, number, opts)`, pero carecen de una estructura formal para manejar credenciales de API (OAuth2, API Keys), soporte de proxies, gestión de cookies de sesión, y rotación de User-Agents.

#### Objetivos y Criterios de Aceptación
- [ ] Definir la interfaz enriquecida `CarrierAdapter` y metadatos del transportista:
  - `ID() string`, `Name() string`, `Capabilities() CarrierCapabilities`.
  - `Track(ctx context.Context, number string, opts TrackOptions) (*Shipment, error)`.
  - Flags de soporte: `RequiresPostalCode`, `RequiresAuth`, `SupportsProofOfDelivery`.
- [ ] Implementar un gestor de configuración de transporte HTTP centralizado (`pkg/carriers/transport.go`):
  - Inyección de timeout por defecto (10s).
  - User-Agent estándar configurable.
  - Soporte opcional de proxy HTTP/SOCKS5 mediante variables de entorno (`OPEN_PARCELS_PROXY_URL`).
- [ ] Estructura para inyección segura de credenciales vía variables de entorno (`OPEN_PARCELS_<CARRIER>_KEY`, `OPEN_PARCELS_<CARRIER>_SECRET`).
- [ ] Documentar en `docs/CARRIER_GUIDE.md` la plantilla obligatoria para crear nuevos adaptadores reales.

#### Implementación Técnica
- Archivos a crear/modificar: `pkg/carriers/carrier.go`, `pkg/carriers/transport.go`, `docs/CARRIER_GUIDE.md`.

#### Fuera de Alcance
- Configuración de servidores proxy externos o pago de suscripciones a APIs privadas.

---

### ISSUE #3: test(ci): Separación de tests unitarios vs integración en vivo (`//go:build integration`) y fixtures HTTP deterministas

- **Área:** CI/CD & Testing  
- **Milestone:** Milestone 1 — Core Engine & Standard  
- **Prioridad:** 🔴 Alta  
- **Etiquetas:** `testing`, `ci`, `dx`

#### Contexto y Justificación
`TestInPostLiveTracking` en `pkg/carriers/inpost_test.go` realiza peticiones de red reales contra `inpost.es` usando un número de seguimiento real (`75616714`). Esto causa que `go test ./...` falle en entornos locales y CI cuando la conexión se corta, InPost aplica rate limiting o el paquete es purgado de sus sistemas.

#### Objetivos y Criterios de Aceptación
- [ ] Agregar tag de compilación `//go:build integration` a todos los tests que requieran conexión de red en vivo contra APIs externas.
- [ ] Implementar un mecanismo de *golden fixtures* o transporte HTTP simulado en memoria (`http.RoundTripper`) para reproducir respuestas JSON reales de transportistas sin tocar la red durante el test unitario.
- [ ] Asegurar que `go test -v -race ./...` (sin flags de integración) se complete en < 5 segundos en local y en CI sin llamadas a Internet.
- [ ] Crear un workflow en GitHub Actions `.github/workflows/integration.yml` que corra semanalmente o mediante `workflow_dispatch` con los tests de integración (`go test -tags=integration ./...`).

#### Implementación Técnica
- Archivos a modificar: `pkg/carriers/inpost_test.go`, `.github/workflows/ci.yml`, `.github/workflows/integration.yml`, creación de `testdata/fixtures/`.

#### Fuera de Alcance
- Mockear servicios externos con servidores Docker mock dedicados (mantenerlo ligero en Go estándar).

---

### ISSUE #4: feat(carriers): Integración real para Correos Express (eliminación de bypass estático 2026 y scraping API pública)

- **Área:** Carriers — España  
- **Milestone:** Milestone 2 — España e Iberia  
- **Prioridad:** 🟠 Media-Alta  
- **Etiquetas:** `enhancement`, `carriers`, `spain`

#### Contexto y Justificación
`pkg/carriers/correosexpress.go` contiene un cortocircuito hardcodeado para la guía `32300035210642701471403` con eventos fijados en noviembre de 2026, y para cualquier otro código inyecta 4 eventos estáticos con `time.Now()`. Correos Express dispone de un endpoint REST público de consulta usado por su portal web.

#### Objetivos y Criterios de Aceptación
- [ ] Eliminar el bypass de código estático y las respuestas simuladas en `correosexpress.go`.
- [ ] Implementar cliente HTTP real contra la API de seguimiento público de Correos Express (`https://www.correosexpress.com/web/correosexpress/...`).
- [ ] Mapear los estados de Correos Express a los modelos canónicos de Open-Parcels (`StatusInTransit`, `StatusOutForDelivery`, `StatusDelivered`, etc.).
- [ ] Parsear correctamente fechas, oficinas de origen/destino y firma de entrega si está disponible.
- [ ] Manejo de respuesta de guía no encontrada devolviendo `ErrShipmentNotFound`.
- [ ] Test unitario con fixture grabada y test de integración con tag `integration`.

#### Implementación Técnica
- Archivo: `pkg/carriers/correosexpress.go`, `pkg/carriers/correosexpress_test.go`.

#### Fuera de Alcance
- Integración con el webservice SOAP B2B privado de Correos Express para clientes de contrato (solo consulta pública de guías).

---

### ISSUE #5: feat(carriers): Integración real para Correos España (seguimiento web público / localizador)

- **Área:** Carriers — España  
- **Milestone:** Milestone 2 — España e Iberia  
- **Prioridad:** 🟠 Media-Alta  
- **Etiquetas:** `enhancement`, `carriers`, `spain`

#### Contexto y Justificación
Correos España (`pkg/carriers/correos.go`) es el operador postal de referencia en España y actualmente devuelve un mock puro de 4 eventos generados en memoria. Debe conectarse a la API pública del localizador de envíos de Correos.

#### Objetivos y Criterios de Aceptación
- [ ] Reemplazar la simulación de `correos.go` por peticiones reales al endpoint JSON del localizador de envíos de Correos España.
- [ ] Soportar formatos de seguimiento estándar UPU S10 (ej. `PQ...`, `CD...`, `LP...`).
- [ ] Parseo y normalización de la línea de tiempo completa de eventos en español e inglés.
- [ ] Extracción de la oficina de recogida y fecha límite de estancia si el estado es "En lista / Disponible en oficina".
- [ ] Test unitario con respuesta simulada de Correos y test de integración.

#### Implementación Técnica
- Archivo: `pkg/carriers/correos.go`, `pkg/carriers/correos_test.go`.

#### Fuera de Alcance
- Conexión al servicio B2B con certificado digital de Correos (alcance acotado a consulta web pública).

---

### ISSUE #6: feat(carriers): Integración real para SEUR y GLS Spain con validación de código postal

- **Área:** Carriers — España  
- **Milestone:** Milestone 2 — España e Iberia  
- **Prioridad:** 🟡 Media  
- **Etiquetas:** `enhancement`, `carriers`, `spain`

#### Contexto y Justificación
Tanto SEUR (`seur.go`) como GLS Spain (`gls_spain.go`) operan como mocks. Ambos operadores requieren o se benefician del código postal de destino (`opts.PostalCode`) para desbloquear la información detallada de entrega en sus portales públicos.

#### Objetivos y Criterios de Aceptación
- [ ] Adaptador para **SEUR**: integración con el endpoint de tracking web público de SEUR (`seur.com`).
- [ ] Adaptador para **GLS Spain**: integración con el localizador de envíos nacional de GLS.
- [ ] Si el operador exige código postal y no se suministra en `opts.PostalCode`, devolver `ErrPostalCodeRequired` con mensaje descriptivo.
- [ ] Normalización de estados y deduplicación de hitos de entrega fallida / reintento.

#### Implementación Técnica
- Archivos: `pkg/carriers/seur.go`, `pkg/carriers/gls_spain.go`, tests correspondientes.

---

### ISSUE #7: refactor(cainiao): Eliminación de hardcode estático 2026, robustez ante anti-bot y optimización del handover

- **Área:** Carriers — Global / Cross-Border  
- **Milestone:** Milestone 3 — Globales y Cross-Border  
- **Prioridad:** 🔴 Alta  
- **Etiquetas:** `bug`, `refactor`, `carriers`, `cainiao`

#### Contexto y Justificación
`pkg/carriers/cainiao.go` es el único adaptador que llama a una API externa (`global.cainiao.com`), pero tiene un bypass hardcodeado para la guía `32300035210642701471403` con fechas estáticas de 2026, y la función `generateSimulatedShipment()` que maquilla las caídas o códigos inexistentes devolviendo 7 eventos falsos.

#### Objetivos y Criterios de Aceptación
- [ ] Eliminar completamente la guía hardcodeada y `generateSimulatedShipment()`.
- [ ] Si la API de Cainiao devuelve código no encontrado (`ORDER_NOT_FOUND` o similar), retornar `ErrShipmentNotFound`.
- [ ] Manejar adecuadamente respuestas con protección anti-bot (desafíos WAF / captcha), devolviendo `ErrCarrierTemporarilyUnavailable`.
- [ ] Extraer fielmente el campo `LastMileCarrier` y `LastMileTrackingNumber` si la API los entrega (e.g. handover hacia Correos Express, Correos o CTT).
- [ ] Añadir suite de tests con JSONs reales capturados de envíos completados y en tránsito.

#### Implementación Técnica
- Archivos: `pkg/carriers/cainiao.go`, `pkg/carriers/cainiao_test.go`.

---

### ISSUE #8: feat(carriers): Adaptador B2B para DHL Express y DHL Paket con autenticación OAuth2 / API Key

- **Área:** Carriers — Global / Europa  
- **Milestone:** Milestone 3 — Globales y Cross-Border  
- **Prioridad:** 🟡 Media  
- **Etiquetas:** `enhancement`, `carriers`, `enterprise`

#### Contexto y Justificación
DHL (`dhl.go`) cuenta con una API oficial abierta para desarrolladores (DHL Developer Portal) con endpoints REST unificados para DHL Express y DHL Parcel/Paket. Debe admitir credenciales vía variables de entorno.

#### Objetivos y Criterios de Aceptación
- [ ] Implementar cliente para la API unificada de DHL (`api-eu.dhl.com/track/shipments`).
- [ ] Soporte de autenticación mediante `DHL_API_KEY`.
- [ ] Detección y distinción automática entre envíos Express (guías de 10 dígitos) y Parcel/Paket.
- [ ] Si la clave no está configurada y se solicita rastreo de DHL sin flag de simulación, devolver error informando que se requiere `DHL_API_KEY`.

#### Implementación Técnica
- Archivos: `pkg/carriers/dhl.go`, `pkg/carriers/dhl_test.go`.

---

### ISSUE #9: feat(carriers): Adaptador B2B para UPS y FedEx con soporte OAuth2 empresarial

- **Área:** Carriers — Global / Norteamérica  
- **Milestone:** Milestone 3 — Globales y Cross-Border  
- **Prioridad:** 🟡 Media  
- **Etiquetas:** `enhancement`, `carriers`, `enterprise`

#### Contexto y Justificación
UPS y FedEx han migrado sus sistemas de seguimiento exclusivamente a flujos OAuth2 (Client Credentials). Actualmente `ups.go` y `fedex.go` devuelven eventos ficticios.

#### Objetivos y Criterios de Aceptación
- [ ] Implementar cliente OAuth2 para UPS Track API (`onlinetools.ups.com/api/track/v1/details/`).
- [ ] Implementar cliente OAuth2 para FedEx Track API (`apis.fedex.com/track/v1/trackingnumbers`).
- [ ] Manejo transparente de renovación de token OAuth2 en memoria con caché de expiración.
- [ ] Configuración mediante variables de entorno: `UPS_CLIENT_ID`, `UPS_CLIENT_SECRET`, `FEDEX_CLIENT_ID`, `FEDEX_CLIENT_SECRET`.

#### Implementación Técnica
- Archivos: `pkg/carriers/ups.go`, `pkg/carriers/fedex.go`.

---

### ISSUE #10: perf(detector): Optimización de compilación de reglas regex, desambiguación y pre-filtrado por longitud

- **Área:** Detector Heurístico  
- **Milestone:** Milestone 4 — Motor Heurístico  
- **Prioridad:** 🟡 Media  
- **Etiquetas:** `performance`, `detector`, `algorithms`

#### Contexto y Justificación
`pkg/detector/detector.go` y `rules.go` evalúan secuencialmente cientos de reglas sobre cada llamada de detección. Algunos números exclusivamente numéricos (e.g. 10 o 12 dígitos) pueden coincidir con 10 o 15 transportistas diferentes sin un ranking de probabilidad claro.

#### Objetivos y Criterios de Aceptación
- [ ] Pre-compilar todas las expresiones regulares en la inicialización del detector (`regexp.MustCompile`).
- [ ] Implementar un índice de pre-filtrado por longitud de cadena (`len(cleanNumber)`) y conjunto de caracteres (solo números, alfanumérico, prefijo específico) para descartar el 80% de reglas antes de ejecutar regexes.
- [ ] Mejorar la ponderación heurística cuando se proporciona `opts.DestinationCountry` (aumentar el `ConfidenceScore` del transportista del país en +0.35).
- [ ] Agregar benchmarks (`BenchmarkDetect`) para asegurar una latencia de detección < 100 microsegundos por guía.

#### Implementación Técnica
- Archivos: `pkg/detector/detector.go`, `pkg/detector/rules.go`, `pkg/detector/detector_test.go`.

---

### ISSUE #11: feat(mcp): Notificaciones de progreso y herramientas avanzadas de diagnóstico de transportistas en Servidor MCP

- **Área:** Model Context Protocol (MCP)  
- **Milestone:** Milestone 5 — MCP y API  
- **Prioridad:** 🟢 Menor-Media  
- **Etiquetas:** `mcp`, `ai-integration`, `enhancement`

#### Contexto y Justificación
El servidor MCP actual expone `track_shipment` y `detect_carrier`. Para agentes de inteligencia artificial (Claude Desktop, agentes autónomos), es valioso poder inspeccionar la salud de los adaptadores (saber qué carriers están activos con API real y cuáles requieren configuración) y recibir trazas estructuradas.

#### Objetivos y Criterios de Aceptación
- [ ] Añadir herramienta MCP `check_carrier_status` que reporte si un carrier está disponible en modo `live_api`, `public_scraper` o `not_implemented`.
- [ ] Incluir en el resultado de `track_shipment` el metadato booleano `is_live_data` para que el LLM sepa si la información proviene de la red o es un mock de pruebas.
- [ ] Añadir soporte para el parámetro `postal_code` en las invocaciones de herramientas MCP.
- [ ] Tests en `pkg/mcp/server_test.go` cubriendo las nuevas herramientas y schemas JSON-RPC.

#### Implementación Técnica
- Archivos: `pkg/mcp/server.go`, `pkg/mcp/types.go`, `pkg/mcp/server_test.go`.

---

### ISSUE #12: feat(server): Compatibilidad estricta con Webhooks y sondeo asíncrono en ParcelsApp v4 API

- **Área:** REST API Daemon  
- **Milestone:** Milestone 5 — MCP y API  
- **Prioridad:** 🟢 Menor-Media  
- **Etiquetas:** `api`, `rest`, `parcelsapp`

#### Contexto y Justificación
La API v4 de ParcelsApp permite registrar peticiones de seguimiento asíncronas donde el cliente consulta el estado pasados unos segundos (`GET /trackings/:request_id`) o recibe un webhook. Actualmente, el servidor ejecuta la consulta de forma sincrónica en el mismo request.

#### Objetivos y Criterios de Aceptación
- [ ] Almacenar peticiones asíncronas en un almacén en memoria con expiración TTL (1 hora).
- [ ] Permitir que `POST /trackings` responde inmediatamente con `request_id` y estado `processing` si el rastreo incluye transportistas lentos.
- [ ] Implementar soporte de callback Webhook si se incluye la URL en el payload de `POST /trackings`.
- [ ] Validar compatibilidad contra la especificación en `docs/openapi.json`.

#### Implementación Técnica
- Archivos: `pkg/server/handlers.go`, `pkg/server/server.go`, `pkg/server/server_test.go`.

---

### ISSUE #13: docs: Actualización de README, matriz de transportistas reales vs catálogo y guía de gobernanza

- **Área:** Documentación y Gobernanza  
- **Milestone:** Milestone 6 — Gobernanza  
- **Prioridad:** 🟡 Media  
- **Etiquetas:** `documentation`, `governance`

#### Contexto y Justificación
Existe una disparidad entre lo publicitado en el `README.md` ("+1.500 transportistas disponibles") y la realidad técnica auditada (el 99% son plantillas sin conexión de red). Para que el proyecto tenga tracción de comunidad y credibilidad, la documentación debe reflejar con total transparencia la matriz de estado.

#### Objetivos y Criterios de Aceptación
- [ ] Crear una tabla en `docs/CARRIERS.md` con badges claros:
  - 🟢 **Live API / Verified**: Conexión de red operativa y probada (e.g. InPost, Cainiao).
  - 🟡 **In Development**: En proceso de scraping o desarrollo activo.
  - ⚪ **Catalog / Rule Only**: Regla de detección regex disponible, esperando contribución de adaptador.
- [ ] Actualizar `README.md` destacando la arquitectura modular y haciendo un llamamiento abierto a la comunidad ("Help us implement real adapters").
- [ ] Documentar en `CONTRIBUTING.md` el proceso de grabación de fixtures de tests para nuevos adaptadores.

#### Implementación Técnica
- Archivos: `README.md`, `docs/CARRIERS.md`, `CONTRIBUTING.md`.

---

## 4. Resumen de Despliegue de Issues

| Issue ID | Título Resumido | Prioridad | Milestone |
|---|---|---|---|
| **#1** | Erradicación de fallbacks falsos y errores canónicos | 🔴 Alta | Milestone 1 |
| **#2** | Especificación estándar `CarrierAdapter` y credenciales | 🔴 Alta | Milestone 1 |
| **#3** | Separación tests unitarios vs integración (`go:build`) | 🔴 Alta | Milestone 1 |
| **#4** | Integración real Correos Express | 🟠 Media-Alta | Milestone 2 |
| **#5** | Integración real Correos España | 🟠 Media-Alta | Milestone 2 |
| **#6** | Integración real SEUR y GLS Spain con código postal | 🟡 Media | Milestone 2 |
| **#7** | Refactor Cainiao (eliminación hardcode 2026 y mock) | 🔴 Alta | Milestone 3 |
| **#8** | Adaptador B2B DHL Express / Paket (API Key) | 🟡 Media | Milestone 3 |
| **#9** | Adaptador B2B UPS y FedEx (OAuth2) | 🟡 Media | Milestone 3 |
| **#10** | Optimización y pre-filtrado de reglas en `pkg/detector` | 🟡 Media | Milestone 4 |
| **#11** | Nuevas herramientas MCP y diagnóstico de estado | 🟢 Menor | Milestone 5 |
| **#12** | Sondeo asíncrono y webhooks en ParcelsApp API v4 | 🟢 Menor | Milestone 5 |
| **#13** | Transparencia de matriz de cobertura y actualización de docs | 🟡 Media | Milestone 6 |

---
*Documento generado por el Administrador del Sistema Hermes Agent para el repositorio gek0v/open-parcels.*
