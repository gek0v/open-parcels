# Guía de Desarrollo de Adaptadores de Transportistas (`CarrierAdapter`)

Esta guía define el estándar oficial de arquitectura para implementar adaptadores de red reales en **Open-Parcels**, asegurando trazabilidad, gestión de credenciales, manejo canónico de errores y pruebas reproducibles sin conexión a Internet.

---

## 1. La Interfaz `CarrierAdapter`

Todo nuevo transportista debe implementar la interfaz `CarrierAdapter` (`pkg/carriers/carrier.go`):

```go
type CarrierAdapter interface {
    Carrier
    ID() string
    Capabilities() CarrierCapabilities
}
```

### Capacidades (`CarrierCapabilities`)

```go
type CarrierCapabilities struct {
    RequiresPostalCode      bool `json:"requires_postal_code"`
    RequiresAuth            bool `json:"requires_auth"`
    SupportsProofOfDelivery bool `json:"supports_proof_of_delivery"`
    SupportsLiveAPI         bool `json:"supports_live_api"`
    SupportsWebScraping     bool `json:"supports_web_scraping"`
}
```

Adicionalmente, para ser reconocido como un adaptador activo con conexión de red por el motor (`pkg/engine`), el adaptador debe implementar la interfaz `LiveTracker`:

```go
func (c *MiCarrier) HasLiveTracking() bool {
    return true
}
```

---

## 2. Plantilla Estándar de Implementación

Guarda tu nuevo transportista en `pkg/carriers/<slug>.go`:

```go
package carriers

import (
    "context"
    "fmt"
    "net/http"
    "time"

    "open-parcels/pkg/models"
)

type ExampleCarrier struct {
    BaseCarrier
    client *http.Client
}

func NewExampleCarrier() *ExampleCarrier {
    bc := NewBaseCarrier(
        "example-express",
        "Example Express",
        "ES",
        "Spain",
        models.CarrierTypeExpress,
        "https://www.example.com",
        "https://www.example.com/track/%s",
    )
    bc.SetCapabilities(CarrierCapabilities{
        RequiresPostalCode:  true,
        RequiresAuth:        false,
        SupportsLiveAPI:     true,
    })

    return &ExampleCarrier{
        BaseCarrier: bc,
        client:      NewCarrierHTTPClient(DefaultRequestTimeout),
    }
}

func (c *ExampleCarrier) HasLiveTracking() bool {
    return true
}

func (c *ExampleCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
    // 1. Validación de requerimientos obligatorios
    if opts.PostalCode == "" {
        return nil, models.ErrPostalCodeRequired
    }

    // 2. Consulta HTTP utilizando el transporte centralizado
    targetURL := fmt.Sprintf("https://api.example.com/v1/shipments/%s", trackingNumber)
    req, err := NewCarrierRequest("GET", targetURL)
    if err != nil {
        return nil, err
    }
    req = req.WithContext(ctx)

    resp, err := c.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("%w: %v", models.ErrCarrierTemporarilyUnavailable, err)
    }
    defer resp.Body.Close()

    // 3. Mapeo estricto de códigos HTTP a errores canónicos
    switch resp.StatusCode {
    case http.StatusNotFound:
        return nil, models.ErrShipmentNotFound
    case http.StatusTooManyRequests:
        return nil, models.ErrCarrierRateLimited
    case http.StatusOK:
        // Parsear respuesta JSON/HTML
    default:
        return nil, fmt.Errorf("%w: HTTP status %d", models.ErrCarrierTemporarilyUnavailable, resp.StatusCode)
    }

    // 4. Retornar modelo canónico models.Shipment
    return &models.Shipment{
        TrackingNumber: trackingNumber,
        Carrier:        c.Info(),
        Status:         models.StatusInTransit,
        UpdatedAt:      time.Now(),
    }, nil
}
```

---

## 3. Transporte HTTP Centralizado (`transport.go`)

Todos los adaptadores deben usar las utilidades provistas en `pkg/carriers/transport.go` en lugar de instanciar `http.DefaultClient`:

1. **`NewCarrierHTTPClient(timeout)`**: Cliente pre-configurado con timeouts de conexión, soporte de proxies HTTP/HTTPS/SOCKS5 (`OPEN_PARCELS_PROXY_URL`) y reutilización de conexiones TLS.
2. **`NewCarrierRequest(method, url)`**: Inicializa peticiones con cabeceras `User-Agent`, `Accept` y `Accept-Language` estandarizadas.
3. **`GetCarrierCredentials(slug)`**: Obtiene claves y secretos corporativos desde variables de entorno.

### Inyección de Credenciales

Para transportistas que requieran claves de acceso o credenciales OAuth2:
- Variable con prefijo Open-Parcels: `OPEN_PARCELS_<CARRIER>_KEY`, `OPEN_PARCELS_<CARRIER>_SECRET`, `OPEN_PARCELS_<CARRIER>_CLIENT_ID`.
- Fallback estándar: `<CARRIER>_API_KEY`, `<CARRIER>_CLIENT_ID`.

---

## 4. Gestión Canónica de Errores

Bajo ninguna circunstancia se deben devolver mocks o eventos simulados cuando la llamada de red falle o el envío no exista. Usa los errores definidos en `pkg/models/errors.go`:

| Situación | Error Canónico | Código HTTP en API |
|---|---|---|
| Paquete inexistente / no registrado | `models.ErrShipmentNotFound` | `404 Not Found` |
| Falta código postal obligatorio | `models.ErrPostalCodeRequired` | `422 Unprocessable Entity` |
| Bloqueo temporal / Rate limit | `models.ErrCarrierRateLimited` | `429 Too Many Requests` |
| Caída del transportista / WAF / Captcha | `models.ErrCarrierTemporarilyUnavailable` | `502 Bad Gateway` |
| Transportista sin adaptador implementado | `models.ErrCarrierNotImplemented` | `501 Not Implemented` |

---

## 5. Pruebas Unitarias y Fixtures Deterministas

1. **Tests unitarios offline:** Deben utilizar `http.RoundTripper` simulado en memoria cargando fixtures JSON desde `testdata/fixtures/` sin realizar peticiones de red reales.
2. **Tests de integración en vivo:** Aquellos tests que requieran conexión de red contra APIs de transportistas deben incluir en la primera línea del archivo el build tag:
   ```go
   //go:build integration
   ```
   Para que no se ejecuten durante el ciclo regular de `go test ./...`.
