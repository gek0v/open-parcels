# Open Parcels REST API Documentation

The `open-parcels` daemon exposes a lightweight, high-performance HTTP REST API supporting both native endpoints (`/api/v1/*`) and drop-in compatibility with **ParcelsApp API v4** (`/trackings`).

Base URL: `http://localhost:8080` (or configured port via `--port`).

---

## 1. Endpoints Overview

| Method | Path | Description |
| :--- | :--- | :--- |
| `GET` | `/health` | Health check, service status, and runtime version. |
| `GET` | `/api/v1/carriers` | List all supported carriers (with optional `?query=`). |
| `GET` | `/tracking-carriers` | Alias for `/api/v1/carriers` (ParcelsApp compatibility). |
| `GET` | `/api/v1/detect?number=...` | Detect candidate carriers for a given tracking number. |
| `GET` | `/api/v1/track?number=...` | Direct tracking query via GET parameters. |
| `POST` | `/api/v1/track` | Track a shipment with JSON body. |
| `POST` | `/trackings` | **ParcelsApp v4 API** batch/single tracking request. |
| `GET` | `/trackings/{request_id}` | **ParcelsApp v4 API** lookup by request ID. |
| `POST` | `/mcp` | Streamable HTTP endpoint for Model Context Protocol. |

---

## 2. Native API Endpoints

### 2.1 Health Check (`GET /health`)
Verifies service availability.

**Response (200 OK):**
```json
{
  "service": "open-parcels",
  "status": "ok",
  "time": "2026-11-05T21:00:00Z",
  "version": "1.0.0"
}
```

---

### 2.2 List Carriers (`GET /api/v1/carriers`)

**Query Parameters:**
- `query` (optional): Filter carriers by name, slug, or country code.

**Example Request:**
```bash
curl http://localhost:8080/api/v1/carriers?query=cainiao
```

**Response (200 OK):**
```json
{
  "total": 1,
  "carriers": [
    {
      "slug": "cainiao",
      "name": "Cainiao Global",
      "country": "CN",
      "country_name": "China",
      "type": "cross_border",
      "website": "https://global.cainiao.com",
      "tracking_url": "https://global.cainiao.com/detail.htm?mailNoList=%s"
    }
  ]
}
```

---

### 2.3 Carrier Auto-Detection (`GET /api/v1/detect`)

**Query Parameters:**
- `number` (required): Tracking number or barcode.
- `country` (optional): Destination country ISO-2 code (e.g. `ES`, `US`).

**Example Request:**
```bash
curl "http://localhost:8080/api/v1/detect?number=LP00561234567890&country=ES"
```

**Response (200 OK):**
```json
{
  "tracking_number": "LP00561234567890",
  "candidates": [
    {
      "carrier": {
        "slug": "cainiao",
        "name": "Cainiao Global",
        "country": "CN",
        "country_name": "China",
        "type": "cross_border"
      },
      "confidence": 0.95,
      "matched_pattern": "Cainiao / AliExpress standard tracking (LP, CAINIAO, CNESP, UN...CN)",
      "reason": "Matched pattern: Cainiao / AliExpress standard tracking (LP, CAINIAO, CNESP, UN...CN)"
    }
  ]
}
```

---

### 2.4 Track Shipment (`POST /api/v1/track`)

**Request Headers:**
`Content-Type: application/json`

**Request Body:**
```json
{
  "tracking_number": "LP00561234567890",
  "carrier_hint": "cainiao",
  "destination_country": "ES",
  "language": "es"
}
```

**Response (200 OK):**
```json
{
  "tracking_number": "LP00561234567890",
  "carrier": {
    "slug": "cainiao",
    "name": "Cainiao Global",
    "country": "CN",
    "country_name": "China",
    "type": "cross_border",
    "website": "https://global.cainiao.com",
    "tracking_url": "https://global.cainiao.com/detail.htm?mailNoList=%s"
  },
  "additional_carriers": [
    {
      "slug": "correos-express",
      "name": "Correos Express",
      "country": "ES",
      "country_name": "Spain",
      "type": "express",
      "website": "https://www.correosexpress.com",
      "tracking_url": "https://www.correosexpress.com/web/correosexpress/consultanos?shippingNumber=%s"
    }
  ],
  "last_mile_tracking_number": "6900561234567890",
  "status": "out_for_delivery",
  "status_text": "En reparto",
  "progress": 85,
  "origin": {
    "name": "China",
    "code": "CN"
  },
  "destination": {
    "name": "Spain",
    "code": "ES"
  },
  "events": [
    {
      "date": "2026-11-05T13:00:00Z",
      "status": "out_for_delivery",
      "stage": "last_mile",
      "description": "En reparto - Envío asignado al repartidor para entrega",
      "location": "Barcelona, Spain",
      "carrier_slug": "correos-express",
      "carrier_name": "Correos Express",
      "carrier_index": 1
    },
    {
      "date": "2026-11-04T19:00:00Z",
      "status": "in_transit",
      "stage": "handover",
      "description": "Handed over to local delivery partner for last mile delivery",
      "location": "Coslada Hub, Madrid, Spain",
      "carrier_slug": "cainiao",
      "carrier_name": "Cainiao Global",
      "carrier_index": 0
    }
  ],
  "days_in_transit": 6,
  "from_cache": false,
  "updated_at": "2026-11-05T13:05:00Z"
}
```

---

## 3. ParcelsApp v4 Compatibility Endpoints

`open-parcels` implements drop-in compatibility with the official ParcelsApp API v4 specification.

### 3.1 Create Tracking Request (`POST /trackings`)

**Request Body:**
```json
{
  "language": "es",
  "shipments": [
    {
      "tracking_number": "LP00561234567890",
      "destination_country": "ES"
    }
  ]
}
```

**Response (200 OK):**
```json
{
  "request_id": "req_1793822684451000",
  "status": "completed",
  "done": true,
  "expires_at": "2026-11-05T21:35:00Z",
  "results": [
    {
      "tracking_number": "LP00561234567890",
      "state": "completed",
      "from_cache": false,
      "shipment": {
        "trackingId": "LP00561234567890",
        "status": "out_for_delivery",
        "origin": "China",
        "destination": "Spain",
        "originCode": "CN",
        "destinationCode": "ES",
        "states": [
          {
            "state": "En reparto - Envío asignado al repartidor para entrega",
            "location": "Barcelona, Spain",
            "date": "2026-11-05T13:00:00Z",
            "carrier": 1
          }
        ],
        "services": [
          { "slug": "cainiao", "name": "Cainiao Global" },
          { "slug": "correos-express", "name": "Correos Express" }
        ],
        "detectedCarrier": {
          "slug": "cainiao",
          "name": "Cainiao Global"
        },
        "detected": [0, 1],
        "daysInTransit": 6,
        "attributes": [
          { "l": "origin", "val": "China", "code": "CN" },
          { "l": "destination", "val": "Spain", "code": "ES" }
        ]
      }
    }
  ]
}
```

### 3.2 Read Tracking Result (`GET /trackings/{request_id}`)
Retrieves the completed tracking result stored during the active session.

**Example Request:**
```bash
curl http://localhost:8080/trackings/req_1793822684451000
```

---

## 4. Error Responses

Errors conform to standardized JSON envelopes:

```json
{
  "error": {
    "code": "TRACKING_FAILED",
    "message": "could not automatically detect carrier for tracking number: INVALID999"
  }
}
```

Common error codes:
- `MISSING_NUMBER` (400): Tracking number parameter was omitted.
- `INVALID_REQUEST` (400): JSON payload was malformed.
- `NO_SHIPMENTS` (400): All submitted shipment items were invalid.
- `TRACKING_FAILED` (404): Carrier could not be detected or lookup failed.
- `REQUEST_NOT_FOUND` (404): Tracking request record expired or does not exist.
