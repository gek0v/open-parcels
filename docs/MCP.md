# Model Context Protocol (MCP) Integration Guide

`open-parcels` natively implements the **Model Context Protocol (MCP)** specification (JSON-RPC 2.0, version `2024-11-05`). This allows autonomous coding and AI agents (such as Claude Desktop, Cursor, and Pi-Bolt) to automatically track shipments, detect carriers, and estimate delivery timelines.

---

## 1. Supported Transports

`open-parcels` provides two transport mechanisms:
1. **`stdio` Transport**: Ideal for local execution with desktop AI clients (Claude Desktop, local Agent runners).
2. **Streamable `HTTP` Transport**: Hosted on `POST /mcp` when running `open-parcels serve --mcp`.

---

## 2. Setting Up Local Clients

### 2.1 Claude Desktop

Add `open-parcels` to your `claude_desktop_config.json`:

* **macOS / Linux**: `~/Library/Application Support/Claude/claude_desktop_config.json`
* **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "open-parcels": {
      "command": "open-parcels",
      "args": ["mcp"]
    }
  }
}
```
*(Ensure `open-parcels` is on your PATH or provide the absolute path to the binary).*

---

### 2.2 Cursor IDE

In Cursor Settings ➔ **Features** ➔ **MCP Servers**, add a new server:
- **Type**: `command`
- **Command**: `/path/to/open-parcels mcp`

Or configure `.cursor/mcp.json`:
```json
{
  "mcpServers": {
    "open-parcels": {
      "command": "open-parcels",
      "args": ["mcp"]
    }
  }
}
```

---

### 2.3 Remote HTTP Transport (`/mcp`)

Start the daemon with the MCP endpoint enabled:
```bash
open-parcels serve --port 8080 --mcp
```
Then point any streamable HTTP JSON-RPC client to `http://localhost:8080/mcp`.

**Authentication & Rate Limiting:**
If authentication is configured via `OPEN_PARCELS_API_KEY`, `OPEN_PARCELS_API_KEYS`, or `--api-key`, HTTP requests to `/mcp` must include either:
- `Authorization: Bearer <token>`
- `X-API-Key: <token>`

Unauthenticated or invalid requests return `401 Unauthorized`. Configured rate limits (`OPEN_PARCELS_RATE_LIMIT` / `--rate-limit`) also protect `/mcp` and return `429 Too Many Requests` with a `Retry-After` header when exceeded.

---

## 3. Available Tools & Schemas

### 3.1 `track_shipment`
Tracks a package with automatic carrier identification, international-to-local handover chaining, and chronological event merging.

**Parameters:**
- `tracking_number` (string, required): Parcel barcode or tracking ID.
- `carrier_hint` (string, optional): Specific carrier slug (e.g. `cainiao`, `correos-express`, `usps`, `ups`, `dhl-express`).
- `destination_country` (string, optional): ISO-2 destination country code (e.g. `ES`, `US`, `DE`).
- `language` (string, optional): Language for status text and event descriptions (`en` or `es`, default `en`).

**Example MCP Call:**
```json
{
  "name": "track_shipment",
  "arguments": {
    "tracking_number": "LP00561234567890",
    "destination_country": "ES",
    "language": "es"
  }
}
```

---

### 3.2 `detect_carrier`
Detects possible logistics carriers for a given tracking number with confidence scoring and regex validation.

**Parameters:**
- `tracking_number` (string, required): Tracking number to evaluate.
- `destination_country` (string, optional): Destination country hint to boost candidate scoring.

**Example MCP Call:**
```json
{
  "name": "detect_carrier",
  "arguments": {
    "tracking_number": "6900123456789012",
    "destination_country": "ES"
  }
}
```

---

### 3.3 `list_carriers`
Lists supported carriers, slugs, countries, and service categories.

**Parameters:**
- `query` (string, optional): Filter by carrier name, slug, or country code.

**Example MCP Call:**
```json
{
  "name": "list_carriers",
  "arguments": {
    "query": "express"
  }
}
```

---

### 3.4 `estimate_delivery`
Calculates delivery progress percentage, transit duration, and current status summary.

**Parameters:**
- `tracking_number` (string, required): Tracking number to estimate.

---

## 4. Example AI Prompts with MCP

Once configured, you can prompt your AI assistant naturally:

- *"¿Dónde está mi paquete con número de seguimiento LP00561234567890 que viene de AliExpress a España?"*
- *"Detect which carrier handles tracking number 9400111206213785678901 and track its status."*
- *"Check if AWB 157-52471764 has departed on its flight."*
- *"List all supported carriers in Spain and Germany."*
