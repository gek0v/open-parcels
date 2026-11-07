# Contributing to Open Parcels

Thank you for your interest in contributing to **open-parcels**! We welcome contributions to expand carrier coverage, enhance tracking algorithms, improve documentation, and optimize performance.

---

## 1. Development Setup

### Prerequisites
- **Go 1.22 or higher**
- **Git**
- Optional: **Node.js** (for running catalog generation utilities)

```bash
# Fork & clone the repository
git clone https://github.com/gek0v/open-parcels.git
cd open-parcels

# Verify dependencies and tests
go test -v ./...

# Build binary
go build -o open-parcels .
```

---

## 2. Adding a New Carrier

Adding a new carrier involves three simple steps:

### Step 1: Implement the Carrier Adapter in `pkg/carriers/`
Create `pkg/carriers/<carrier_slug>.go` adhering to the enriched `CarrierAdapter` and `LiveTracker` contracts:

```go
package carriers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"open-parcels/pkg/models"
)

var myCarrierHTTPClient = NewCarrierHTTPClient(DefaultRequestTimeout)

// SetMyCarrierHTTPClientForTest overrides the HTTP client for deterministic offline tests.
func SetMyCarrierHTTPClientForTest(client *http.Client) func() {
	orig := myCarrierHTTPClient
	myCarrierHTTPClient = client
	return func() { myCarrierHTTPClient = orig }
}

type MyCarrier struct {
	BaseCarrier
}

func NewMyCarrier() *MyCarrier {
	bc := NewBaseCarrier(
		"my-carrier",
		"My Carrier Name",
		"ES",
		"Spain",
		models.CarrierTypeExpress,
		"https://www.mycarrier.com",
		"https://www.mycarrier.com/track/%s",
	)
	bc.SetCapabilities(CarrierCapabilities{
		SupportsLiveAPI: true,
	})
	return &MyCarrier{BaseCarrier: bc}
}

func (c *MyCarrier) HasLiveTracking() bool {
	return true
}

func (c *MyCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	// 1. Perform network request via the centralized HTTP client
	req, err := NewCarrierRequest(http.MethodGet, fmt.Sprintf("https://api.mycarrier.com/v1/track/%s", trackingNumber))
	if err != nil {
		return nil, err
	}
	resp, err := myCarrierHTTPClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", models.ErrCarrierTemporarilyUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, models.ErrShipmentNotFound
	}

	// 2. Parse response and normalize to *models.Shipment
	// (Under no circumstances return silent simulated mocks when AllowSimulated is false)
	return &models.Shipment{
		TrackingNumber: trackingNumber,
		Carrier:        c.Info(),
		Status:         models.StatusInTransit,
		UpdatedAt:      time.Now(),
	}, nil
}
```

### Step 2: Register the Carrier in `pkg/carriers/carrier.go`
In `registerDefaultCarriers(r *Registry)`:
```go
r.Register(NewMyCarrier())
```

### Step 3: Add Detection Rules in `pkg/detector/rules.go`
Add a pattern rule with priority, confidence base, and regex:
```go
{
    CarrierSlug:    "my-carrier",
    CarrierName:    "My Carrier Name",
    Country:        "ES",
    CountryName:    "Spain",
    CarrierType:    models.CarrierTypeExpress,
    Priority:       85,
    ConfidenceBase: 0.94,
    Regex:          regexp.MustCompile(`^MY\d{10,12}$`),
    Description:    "My Carrier parcel barcode",
},
```

---

## 3. Testing Methodology: Golden Fixtures & Integration Tags

To prevent flaky tests and keep CI execution under 5 seconds without internet dependencies:

### 1. Deterministic Unit Tests with Golden Fixtures
1. Save sanitized actual JSON/HTML payloads from the carrier under `testdata/fixtures/<carrier_slug>_<status>.json`.
2. In your unit test `pkg/carriers/<carrier_slug>_test.go`, load the fixture using `LoadFixture` and inject a mock `http.RoundTripper` using `NewFixtureClient`:

```go
func TestMyCarrierDeliveredFixture(t *testing.T) {
	fixtureData, err := LoadFixture("mycarrier_delivered.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	teardown := SetMyCarrierHTTPClientForTest(NewFixtureClient(http.StatusOK, fixtureData, "application/json"))
	defer teardown()

	c := NewMyCarrier()
	shipment, err := c.Track(context.Background(), "MY1234567890", models.TrackOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if shipment.Status != models.StatusDelivered {
		t.Errorf("expected StatusDelivered, got %s", shipment.Status)
	}
}
```

### 2. Live Integration Tests
Any test performing real external network calls must be placed in a separate file `pkg/carriers/<carrier_slug>_live_test.go` with the build tag:

```go
//go:build integration

package carriers
// ...
```

Live tests will run exclusively during scheduled weekly CI or via `go test -tags=integration ./...`.

---

## 4. Code Standards and Anti-Mock Governance

1. **No Silent Mocks**: Never return simulated shipments with `time.Now()` unless the caller explicitly passed `AllowSimulated: true`.
2. **Canonical Errors**:
   - `models.ErrShipmentNotFound` for missing tracking codes.
   - `models.ErrPostalCodeRequired` for missing postal codes.
   - `models.ErrCarrierRateLimited` for HTTP 429.
   - `models.ErrCarrierTemporarilyUnavailable` for WAF/captcha challenges or service downtime.
3. **No Hardcoded Guides**: Never hardcode static delivery dates or tracking guides in production drivers.
4. **Race Free**: Ensure all code passes `go test -v -race ./...` without data races.

### Step 4: Add Unit Tests
Add test assertions to:
- `pkg/detector/detector_test.go`
- `pkg/carriers/carriers_test.go`

Run tests:
```bash
go test -v ./...
```

---

## 3. Pull Request Workflow

1. Create a feature branch with a clear name:
   ```bash
   git checkout -b feat/add-my-carrier
   ```
2. Commit changes using conventional commit messages:
   - `feat(carriers): add My Carrier tracking adapter`
   - `fix(detector): correct regex boundary for Carrier X`
   - `docs: update API reference for new endpoint`
3. Push to your fork and submit a Pull Request to `main`.
4. Ensure all tests and static analysis pass cleanly.
