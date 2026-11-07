# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.2] - 2026-11-07

### Added
- **Live InPost / Mondial Relay Integration**: Consulta HTTP real contra inpost.es mediante SSR Astro Island, extrayendo hitos auténticos, fechas verídicas y código de evento (ej. CRE.1001).
- **Soporte de Código Postal de Destino**: Flags CLI `--postal-code` (`-z`), soporte en endpoints REST y herramientas MCP. Validación estricta para carriers que requieren código postal.

### Fixed
- **Eliminación de Mocks Silenciosos**: Fin de los datos inventados con `time.Now()`. Si una guía no existe o el CP es erróneo, el sistema devuelve error legítimo `NOT_FOUND` o `MISSING_POSTAL_CODE`.
- **Corrección Semántica en Normalizador**: Distinción estricta de 'being delivered' / 'to be delivered' como `OUT FOR DELIVERY` (85%) evitando falsos positivos de estado `DELIVERED` (100%).

## [1.0.0] - 2026-11-05

### Added

#### Core Engine & Architecture
- **Universal Parcel Tracking Engine**: Modular, high-performance tracking orchestrator written in Go.
- **Intelligent Carrier Auto-Detection**:
  - Regular expression and prefix heuristics with confidence scoring (0.0 to 1.0).
  - Mathematical checksum algorithms:
    - **UPU S10 Modulo 11** (international postal items).
    - **UPS 1Z Modulo 10** alphanumeric checksum.
    - **IATA AWB Modulo 7** air cargo waybills.
    - **ISO 6346** ocean container BIC identification and check digit calculation.
  - Destination country hint reinforcement (`--country`) for disambiguation.
- **Multi-Carrier Journey & Event Merging**:
  - End-to-end tracking of international origin legs, customs handovers, and domestic last-mile delivery legs (e.g., Cainiao ➔ Correos Express, YunExpress ➔ Correos España, SpeedPAK ➔ Local Courier).
  - Chronological descending event fusion with 2-hour sliding window deduplication.
- **Universal State Normalization**:
  - 8 canonical normalized package statuses: `pending`, `info_received`, `in_transit`, `out_for_delivery`, `delivered`, `failed_attempt`, `exception`, `expired`.
  - Dynamic shipment progress percentage calculation (0% to 100%).
  - Localized status descriptions and milestones in English (`en`) and Spanish (`es`).

#### Carrier Catalogs (+1,500 Operators across 17 Regional Groups)
- **China & Hong Kong** (Issue #1, #21): 271 express, e-commerce, and courier networks.
- **Global & Cross-Border Logistics** (Issue #2, #22): 595 global freight forwarders and cross-border consolidators.
- **Asia-Pacific & Oceania / APAC** (Issue #3, #23): 125 domestic and regional postal services and logistics providers.
- **Central & Eastern Europe / CEE & Balkans** (Issue #4, #24): 81 carriers covering Poland, Czechia, Romania, Hungary, and Balkan states.
- **North America** (Issue #5, #25): 42 postal and private couriers across the United States and Canada (USPS, FedEx, UPS, OnTrac, Purolator, Canada Post).
- **Nordic & Baltic Regions** (Issue #6, #26): 23 carriers covering Sweden, Norway, Denmark, Finland, Estonia, Latvia, and Lithuania (PostNord, Bring, Omniva, Itella).
- **Spain** (Issue #7, #19): 24 domestic parcel and express operators (Correos, Correos Express, SEUR, MRW, GLS Spain, CTT Express, Tipsa, Zeleris, Paack).
- **Air Cargo & Airlines** (Issue #8, #27): 136 international commercial airlines and dedicated cargo operators with IATA AWB prefix mapping.
- **Italy & San Marino** (Issue #9, #20): 19 national carriers (Poste Italiane, SDA, BRT Bartolini, Nexive, GLS Italy).
- **France & Monaco** (Issue #10, #28): 22 operators (La Poste, Colissimo, Chronopost, Mondial Relay, Relais Colis).
- **United Kingdom & Ireland** (Issue #11, #29): 38 carriers (Royal Mail, Parcelforce, Evri / Hermes UK, DPD UK, Yodel, An Post).
- **Middle East, Central Asia & Africa / MENA** (Issue #12, #30): 57 operators across GCC, Levant, North Africa, Sub-Saharan Africa, and Central Asia.
- **Germany & DACH Region** (Issue #13, #31): 26 carriers covering Germany, Austria, and Switzerland (DHL Paket, Deutsche Post, Hermes Germany, DPD Germany, Österreichische Post, Swiss Post).
- **Latin America / LATAM** (Issue #14, #32): 25 domestic and regional carriers across Mexico, Brazil, Colombia, Argentina, and Chile (Correios Brasil, Estafeta, Mercado Envíos, Chilexpress, Andreani).
- **Ocean Freight & Shipping Lines** (Issue #15, #33): 49 global maritime container shipping lines (Maersk, MSC, CMA CGM, COSCO, Hapag-Lloyd, ONE, Evergreen).
- **Benelux** (Issue #16, #34): 18 carriers covering Belgium, Netherlands, and Luxembourg (PostNL, bpost, DHL Parcel NL, DPD Belux).
- **Portugal** (Issue #17, #18): Comprehensive Portuguese postal and courier support (CTT Correios de Portugal, CTT Express PT).

#### Interfaces & Consumption Modes
- **Interactive Terminal CLI**:
  - Visual tracking view with ANSI styling, progress meters, and milestone timelines.
  - Commands: `track`, `detect`, `carriers`, `serve`, `mcp`, `version`, `help`.
  - Plain `--json` output flag for scriptable pipeline integration.
- **REST API Server**:
  - High-throughput HTTP API daemon (`open-parcels serve`).
  - Native endpoints: `POST /api/v1/track`, `GET /api/v1/carriers`, `GET /api/v1/detect`.
  - **ParcelsApp v4 API Compatibility**: `POST /trackings` and polling `GET /trackings/:request_id`.
- **Model Context Protocol (MCP) Server**:
  - stdio JSON-RPC 2.0 interface for AI assistants (Claude Desktop, Cursor, Pi).
  - Streamable HTTP `/mcp` endpoint for remote agent integration.
  - Structured tools: `track_package`, `detect_carrier`, `list_carriers`.

#### Documentation & Quality Assurance
- Complete test suite passing across all packages (`pkg/carriers`, `pkg/detector`, `pkg/engine`, `pkg/mcp`, `pkg/server`).
- Comprehensive documentation suite:
  - System Architecture (`docs/ARCHITECTURE.md`)
  - REST API Reference (`docs/API.md`)
  - MCP Integration Guide (`docs/MCP.md`)
  - Carrier Catalog Reference (`docs/CARRIERS.md`)
  - Contribution Standards (`CONTRIBUTING.md`)
  - Security Disclosure Policy (`SECURITY.md`)
