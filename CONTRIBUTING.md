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
Create `pkg/carriers/<carrier_slug>.go`:

```go
package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type MyCarrier struct {
	BaseCarrier
}

func NewMyCarrier() *MyCarrier {
	return &MyCarrier{
		BaseCarrier: NewBaseCarrier(
			"my-carrier",
			"My Carrier Name",
			"ES",
			"Spain",
			models.CarrierTypeExpress,
			"https://www.mycarrier.com",
			"https://www.mycarrier.com/track/%s",
		),
	}
}

func (c *MyCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	// Attempt real HTTP request or generate simulated lifecycle
    // ...
}
```

### Step 2: Register the Carrier in `pkg/carriers/carrier.go`
In `registerDefaultCarriers(r *Registry)`:
```go
r.Register(NewMyCarrier())
```

### Step 3: Add Detection Rules in `pkg/detector/rules.go`
Add a pattern rule with priority and confidence:
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
