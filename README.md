# Open Parcels (open-parcels)

> Motor universal de rastreo de envíos y paquetería multitransportista de alto rendimiento, autoalojable y modular en Go, con compatibilidad de API con ParcelsApp (v4), CLI interactivo en terminal y Servidor MCP (Model Context Protocol).

---

## 🚀 Características Principales

1. **Detección Automática de Transportistas (Carrier Auto-Detection Engine)**:
   - Identificación inteligente basada en expresiones regulares, longitudes fijas, prefijos y validadores de suma de comprobación (Luhn, Modulo 7, Modulo 10, UPU S10, ISO 6346).
   - Soporte para más de 25 operadores globales y locales: **Cainiao / AliExpress**, **Correos España**, **Correos Express**, **GLS**, **CTT Express**, **SEUR**, **Paack**, **MRW**, **InPost / Mondial Relay**, **DHL (Express & Parcel)**, **UPS**, **FedEx**, **USPS**, **YunExpress**, **4PX**, **Royal Mail**, **Colissimo**, transportistas de Portugal (**Delnext**, **DPD Portugal**, **Portugal Post - CTT**, **WePickUp**), **IATA Air Cargo (AWB)** y **Contenedores Marítimos (ISO 6346)**.
   - Puntuación de confianza (0.0 a 1.0) y refuerzo heurístico mediante pista de país de destino.

2. **Encadenamiento y Fusión Multitransportista (Multi-Carrier Journey & Event Merging)**:
   - Rastreabilidad del ciclo de vida completo: tramo internacional (origen Asia / Europa / América) + transferencia aduanera + última milla local (ej. *Cainiao ➔ Correos Express*, *YunExpress ➔ Correos España*).
   - Fusión cronológica descendente de hitos de ambos operadores en una única línea de tiempo continua.
   - Deduplicación automática de hitos equivalentes dentro de ventanas temporales concurrentes.

3. **Máquina de Estados y Normalización Universal**:
   - Estados canónicos: `pending`, `info_received`, `in_transit`, `out_for_delivery`, `delivered`, `failed_attempt`, `exception`, `expired`.
   - Estimación y cálculo de progreso porcentual del envío (0% a 100%).
   - Soporte bilingüe en descripciones de estado (`es` y `en`).

4. **Múltiples Interfaces de Consumo**:
   - **CLI en Go**: Salida formateada en terminal con tarjetas ANSI, barras de progreso y línea de tiempo enriquecida, además de modo `--json`.
   - **REST API Daemon**: Servidor HTTP de alto rendimiento con endpoints nativos `/api/v1/track` y compatibilidad 100% con la API v4 de ParcelsApp (`POST /trackings`, `GET /trackings/:request_id`).
   - **Servidor MCP (Model Context Protocol)**: Compatible con Claude Desktop, Cursor y agentes autónomos mediante transporte `stdio` y endpoint HTTP streamable `/mcp`.

---

## 🛠️ Instalación y Compilación

Requisitos: **Go 1.22+**

```bash
# Clonar el repositorio
git clone https://github.com/open-parcels/open-parcels.git
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
# Ver todos los transportistas
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

#### Ejemplo de llamada a `/api/v1/track`:

```bash
curl -X POST http://localhost:8080/api/v1/track \
  -H "Content-Type: application/json" \
  -d '{
    "tracking_number": "LP00561234567890",
    "language": "es",
    "destination_country": "ES"
  }'
```

#### Ejemplo de compatibilidad con ParcelsApp v4 (`POST /trackings`):

```bash
curl -X POST http://localhost:8080/trackings \
  -H "Content-Type: application/json" \
  -d '{
    "language": "es",
    "shipments": [
      {
        "tracking_number": "LP00561234567890",
        "destination_country": "ES"
      }
    ]
  }'
```

---

## 🤖 Integración con Agentes de IA (Servidor MCP)

`open-parcels` incluye soporte nativo para **Model Context Protocol (MCP)**, permitiendo a asistentes y agentes de IA (Claude Desktop, Cursor, Pi, etc.) rastrear envíos de manera autónoma.

### 1. Uso mediante stdio

Ejecuta el servidor en modo stdio:
```bash
open-parcels mcp
```

#### Configuración en Claude Desktop (`claude_desktop_config.json`):

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

### 2. Herramientas MCP expuestas:

- `track_shipment`:
  - `tracking_number` (requerido): Número o código de seguimiento.
  - `carrier_hint` (opcional): Slug del transportista.
  - `destination_country` (opcional): Código ISO-2 de destino.
  - `language` (opcional): Idioma (`es` o `en`).
- `detect_carrier`:
  - `tracking_number` (requerido): Identifica el transportista más probable.
- `list_carriers`:
  - `query` (opcional): Consulta el catálogo de transportistas.
- `estimate_delivery`:
  - `tracking_number` (requerido): Calcula estado, progreso porcentual y días transcurridos.

---

## 🏛️ Estructura del Proyecto

```
open-parcels/
├── cmd/                # Comandos CLI (track, detect, carriers, serve, mcp)
├── pkg/
│   ├── models/         # Modelos de dominio y contratos JSON (Shipment, TrackingEvent, Status)
│   ├── detector/       # Motor de auto-detección y reglas de patrones
│   ├── carriers/       # Adaptadores de transportistas y registro central
│   ├── engine/         # Orquestador multitransportista, encadenamiento, deduplicación y máquina de estados
│   ├── server/         # Daemon HTTP REST y compatibilidad ParcelsApp v4
│   ├── mcp/            # Servidor Model Context Protocol (stdio y HTTP)
│   └── ui/             # Renderizado visual en terminal (ANSI, barra de progreso, iconos)
├── main.go             # Punto de entrada
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

MIT License. Desarrollado como alternativa abierta, rápida y autoalojable a los servicios comerciales de rastreo.
