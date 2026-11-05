# Open Parcels Architecture Document

This document provides a comprehensive overview of the design principles, domain models, algorithms, and modular components of **open-parcels**.

---

## 1. System Overview & Philosophy

`open-parcels` is designed as a self-hosted, modular, high-performance logistics tracking engine and CLI in Go. It operates with zero mandatory external SaaS dependencies, offering complete feature parity with the official **ParcelsApp API v4**.

```
                       ┌───────────────────────────────────────────────┐
                       │               Interfaces                      │
                       │   CLI (terminal)  │  REST API  │  MCP Server  │
                       └───────────────────────┬───────────────────────┘
                                               │
                                               ▼
                       ┌───────────────────────────────────────────────┐
                       │           Engine Orchestrator                 │
                       │           (pkg/engine/engine.go)              │
                       └───────┬───────────────────────────────┬───────┘
                               │                               │
            ┌──────────────────┴─────────────┐   ┌─────────────┴─────────────────┐
            │   Carrier Auto-Detection       │   │   Carrier Registry & Adapters │
            │   (pkg/detector)               │   │   (pkg/carriers)              │
            │   - Regex & Prefixes           │   │   - Flagship Adapters         │
            │   - Checksums (Luhn, Mod 7/11) │   │   - 17 Regional Catalogs      │
            │   - Confidence Scoring         │   │   - Handover Chaining         │
            └────────────────────────────────┘   └───────────────────────────────┘
                               │                               │
                               ▼                               ▼
                       ┌───────────────────────────────────────────────┐
                       │          Event Merger & Normalizer            │
                       │   - Chronological Sorting (Descending)        │
                       │   - Windowed Deduplication (2-hour limit)     │
                       │   - Universal Status Normalization            │
                       │   - 0-100% Progress Calculation               │
                       └───────────────────────────────────────────────┘
```

---

## 2. Core Domain Models (`pkg/models`)

The system centers on a set of clean, serializable domain structs:

### 2.1 Universal Status Machine
Shipment milestones are normalized to eight canonical states:
- `pending`: Registered or awaiting first carrier update.
- `info_received`: Electronic shipping data received / label printed.
- `in_transit`: Package moving through transit hubs, customs, or sorting depots.
- `out_for_delivery`: Package loaded onto courier vehicle for delivery today.
- `delivered`: Shipment successfully delivered to recipient or mailbox.
- `failed_attempt`: Delivery attempt failed (recipient absent, wrong address, notice left).
- `exception`: Customs retention, delay, damage, or delivery exception.
- `expired`: Tracking record stale or inactive.

### 2.2 Lifecycle Stages
Each tracking milestone is tagged with its lifecycle stage:
- `created`: Origin preparation.
- `picked_up`: Physical collection.
- `customs`: Export or import clearance.
- `transit`: Inter-depot or intercontinental linehaul.
- `handover`: International-to-local partner transfer.
- `last_mile`: Delivery unit processing and courier route.
- `delivery`: Final drop-off.
- `exception`: Incident handling.

---

## 3. Two-Tier Carrier Architecture

To support **more than 1,500 carriers worldwide** without code bloat or excessive binary weight, `open-parcels` employs a two-tier architecture:

### Tier 1: Dedicated Flagship Adapters
Major logistics providers feature dedicated adapter structs implementing `Carrier` with bespoke API scrapers or detailed lifecycle generators:
- **China & Cross-Border**: Cainiao Global (`cainiao`), China Post (`china-post`), SF Express (`sf-express`), Hong Kong Post (`hongkong-post`), Yanwen (`yanwen`), SunYou (`sytrack`), CNE Express (`cne-express`), YTO Express (`yto-express`), Deppon (`deppon`), 4PX (`4px` / `four-px`), YunExpress (`yunexpress`).
- **Iberia**: Correos España (`correos-spain`), Correos Express (`correos-express`), CTT Express (`ctt-express`), SEUR (`seur`), Paack (`paack`), MRW (`mrw`).
- **North America**: USPS (`usps`), UPS (`ups`), FedEx (`fedex`), DHL Express (`dhl-express`), DHL Parcel (`dhl-parcel`).
- **Europe & DACH**: Royal Mail (`royal-mail`), Colissimo / La Poste (`colissimo`), InPost / Mondial Relay (`inpost`), GLS (`gls`).
- **Freight & Cargo**: IATA Air Cargo (`air-cargo`), ISO Ocean Containers (`ocean-container`).

### Tier 2: Catalog-Driven Regional Registries
All remaining regional carriers are organized across 17 catalogs covering every geographic territory. Each regional catalog provides metadata, slug lookups, tracking URL templates, and localized event lifecycles.

---

## 4. Carrier Auto-Detection Engine (`pkg/detector`)

When a user submits a tracking number without a `--carrier` flag or `carrier_hint`, the detection engine inspects the number:

1. **Cleaning**: Strip white-space, non-printable characters, and dashes (unless it matches the IATA Air Waybill format `NNN-NNNNNNNN`).
2. **Rule Matching**: Evaluate pattern rules ordered by priority.
3. **Checksum Verification**:
   - **UPU S10 (Mod 11)**: Two letters + 8 digits + 1 check digit + 2 country letters (e.g. `EA123456785CN`).
   - **UPS 1Z (Mod 10)**: Validates weighted alphanumeric sum of chars 2..16 against check digit at index 17.
   - **IATA Air Cargo (Mod 7)**: 3-digit prefix + hyphen + 7 serial digits + 1 check digit (`serial % 7 == check`).
   - **ISO 6346 Container**: 4 letters (ending in U, J, or Z) + 7 digits.
4. **Confidence Scoring**:
   - Base confidence (0.80 - 0.99).
   - Country hint boost (+0.15 if destination country matches carrier country).
   - Checksum bonus (+0.05 if check digit verified, -0.20 if check digit fails).
5. **Ranking**: Results are sorted descending by confidence score.

---

## 5. Multi-Carrier Journey Chaining & Merging (`pkg/engine`)

For cross-border packages (e.g. AliExpress from China to Europe), tracking involves two distinct logistics phases:
1. **International Leg**: Origin collection, export customs, airline uplift, destination customs.
2. **Handover**: International partner delivers to local depot.
3. **Last-Mile Leg**: Local domestic courier (e.g. Correos Express, CTT, GLS, SEUR) distributes and delivers the parcel.

### Resolution Flow:
```text
Track(trackingNumber)
  │
  ├─ 1. Primary Carrier Track()
  │     └─ Resolves international events + lastMileTrackingNumber + lastMileCarrier
  │
  ├─ 2. Secondary Carrier Track() (Concurrent, bounded 4s timeout)
  │     └─ Resolves domestic destination events
  │
  ├─ 3. MergeEvents(primary, secondary)
  │     ├─ Annotate carrier index (0 = primary, 1 = secondary)
  │     ├─ Sort chronologically descending
  │     └─ Deduplicate equivalent events within 2-hour window
  │
  └─ 4. Normalize & Progress Calculation
        ├─ DeriveOverallStatus()
        ├─ CalculateProgress() (0 - 100%)
        └─ CalculateDaysInTransit()
```

---

## 6. Progress Calculation Algorithm

Progress percentage is computed deterministically based on verified milestone stages:
- **`pending`**: 5%
- **`info_received`**: 15%
- **`in_transit`**:
  - Departed origin: 35%
  - Customs cleared: 55%
  - Handed over to local partner: 75%
- **`out_for_delivery`**: 85%
- **`delivered`**: 100%
- **`failed_attempt`**: 80% (flagged with warning)
- **`exception`**: 60% (flagged with exception indicator)

---

## 7. Interfaces Architecture

- **CLI (`cmd/`)**: Built on Go standard library with flexible flag normalization allowing flags before or after arguments (`open-parcels track LP... --lang es` and `open-parcels track --lang es LP...` both work).
- **REST Daemon (`pkg/server`)**: Standard HTTP multiplexer with CORS, JSON middleware, and dual v1 (`/api/v1/track`) + ParcelsApp v4 (`/trackings`, `/trackings/:id`) endpoint handlers.
- **MCP Server (`pkg/mcp`)**: Compliant JSON-RPC 2.0 implementation over both `stdio` and streamable HTTP (`/mcp`), exposing tools for AI agents.
