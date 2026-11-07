# Open Parcels (open-parcels)

[![Go Version](https://img.shields.io/badge/go-1.22%2B-blue.svg)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Tests](https://img.shields.io/badge/tests-100%25%20passing-brightgreen.svg)](#-ejecución-de-tests)
[![ParcelsApp API v4](https://img.shields.io/badge/API-ParcelsApp%20v4%20compatible-orange.svg)](docs/API.md)
[![MCP Protocol](https://img.shields.io/badge/MCP-JSON--RPC%202.0-purple.svg)](docs/MCP.md)

> Motor universal de rastreo de envíos y paquetería multitransportista de alto rendimiento, autoalojable y modular en Go, con compatibilidad de API con **ParcelsApp (v4)**, CLI interactivo en terminal y **Servidor MCP** (Model Context Protocol).

---

## 🚀 Características Principales

1. **Detección Automática de Transportistas (Carrier Auto-Detection Engine)**:
   - Identificación inteligente basada en expresiones regulares ultra-optimizadas (< 20 µs), longitudes fijas, prefijos y validadores de suma de comprobación matemática (**UPU S10 Modulo 11**, **UPS 1Z Modulo 10**, **IATA AWB Modulo 7**, **ISO 6346 para contenedores marítimos**).
   - Catálogo de **más de 1.500 operadores** en 17 grupos regionales con pre-filtrado por longitud y charset.
   - Puntuación de confianza (0.0 a 1.0) y refuerzo heurístico mediante pista de país de destino (+0.35).

2. **Adaptadores de Red Reales e Integraciones B2B**:
   - Conexión real con APIs públicas y portales de seguimiento (e.g. **Cainiao Global**, **InPost / Mondial Relay**, **Correos Express**, **Correos España**, **SEUR**, **GLS Spain**).
   - Soporte B2B corporativo con autenticación API Key y flujos OAuth2 Client Credentials con caché en memoria (e.g. **DHL Express / Paket**, **UPS**, **FedEx**).
   - Consulta el estado de cada operador en la **[Matriz de Estado de Transportistas](docs/CARRIERS.md)**.
   - 📢 *¡Llamamiento abierto a la comunidad!* Revisa nuestra **[Guía de Desarrollo de Adaptadores](docs/CARRIER_GUIDE.md)** y ayúdanos a transformar transportistas del catálogo en adaptadores de red reales.

3. **Encadenamiento y Fusión Multitransportista (Multi-Carrier Journey & Event Merging)**:
   - Rastreabilidad del ciclo de vida completo: tramo internacional + transferencia aduanera + última milla local (ej. *Cainiao ➔ Correos Express*, *YunExpress ➔ Correos España*, *SpeedPAK ➔ Local Courier*).
   - Fusión cronológica descendente de hitos de ambos operadores en una única línea de tiempo continua.
   - Deduplicación automática de hitos equivalentes o reintentos de entrega sucesivos.

4. **Máquina de Estados y Normalización Universal**:
   - Estados canónicos normalizados: `pending`, `info_received`, `in_transit`, `out_for_delivery`, `delivered`, `failed_attempt`, `exception`, `expired`.
   - Erradicación total de mocks silenciosos: gestión estricta de errores canónicos (`ErrShipmentNotFound`, `ErrPostalCodeRequired`, `ErrCarrierNotImplemented`, `ErrCarrierRateLimited`).
   - Estimación y cálculo de progreso porcentual del envío (0% a 100%) y soporte bilingüe (`es` y `en`).

5. **Múltiples Interfaces de Consumo**:
   - **CLI en Go**: Salida formateada en terminal con tarjetas ANSI, barras de progreso y línea de tiempo enriquecida, además de modo `--json`.
   - **REST API Daemon**: Servidor HTTP de alto rendimiento con endpoints nativos `/api/v1/track`, compatibilidad 100% con la API v4 de ParcelsApp (`POST /trackings`, `GET /trackings/:request_id`), sondeo asíncrono y despacho automático de Webhooks.
   - **Servidor MCP (Model Context Protocol)**: Compatible con Claude Desktop, Cursor y agentes autónomos mediante transporte `stdio` y endpoint HTTP streamable `/mcp`, con herramientas de diagnóstico (`check_carrier_status`) y metadato `is_live_data`.

---

## 📚 Documentación Técnica

* 🏛️ **[Arquitectura del Sistema](docs/ARCHITECTURE.md)**: Modelos de dominio, diseño de dos niveles, motor de deduplicación y algoritmos de progreso.
* 🌐 **[Referencia de la REST API](docs/API.md)**: Guía completa de endpoints, contratos JSON y compatibilidad ParcelsApp v4.
* 🤖 **[Integración MCP para IA](docs/MCP.md)**: Configuración en Claude Desktop, Cursor, herramientas y schemas JSON-RPC.
* 📦 **[Catálogo y Matriz de Estado de Transportistas](docs/CARRIERS.md)**: Estado de integración (Live API vs Enterprise Auth vs Catalog Only).
* 🛠️ **[Guía de Desarrollo de Adaptadores](docs/CARRIER_GUIDE.md)**: Plantilla obligatoria y arquitectura para crear nuevos adaptadores reales.
* 🤝 **[Guía de Contribución](CONTRIBUTING.md)**: Cómo colaborar con el proyecto, grabar fixtures y correr la suite de tests offline.
* 🔒 **[Política de Seguridad](SECURITY.md)**: Reporte y divulgación responsable de vulnerabilidades.

---

## 🛠️ Instalación y Compilación

Requisitos: **Go 1.22+**

```bash
# Clonar el repositorio
git clone https://github.com/gek0v/open-parcels.git
cd open-parcels

# Compilar el binario
go build -o open-parcels .

# Verificar instalación
./open-parcels version
```

---

## 💻 Uso de la CLI

### 1. Rastrear un envío con detección automática y fusión multitransportista

```bash
# Rastreo automático con salida visual
open-parcels track LP00561234567890 --lang es

# Rastrear con transportista explícito y salida JSON
open-parcels track 9400111206213785678901 --carrier usps --json

# Rastrear envío internacional con pista de país de destino
open-parcels track YT2401234567890123 --country ES
```

### 2. Detección automática de transportistas

```bash
# Identificar operadores candidatos con porcentaje de coincidencia
open-parcels detect 6900123456789012

# Con pista de país de destino para refinar el ranking
open-parcels detect 123456789012 --country ES
```

### 3. Listar el catálogo de transportistas disponibles

```bash
# Ver todos los transportistas (+1.500)
open-parcels carriers

# Filtrar por palabra clave
open-parcels carriers --query express
```

---

## 🌐 Servidor REST API (Daemon)

Inicia el servidor daemon en el puerto deseado:

```bash
open-parcels serve --port 8080
```

### Endpoints Disponibles:

| Método | Endpoint | Descripción |
|---|---|---|
| `GET` | `/health` | Estado del servicio y versión. |
| `GET` | `/api/v1/carriers` | Lista de transportistas soportados (`?query=`). |
| `GET` | `/api/v1/detect?number=...` | Detecta los operadores más probables para un número. |
| `GET` | `/api/v1/track?number=...` | Consulta directa de rastreo (con `&carrier=`, `&lang=`, `&country=`). |
| `POST` | `/api/v1/track` | Consulta directa enviando payload JSON. |
| `POST` | `/trackings` | **Compatibilidad ParcelsApp v4** (admite lotes de envíos). |
| `GET` | `/trackings/:request_id` | **Compatibilidad ParcelsApp v4** (consulta por ID de petición). |
| `POST` | `/mcp` | Endpoint HTTP del servidor Model Context Protocol. |

Para ejemplos detallados de peticiones y respuestas cURL, consulta **[docs/API.md](docs/API.md)**.

---

## 🤖 Integración con Agentes de IA (Servidor MCP)

`open-parcels` incluye soporte nativo para **Model Context Protocol (MCP)** mediante transporte dual (`stdio` y `HTTP`).

### Configuración en Claude Desktop (`claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "open-parcels": {
      "command": "/ruta/absoluta/a/open-parcels",
      "args": ["mcp"]
    }
  }
}
```

### Herramientas MCP expuestas:
- `track_shipment`: Rastreo de envíos con encadenamiento multitransportista.
- `detect_carrier`: Identificación automática del transportista más probable.
- `list_carriers`: Búsqueda en el catálogo de operadores.
- `estimate_delivery`: Cálculo de progreso porcentual y días en tránsito.

Para la guía de configuración en Cursor y clientes HTTP streamable, consulta **[docs/MCP.md](docs/MCP.md)**.

---

## 🏛️ Estructura del Proyecto

```
open-parcels/
├── cmd/                # Comandos CLI (track, detect, carriers, serve, mcp)
├── docs/               # Documentación técnica, API, MCP, arquitectura y catálogos
├── pkg/
│   ├── models/         # Modelos de dominio y contratos JSON (Shipment, TrackingEvent, Status)
│   ├── detector/       # Motor de auto-detección, regex y validadores de checksum
│   ├── carriers/       # Adaptadores de transportistas y 17 catálogos regionales
│   ├── engine/         # Orquestador multitransportista, encadenamiento y deduplicación
│   ├── server/         # Daemon HTTP REST y compatibilidad ParcelsApp v4
│   ├── mcp/            # Servidor Model Context Protocol (stdio y HTTP)
│   └── ui/             # Renderizado visual en terminal (ANSI, barra de progreso, iconos)
├── main.go             # Entrypoint principal
├── CONTRIBUTING.md     # Guía de contribución
├── SECURITY.md         # Política de seguridad
├── LICENSE             # Licencia MIT
└── go.mod
```

---

## 🧪 Ejecución de Tests

```bash
# Correr toda la suite de pruebas unitarias e integración
go test -v ./...
```

---

## 📄 Licencia

[MIT License](LICENSE). Desarrollado como alternativa abierta, rápida y autoalojable a los servicios comerciales de rastreo.
