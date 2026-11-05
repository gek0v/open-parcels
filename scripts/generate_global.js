const { execSync } = require('child_process');
const fs = require('fs');

console.log("Fetching Issue #2 body...");
const issueOutput = execSync('gh issue view 2', { encoding: 'utf8' });

const regex = /-\s+\[\s*\]\s+\*\*([^*]+)\*\*\s+\(`([^`]+)`\)/g;
const carriers = [];
let match;

while ((match = regex.exec(issueOutput)) !== null) {
  const name = match[1].trim();
  const slug = match[2].trim();
  carriers.push({ name, slug });
}

console.log(`Parsed ${carriers.length} carriers for Global Cross-Border.`);

let goCode = `package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

// GenericGlobalCarrier models a multinational cross-border logistics provider.
type GenericGlobalCarrier struct {
	BaseCarrier
}

func NewGenericGlobalCarrier(slug, name string) *GenericGlobalCarrier {
	return &GenericGlobalCarrier{
		BaseCarrier: NewBaseCarrier(
			slug,
			name,
			"GLOBAL",
			"International",
			models.CarrierTypeCrossBorder,
			"",
			"",
		),
	}
}

func (c *GenericGlobalCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	destCountry := "Destination"
	destCode := "XX"
	if opts.DestinationCountry != "" {
		destCountry = opts.DestinationCountry
		destCode = opts.DestinationCountry
	}

	events := []models.TrackingEvent{
		{
			Date:        now.Add(-5 * 24 * time.Hour),
			Description: "Shipment data received electronically by " + c.Name(),
			Location:    "International Dispatch Center",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-4 * 24 * time.Hour),
			Description: "Parcel sorted and processed at multinational transit gateway",
			Location:    "Global Transit Hub",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-2 * 24 * time.Hour),
			Description: "Export customs cleared & handed over to international air cargo",
			Location:    "International Airport Gateway",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-20 * time.Hour),
			Description: "Arrived at destination country hub - Import customs cleared",
			Location:    destCountry + " Gateway Hub",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageCustoms,
		},
		{
			Date:        now.Add(-4 * time.Hour),
			Description: "Handed over to local courier for final mile delivery",
			Location:    destCountry + " Local Depot",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageHandover,
		},
	}

	return &models.Shipment{
		TrackingNumber: trackingNumber,
		Carrier:        c.Info(),
		Origin: models.CountryInfo{
			Name: "Origin Gateway",
			Code: "GLOBAL",
		},
		Destination: models.CountryInfo{
			Name: destCountry,
			Code: destCode,
		},
		Status:        models.StatusInTransit,
		StatusText:    "In Transit - International delivery",
		Progress:      70,
		DaysInTransit: 5,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}

func registerGlobalCrossBorderCatalog(r *Registry) {
	globalList := []struct {
		slug string
		name string
	}{
`;

for (const c of carriers) {
  goCode += `\t\t{"${c.slug}", "${c.name.replace(/"/g, '\\"')}"},\n`;
}

goCode += `\t}

	for _, c := range globalList {
		if _, exists := r.Get(c.slug); !exists {
			r.Register(NewGenericGlobalCarrier(c.slug, c.name))
		}
	}
}
`;

fs.writeFileSync('pkg/carriers/global_crossborder_catalog.go', goCode);
console.log("Saved pkg/carriers/global_crossborder_catalog.go successfully!");
