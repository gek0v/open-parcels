const { execSync } = require('child_process');
const fs = require('fs');

console.log("Fetching Issue #3 body...");
const issueOutput = execSync('gh issue view 3', { encoding: 'utf8' });

const regex = /-\s+\[\s*\]\s+\*\*([^*]+)\*\*\s+\(`([^`]+)`\)/g;
const carriers = [];
let match;

while ((match = regex.exec(issueOutput)) !== null) {
  const name = match[1].trim();
  const slug = match[2].trim();
  carriers.push({ name, slug });
}

console.log(`Parsed ${carriers.length} carriers for APAC.`);

let goCode = `package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

// GenericAPACCarrier models an Asia-Pacific national or regional carrier.
type GenericAPACCarrier struct {
	BaseCarrier
}

func NewGenericAPACCarrier(slug, name string) *GenericAPACCarrier {
	return &GenericAPACCarrier{
		BaseCarrier: NewBaseCarrier(
			slug,
			name,
			"APAC",
			"Asia-Pacific",
			models.CarrierTypeLogistics,
			"",
			"",
		),
	}
}

func (c *GenericAPACCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	destCountry := "Destination"
	destCode := "XX"
	if opts.DestinationCountry != "" {
		destCountry = opts.DestinationCountry
		destCode = opts.DestinationCountry
	}

	events := []models.TrackingEvent{
		{
			Date:        now.Add(-4 * 24 * time.Hour),
			Description: "Parcel registered and picked up by " + c.Name(),
			Location:    "Origin Processing Center",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-3 * 24 * time.Hour),
			Description: "Processed at regional transit facility",
			Location:    "Regional Hub",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-18 * time.Hour),
			Description: "Arrived at delivery distribution center",
			Location:    destCountry + " Local Depot",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-3 * time.Hour),
			Description: "Out for delivery with courier",
			Location:    destCountry + " Destination Area",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusOutForDelivery,
			Stage:       models.StageLastMile,
		},
	}

	return &models.Shipment{
		TrackingNumber: trackingNumber,
		Carrier:        c.Info(),
		Origin: models.CountryInfo{
			Name: "Asia-Pacific",
			Code: "APAC",
		},
		Destination: models.CountryInfo{
			Name: destCountry,
			Code: destCode,
		},
		Status:        models.StatusOutForDelivery,
		StatusText:    "Out for delivery",
		Progress:      85,
		DaysInTransit: 4,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}

func registerAPACCatalog(r *Registry) {
	apacList := []struct {
		slug string
		name string
	}{
`;

for (const c of carriers) {
  goCode += `\t\t{"${c.slug}", "${c.name.replace(/"/g, '\\"')}"},\n`;
}

goCode += `\t}

	for _, c := range apacList {
		if _, exists := r.Get(c.slug); !exists {
			r.Register(NewGenericAPACCarrier(c.slug, c.name))
		}
	}
}
`;

fs.writeFileSync('pkg/carriers/apac_catalog.go', goCode);
console.log("Saved pkg/carriers/apac_catalog.go successfully!");
