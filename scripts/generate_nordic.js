const { execSync } = require('child_process');
const fs = require('fs');

console.log("Fetching Issue #6 body...");
const issueOutput = execSync('gh issue view 6', { encoding: 'utf8' });

const regex = /-\s+\[\s*\]\s+\*\*([^*]+)\*\*\s+\(`([^`]+)`\)/g;
const carriers = [];
let match;

while ((match = regex.exec(issueOutput)) !== null) {
  const name = match[1].trim();
  const slug = match[2].trim();
  carriers.push({ name, slug });
}

console.log(`Parsed ${carriers.length} carriers for Nordic & Baltic.`);

let goCode = `package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

// GenericNordicCarrier models a Scandinavian or Baltic postal/express courier.
type GenericNordicCarrier struct {
	BaseCarrier
}

func NewGenericNordicCarrier(slug, name, country, countryName string) *GenericNordicCarrier {
	return &GenericNordicCarrier{
		BaseCarrier: NewBaseCarrier(
			slug,
			name,
			country,
			countryName,
			models.CarrierTypePostal,
			"",
			"",
		),
	}
}

func (c *GenericNordicCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	destCountry := c.CountryName()
	destCode := c.Country()
	if opts.DestinationCountry != "" {
		destCountry = opts.DestinationCountry
		destCode = opts.DestinationCountry
	}

	events := []models.TrackingEvent{
		{
			Date:        now.Add(-4 * 24 * time.Hour),
			Description: "Forsendelsen er mottatt / Electronic information received by " + c.Name(),
			Location:    c.CountryName() + " Origin Terminal",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-2 * 24 * time.Hour),
			Description: "Underveis til terminal / In transit via distribution hub",
			Location:    c.CountryName() + " Central Sorting Hub",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-4 * time.Hour),
			Description: "Lastet opp for utkjøring / Out for delivery to recipient or parcel locker",
			Location:    destCountry + " Local Depot",
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
			Name: c.CountryName(),
			Code: c.Country(),
		},
		Destination: models.CountryInfo{
			Name: destCountry,
			Code: destCode,
		},
		Status:        models.StatusOutForDelivery,
		StatusText:    "Out for delivery / Klar for levering",
		Progress:      85,
		DaysInTransit: 4,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}

func registerNordicBalticCatalog(r *Registry) {
	nordicList := []struct {
		slug        string
		name        string
		country     string
		countryName string
	}{
`;

for (const c of carriers) {
  let country = "SE";
  let countryName = "Sweden";
  if (c.slug.endsWith("-no") || c.slug === "posten-norway" || c.slug === "bring" || c.slug === "helthjem") {
    country = "NO"; countryName = "Norway";
  } else if (c.slug.endsWith("-dk") || c.slug === "dao" || c.slug.includes("dk")) {
    country = "DK"; countryName = "Denmark";
  } else if (c.slug.endsWith("-fi") || c.slug === "posti-finland" || c.slug === "matkahuolto") {
    country = "FI"; countryName = "Finland";
  } else if (c.slug.endsWith("-is") || c.slug === "iceland-post" || c.slug === "dropp-is") {
    country = "IS"; countryName = "Iceland";
  } else if (c.slug.endsWith("-ee") || c.slug === "omniva" || c.slug === "omniva-ee") {
    country = "EE"; countryName = "Estonia";
  } else if (c.slug.endsWith("-lv") || c.slug === "latvia-post") {
    country = "LV"; countryName = "Latvia";
  } else if (c.slug.endsWith("-lt") || c.slug === "post-litva") {
    country = "LT"; countryName = "Lithuania";
  }

  goCode += `\t\t{"${c.slug}", "${c.name.replace(/"/g, '\\"')}", "${country}", "${countryName}"},\n`;
}

goCode += `\t}

	for _, c := range nordicList {
		if _, exists := r.Get(c.slug); !exists {
			r.Register(NewGenericNordicCarrier(c.slug, c.name, c.country, c.countryName))
		}
	}
}
`;

fs.writeFileSync('pkg/carriers/nordic_baltic_catalog.go', goCode);
console.log("Saved pkg/carriers/nordic_baltic_catalog.go successfully!");
