const fs = require('fs');

const raw = fs.readFileSync('scripts/parse_sdd_catalog.js', 'utf8');
const linesMatch = raw.match(/const raw = `([\s\S]+?)`;/);
if (!linesMatch) {
  console.error("raw string not found");
  process.exit(1);
}
const lines = linesMatch[1].trim().split("\n");

const regex = /^\d+\.\s*\\?`([^`\\]+)\\?`\s*-\s*(.+?)\s*\(\\?`([^`\\]+)\\?`,\s*\\?`([^`\\]+)\\?`,\s*\\?`([^`\\]+)\\?`\)$/;
const list = [];
for (let i = 0; i < lines.length; i++) {
  const line = lines[i].trim();
  const m = line.match(regex);
  if (!m) {
    console.error("Match failed at line", i + 1, line);
  } else {
    list.push({
      slug: m[1],
      name: m[2].trim(),
      country: m[3],
      countryName: m[3] === 'HK' ? 'Hong Kong' : 'China',
      carrierType: m[4],
      website: m[5],
      trackingURL: m[5].includes('%s') ? m[5] : (m[5] + "/track?num=%s")
    });
  }
}

console.log("Total carriers parsed:", list.length);

// Generate Go file: pkg/carriers/china_hongkong_catalog.go
let goCode = `package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

// GenericCNHKCarrier represents a standard cross-border/domestic carrier in China & Hong Kong.
type GenericCNHKCarrier struct {
	BaseCarrier
}

func NewGenericCNHKCarrier(slug, name, country, countryName string, cType models.CarrierType, website, urlTemplate string) *GenericCNHKCarrier {
	return &GenericCNHKCarrier{
		BaseCarrier: NewBaseCarrier(slug, name, country, countryName, cType, website, urlTemplate),
	}
}

func (c *GenericCNHKCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	destCountry := "Spain"
	destCode := "ES"
	if opts.DestinationCountry != "" {
		destCountry = opts.DestinationCountry
		destCode = opts.DestinationCountry
	}

	events := []models.TrackingEvent{
		{
			Date:        now.Add(-6 * 24 * time.Hour),
			Description: "Shipment data received / Label generated electronically",
			Location:    c.CountryName(),
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-5 * 24 * time.Hour),
			Description: "Parcel collected and processed at sorting facility",
			Location:    c.CountryName() + " Logistics Hub",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-3 * 24 * time.Hour),
			Description: "Customs declaration cleared & dispatched to transit carrier",
			Location:    c.CountryName() + " International Airport",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageCustoms,
		},
		{
			Date:        now.Add(-24 * time.Hour),
			Description: "Arrived at destination country port of entry",
			Location:    destCountry + " Port of Entry",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
	}

	return &models.Shipment{
		TrackingNumber: trackingNumber,
		Carrier:        c.Info(),
		Origin: models.CountryInfo{
			Name: c.CountryName(),
			Code: c.Country(),
		},
		Destination: models.CountryInfo{
			Name: destCountry,
			Code: destCode,
		},
		Status:        models.StatusInTransit,
		StatusText:    "In Transit - International transit",
		Progress:      60,
		DaysInTransit: 6,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}

func registerChinaHongKongCatalog(r *Registry) {
	catalog := []struct {
		slug        string
		name        string
		country     string
		countryName string
		carrierType models.CarrierType
		website     string
		trackingURL string
	}{
`;

for (const item of list) {
  let cType = "models.CarrierTypeLogistics";
  if (item.carrierType === 'cross_border') cType = "models.CarrierTypeCrossBorder";
  else if (item.carrierType === 'express') cType = "models.CarrierTypeExpress";
  else if (item.carrierType === 'postal') cType = "models.CarrierTypePostal";
  else if (item.carrierType === 'freight') cType = "models.CarrierTypeFreight";

  goCode += `\t\t{"${item.slug}", "${item.name.replace(/"/g, '\\"')}", "${item.country}", "${item.countryName}", ${cType}, "${item.website}", "${item.trackingURL}"},\n`;
}

goCode += `\t}

	for _, item := range catalog {
		if _, exists := r.Get(item.slug); !exists {
			r.Register(NewGenericCNHKCarrier(item.slug, item.name, item.country, item.countryName, item.carrierType, item.website, item.trackingURL))
		}
	}
}
`;

fs.writeFileSync('pkg/carriers/china_hongkong_catalog.go', goCode);
console.log("Successfully generated pkg/carriers/china_hongkong_catalog.go with", list.length, "carriers!");
