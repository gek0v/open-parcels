# Auditoría de Mocks, Simulación y Fallbacks en Producción

**Proyecto:** Open-Parcels  
**Ruta auditada:** `D:\Workspaces\open-parcels`  
**Alcance:** Código de producción en `pkg/carriers/`, `pkg/engine/` y `pkg/server/` (excluyendo tests `*_test.go`).  
**Fecha:** Febrero 2025  
**Estado general:** ⚠️ **CRÍTICO** — El 99.9% de los adaptadores de carriers (más de 1.570 transportistas) operan como **Mocks puros** en memoria sin conexión a redes ni APIs reales, devolviendo siempre estados exitosos ficticios ("En reparto" / "En tránsito") basados en `time.Now()`.

---

## 1. Resumen Ejecutivo y Diagnóstico Global

1. **Total de carriers registrados en el sistema:** **1.575+ transportistas**.
2. **Carriers con conexión real a API/HTTP:** **1 solo** (`cainiao.go`).
   - Incluso `cainiao.go` posee:
     - Un cortocircuito hardcodeado para la guía `32300035210642701471403` con eventos estáticos fijados en **noviembre de 2026**.
     - La función `generateSimulatedShipment()`, que inyecta una traza internacional ficticia de 7 eventos con `time.Now().Add(-X)` cada vez que la API real falla, no responde o no encuentra el paquete.
3. **Carriers con MOCKS PUROS (Sin HTTP ni scraping):**
   - **55 adaptadores dedicados** (SEUR, Correos, Correos Express, InPost, MRW, Nacex, GLS, DHL, FedEx, UPS, USPS, etc.).
   - **14 catálogos genéricos regionales/sectoriales** (Air Cargo, APAC, Benelux, CEE, China/HK, DACH, Francia, Global Cross-border, LATAM, MENA/África, Nórdicos/Bálticos, Norteamérica, Marítimo, UK/Irlanda) que registran **1.511 carriers**.
4. **Ningún archivo de carrier salvo `cainiao.go` y `http_client.go` importa `net/http`**.
5. **Efecto de encubrimiento:**
   - Si un usuario ingresa una guía inexistente (`00000000`, `FAKE123456`), el sistema devuelve **HTTP 200 OK**, con progreso del 85%, paquete "En reparto" y eventos simulados de recogida, clasificación y tránsito.
   - En `pkg/engine/merger.go` (línea 40), existe código explícito para **descartar eventos posteriores a una entrega**, introducido específicamente para ocultar que los mocks generados con `time.Now()` colisionaban cronológicamente con eventos entregados reales o estáticos.

---

## 2. Hallazgos Críticos Específicos

### 2.1. `cainiao.go` — API real con bypass hardcodeado y fallback simulado
- **Archivo:** `pkg/carriers/cainiao.go`
- **Líneas críticas:**
  - Línea 234–240: Intento de llamada a `https://global.cainiao.com/global/detail.json`.
  - Línea 440: `return c.generateSimulatedShipment(trackingNumber, opts), nil`.
  - Líneas 443–675: Implementación de `generateSimulatedShipment()`.
  - Líneas 460–573: Detección fija de la guía `32300035210642701471403`, devolviendo 6 eventos fijos fechados en `2026-11-03` y `2026-11-04` con estado `models.StatusDelivered`.
  - Líneas 576–675: Generación de 7 eventos simulados relativos a `time.Now()` (Shenzhen -> Madrid Barajas -> Hub Coslada -> Correos Express) y síntesis de un falso `LastMileTrackingNumber` (`6900...`).
- **Impacto:** Oculta caídas de red, bloqueos anti-bot o guías inexistentes de Cainiao. En lugar de devolver error, simula un envío perfecto en tránsito (progreso 75%).

### 2.2. `correosexpress.go` — Hardcode 2026 y Mock puro por defecto
- **Archivo:** `pkg/carriers/correosexpress.go`
- **Líneas críticas:**
  - Líneas 79–154: Hardcode especial para la guía `32300035210642701471403` con timestamps fijos de noviembre de 2026 (`2026-11-03T18:00:45Z` a `2026-11-04T17:45:05Z`).
  - Líneas 157–217: Para cualquier otra guía de Correos Express, no hace ninguna consulta externa; genera 4 eventos con `now := time.Now()`:
    - `now.Add(-18 * time.Hour)`: "Sin recibir - Envío informado electrónicamente" (Madrid)
    - `now.Add(-12 * time.Hour)`: "En tránsito - Envío clasificado en nave central" (Coslada)
    - `now.Add(-6 * time.Hour)`: "En tránsito - Llegada a delegación de destino" (Barcelona)
    - `now.Add(-2 * time.Hour)`: "En reparto - Envío asignado al repartidor para entrega" (Barcelona)
  - Línea 225: Retorna `models.StatusOutForDelivery`, progreso 85% y `error == nil`.
- **Impacto:** Correos Express nunca falla ni reporta paquetes no encontrados.

### 2.3. `inpost.go` e `inpost_es.go` — Mocks puros de taquillas / Lockers
- **Archivos:**
  - `pkg/carriers/inpost.go` (líneas 28–66)
  - `pkg/carriers/inpost_es.go` (líneas 28–74)
- **Slug / Carrier:** `inpost` ("InPost / Mondial Relay") e `inpost-es` ("InPost Spain").
- **Eventos inyectados:**
  - `now.Add(-30h)` / `now.Add(-28h)`: "Parcel registered by sender for locker drop-off" / "Envío registrado por remitente para Punto Pack o Locker"
  - `now.Add(-18h)` / `now.Add(-16h)`: "Parcel collected from locker and in transit to sorting depot" / "Paquete recogido y en tránsito a nave de clasificación"
  - `now.Add(-4h)` / `now.Add(-3h)`: "Parcel is being delivered to destination locker / Punto Pack" / "Paquete en reparto hacia Punto Pack / Locker de destino"
  - Estado: `models.StatusOutForDelivery`, Progreso: 85%.
- **Impacto:** Cualquier código de 7, 8 o 24 dígitos aparece inmediatamente como "En reparto a locker" sin consultar la API pública de InPost ni sus endpoints móviles.

---

## 3. Censo de Adaptadores Dedicados (55 Archivos con Mock Puro)

Todos los archivos listados a continuación devuelven estructuras `*models.Shipment` construidas en memoria mediante `now := time.Now()`, asignando estados fijos `StatusOutForDelivery` (85%) o `StatusInTransit` (50–75%) y `err = nil`:

| Archivo | Carrier / Slug | Línea Track | Eventos Simulados | Ubicaciones Ficticias |
|---|---|---|---|---|
| `activos24.go` | `activos24` (Activos24) | L28 | 3 eventos (`now.Add(-20h, -14h, -3h)`) | Madrid, Coslada Hub |
| `aircargo.go` | `air-cargo` (IATA Air Cargo) | L28 | 2 eventos (`now.Add(-24h, -6h)`) | Origin Airport, Hub |
| `amazon_italy.go` | `amazon-it` (Amazon Italy) | L29 | 3 eventos (`now.Add(-24h, -14h, -3h)`) | Milano Hub, Roma |
| `asendia_es.go` | `asendia-es` (Asendia Spain) | L28 | 3 eventos (`now.Add(-36h, -20h, -5h)`) | Madrid Barajas |
| `asigna.go` | `asigna-es` (Asigna) | L28 | 3 eventos (`now.Add(-20h, -12h, -3h)`) | Madrid Delegación |
| `brt.go` | `brt-bartolini`, `brt-italy-parcelid`, `mybrt-it` | L29, L100, L171 | 3 a 6 eventos por método | Bologna Hub, Milano |
| `celeritas.go` | `celeritas` (Celeritas Transporte) | L28 | 3 eventos (`now.Add(-24h, -14h, -3h)`) | Madrid, Punto Celeritas |
| `chinapost.go` | `china-post` (China Post) | L28 | 7 eventos (`now.Add(-7d..-12h)`) | Beijing, Transit Airport |
| `cne.go` | `cne-express` (CNE Express) | L28 | 6 eventos (`now.Add(-6d..-18h)`) | Shanghai, Transit Hub |
| `container.go` | `ocean-container` (ISO Container) | L28 | 3 eventos (`now.Add(-10d, -5d, -1d)`) | Port of Loading, At Sea |
| `correos.go` | `correos-spain` (Correos España) | L28 | 4 eventos (`now.Add(-24h, -16h, -8h, -2h)`) | Madrid, Unidad de Reparto |
| `ctt.go` | `ctt-express` (CTT Express) | L28 | 3 eventos (`now.Add(-18h, -10h, -2h)`) | Madrid CTM, Reparto |
| `delnext.go` | `delnext` (Delnext) | L28 | 3 eventos (`now.Add(-22h, -12h, -3h)`) | Lisboa Hub, Reparto |
| `deppon.go` | `deppon` (Deppon) | L28 | 6 eventos (`now.Add(-5d..-14h)`) | Guangzhou, Transit Hub |
| `dhl.go` | `dhl-express`, `dhl-parcel` | L28, L98 | 2 a 5 eventos (`now.Add(-24h..-2h)`) | Leipzig Hub, Local Hub |
| `dhl_parcel_es.go` | `dhl-parcel-es` (DHL Parcel Spain) | L28 | 3 eventos (`now.Add(-18h, -10h, -2h)`) | Madrid Hub, Reparto |
| `dpd_portugal.go` | `dpd-portugal` (DPD Portugal) | L28 | 3 eventos (`now.Add(-20h, -12h, -3h)`) | Lisboa Hub, Porto |
| `envialia.go` | `envialia` (Envialia) | L28 | 3 eventos (`now.Add(-18h, -10h, -2h)`) | Coslada, Plataforma |
| `fedex.go` | `fedex` (FedEx) | L28 | 3 eventos (`now.Add(-24h, -12h, -3h)`) | Paris CDG Hub, Reparto |
| `fourpx.go` | `4px` (4PX Express) | L28 | 4 eventos (`now.Add(-5d..-20h)`) | Shenzhen, HK Airport |
| `gls.go` | `gls` (GLS) | L28 | 3 eventos (`now.Add(-20h, -12h, -3h)`) | Neuenstein, Local Depot |
| `gls_italy.go` | `gls-italy` (GLS Italy) | L29 | 3 eventos (`now.Add(-22h, -14h, -3h)`) | Milano Hub, Sede Destino |
| `gls_spain.go` | `asm-gls-spain` (GLS Spain ASM) | L28 | 3 eventos (`now.Add(-18h, -10h, -2h)`) | Madrid Central, Agencia |
| `hongkongpost.go` | `hongkong-post` (Hong Kong Post) | L28 | 7 eventos (`now.Add(-6d..-14h)`) | Hong Kong AMC, Transit |
| `inpost.go` | `inpost` (InPost / Mondial Relay) | L28 | 3 eventos (`now.Add(-30h, -18h, -4h)`) | Locker / Punto Pack |
| `inpost_es.go` | `inpost-es` (InPost Spain) | L28 | 3 eventos (`now.Add(-28h, -16h, -3h)`) | Madrid Centro Clasificación |
| `inpost_italy.go` | `inpost-it` (InPost.it) | L29 | 3 eventos (`now.Add(-26h, -15h, -3h)`) | Milano Locker |
| `instapack.go` | `instapack-es` (Instapack) | L28 | 3 eventos (`now.Add(-16h, -8h, -2h)`) | Madrid Express Hub |
| `italy_logistics.go` | 10 carriers (`alfred-it`, `ceva-it`, etc.) | L29..L614 | 3 a 6 eventos por carrier | Roma, Milano, Bologna |
| `mbe_es.go` | `mbe-es` (Mail Boxes Etc. Spain) | L28 | 3 eventos (`now.Add(-18h, -10h, -2h)`) | Centro MBE Madrid |
| `mrw.go` | `mrw-spain` (MRW) | L28 | 3 eventos (`now.Add(-18h, -10h, -2h)`) | Plataforma Coslada |
| `nacex.go` | `nacex-spain` (Nacex) | L28 | 3 eventos (`now.Add(-18h, -10h, -2h)`) | Coslada Hub 2801 |
| `ontime.go` | `ontime` (Ontime) | L28 | 3 eventos (`now.Add(-20h, -12h, -3h)`) | Ontime Hub Central |
| `paack.go` | `paack` (Paack Logistics) | L28 | 3 eventos (`now.Add(-16h, -8h, -2h)`) | Paack Distribution Center |
| `packlink.go` | `packlink` (Packlink) | L28 | 3 eventos (`now.Add(-20h, -12h, -3h)`) | Integrated Carrier Network |
| `portugal_post.go` | `portugal-post` (Portugal Post CTT) | L28 | 4 eventos (`now.Add(-24h..-2h)`) | Lisboa CTC, Distribuição |
| `poste_italiane.go` | `poste-italiane` (Poste Italiane) | L29 | 3 eventos (`now.Add(-24h, -14h, -3h)`) | Centro Scambi Roma |
| `redur.go` | `redur-spain`, `redur-tracking`, etc. | L29, L52, L75 | 3 eventos compartidos | Algete Nave Central |
| `royalmail.go` | `royal-mail`, `colissimo` | L28, L94 | 3 a 6 eventos | Heathrow / Paris Hub |
| `sailpost.go` | `sailpost` (Sailpost) | L29 | 3 eventos (`now.Add(-22h, -12h, -3h)`) | Pisa Hub, Agenzia |
| `sending.go` | `sending-es` (Sending) | L28 | 3 eventos (`now.Add(-18h, -10h, -2h)`) | Nave Central Coslada |
| `seur.go` | `seur` (SEUR) | L28 | 3 eventos (`now.Add(-20h, -12h, -2h)`) | Unidad Operativa Madrid |
| `sfexpress.go` | `sf-express` (SF Express) | L28 | 7 eventos (`now.Add(-6d..-12h)`) | Shenzhen Hub, Airport |
| `sunyou.go` | `sytrack` (SunYou - Sytrack) | L28 | 6 eventos (`now.Add(-6d..-16h)`) | Shenzhen Sorting |
| `tipsa.go` | `tipsa` (TIPSA) | L28 | 3 eventos (`now.Add(-18h, -10h, -2h)`) | Hub Central Madrid |
| `tnt_italy.go` | `tnt-italy` (TNT Italy) | L29 | 3 eventos (`now.Add(-22h, -12h, -3h)`) | Peschiera Borromeo Hub |
| `ups.go` | `ups` (UPS) | L28 | 4 eventos (`now.Add(-24h..-3h)`) | Koeln Hub, Delivery Facility |
| `usps.go` | `usps` (USPS) | L28 | 4 eventos (`now.Add(-24h..-3h)`) | Regional Facility |
| `wepickup.go` | `wepickup-pt` (WePickUp) | L28 | 3 eventos (`now.Add(-18h, -8h, -2h)`) | Lisboa Hub |
| `yanwen.go` | `yanwen` (Yanwen Logistics) | L28 | 7 eventos (`now.Add(-7d..-14h)`) | Beijing Sorting Center |
| `yto.go` | `yto-express` (YTO Express) | L28 | 5 eventos (`now.Add(-5d..-12h)`) | Shanghai Transit |
| `yunexpress.go` | `yunexpress` (YunExpress) | L28 | 7 eventos (`now.Add(-6d..-14h)`) | Shenzhen Facility |
| `zeleris.go` | `zeleris` (Zeleris) | L28 | 3 eventos (`now.Add(-18h, -10h, -2h)`) | Centro Logístico Madrid |

---

## 4. Censo de Catálogos Genéricos Regionales (14 Archivos, 1.511+ Carriers)

Los siguientes 14 archivos implementan structs `Generic<Region>Carrier` que instancian masivamente cientos de transportistas con un método `Track()` común que devuelve eventos simulados basados en `time.Now()`:

| Archivo | Struct Genérico | Nº Carriers Registrados | Ejemplos de Carriers Afectados |
|---|---|---|---|
| `air_cargo_catalog.go` | `GenericAirCargoCarrier` | **139** | Qatar Airways, Emirates SkyCargo, Lufthansa Cargo, Iberia Cargo |
| `apac_catalog.go` | `GenericAPACCarrier` | **125** | Australia Post, Japan Post, Delhivery, Bluedart, CJ Logistics |
| `benelux_catalog.go` | `GenericBeneluxCarrier` | **18** | Bpost Belgium, Bringly, Budbee, PostNL |
| `cee_catalog.go` | `GenericCEECarrier` | **81** | 5Post, A1 Post, ACS Courier, Fan Courier |
| `china_hongkong_catalog.go` | `GenericCNHKCarrier` | **271** | 139 Express, 360Lion, J&T Express, Cainiao Domestic |
| `dach_catalog.go` | `GenericDACHCarrier` | **26** | Deutsche Post / DHL Paket, DPD Germany, Swiss Post, Dachser |
| `france_catalog.go` | `GenericFranceCarrier` | **22** | Chronopost, Agediss, Amazon Shipping France, Colis Privé |
| `global_crossborder_catalog.go` | `GenericGlobalCarrier` | **595** | Landmark Global, Asendia Global, Aramex, Pitney Bowes |
| `latam_catalog.go` | `GenericLATAMCarrier` | **25** | Andreani, Correo Argentino, Correios Brasil, Servientrega |
| `mena_africa_catalog.go` | `GenericMENACarrier` | **57** | AJEX, AKIS Express, Egypt Post, Saudi Post |
| `nordic_baltic_catalog.go` | `GenericNordicCarrier` | **23** | PostNord, Bring Sweden, Omniva Estonia, Matkahuolto |
| `north_america_catalog.go` | `GenericNACarrier` | **42** | Purolator, Canada Post, OnTrac, Spee-Dee |
| `ocean_freight_catalog.go` | `GenericOceanCarrier` | **49** | Maersk, MSC, CMA CGM, Hapag-Lloyd, Evergreen, Cosco |
| `uk_ireland_catalog.go` | `GenericUKCarrier` | **38** | Evri / Hermes, DPD UK, Yodel, An Post |
| **TOTAL** | | **1.511 carriers** | |

---

## 5. Mecanismos de Ocultamiento en `engine` y `server`

### 5.1. `pkg/engine/merger.go` (Línea 40): Eliminación de eventos posteriores a entrega
```go
// If shipment has a delivered event, discard any non-delivered anomalies/mocks with timestamps after delivery
var deliveredDate *time.Time
for _, ev := range all {
    if ev.Status == models.StatusDelivered {
        d := ev.Date
        if deliveredDate == nil || d.After(*deliveredDate) {
            deliveredDate = &d
        }
    }
}
if deliveredDate != nil {
    filtered := make([]models.TrackingEvent, 0, len(all))
    for _, ev := range all {
        if ev.Date.After(*deliveredDate) && ev.Status != models.StatusDelivered {
            continue
        }
        filtered = append(filtered, ev)
    }
    all = filtered
}
```
**Diagnóstico:** El propio comentario del motor admite que los mocks secundarios generan eventos con `time.Now()` que quedan situados en el futuro o después de la entrega. En vez de subsanar los mocks, se implementó un filtro de limpieza que borra los eventos anómalos.

### 5.2. `pkg/engine/engine.go` (Líneas 65–75 y 147–175): Chaining de Carriers Ficticios
Cuando `cainiao.go` genera una simulación, asigna `LastMileCarrier: "correosexpress"` y una guía ficticia (`6900...`). El motor invoca automáticamente a `correosexpress.Track()`, el cual a su vez dispara **otro mock**, y ambos se fusionan, entregando al usuario una respuesta en la que dos compañías diferentes parecen haber reportado tracking real coordinado.

### 5.3. `pkg/server/handlers.go`: Inmunidad ante errores y 100% de falsos positivos
En los endpoints `/api/v1/track` y `/trackings` (compatible con ParcelsApp v4), las llamadas a `engine.Track` nunca fallan porque los adaptadores jamás retornan error. Cualquier número aleatorio que coincida con una expresión regular devuelve un envío real aparente, lo que impide detectar códigos de seguimiento mal introducidos, paquetes devueltos al remitente o incidencias graves de logística.

---

## 6. Requerimientos Técnicos para Migrar a Integraciones Reales

Para que los adaptadores funcionen de verdad, se debe erradicar la simulación estática y reemplazarla por:

1. **Adaptadores de Acceso Público (Web Scraping / Endpoints Reverse-Engineered):**
   - **InPost (`inpost.es` / `mondialrelay`):** Utilizar el endpoint público `https://inpost.es/api/shipment/...` o el endpoint de la app móvil. Requiere cookies de sesión y parseo de JSON.
   - **Correos Express Web:** Scraping del formulario `consultanos?shippingNumber=%s` o la API REST interna utilizada por su SPA pública.
   - **SEUR / GLS / MRW:** Scrapers dedicados contra los portales de seguimiento con soporte para tokens CSRF y cabeceras de emulación de navegador.
2. **Adaptadores Empresariales (B2B APIs con Credenciales):**
   - **DHL Express:** Requiere API Key y Secret en el portal DHL Developer (REST Tracking API).
   - **UPS:** Requiere OAuth2 Client ID/Secret y suscripción a UPS Tracking API.
   - **FedEx:** Requiere FedEx Developer Portal API Key & Secret.
   - **Nacex / Correos / CTT:** Frecuentemente requieren código de cliente, password SOAP/REST y el **código postal de destino** (`opts.PostalCode`) como factor obligatorio de autenticación.
3. **Manejo Estricto de Errores y Estados Reales:**
   - Si la guía no existe en el carrier: Retornar error estructurado `ErrShipmentNotFound` -> HTTP 404.
   - Si el carrier requiere código postal y no se suministró: Retornar error de campos faltantes (`ErrMissingPostalCode`).
   - Si el servicio del carrier está caído: Retornar `ErrCarrierUnavailable` -> HTTP 502/504, **nunca** un mock de éxito ficticio.
4. **Bandera Explícita de Simulación (`--allow-simulated` / `OP_MOCK_FALLBACK=0`):**
   - Si se desean mantener mocks para pruebas locales o CI, deben quedar confinados a archivos `*_test.go` o condicionados a una variable de entorno explícita de desarrollo (ej. `OPENPARCELS_ENABLE_SIMULATION=true`), estando deshabilitados por defecto en entornos de producción.
