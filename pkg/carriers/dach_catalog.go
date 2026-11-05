package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

// GenericDACHCarrier models a German, Austrian, or Swiss delivery courier.
type GenericDACHCarrier struct {
	BaseCarrier
}

func NewGenericDACHCarrier(slug, name, country, countryName string) *GenericDACHCarrier {
	return &GenericDACHCarrier{
		BaseCarrier: NewBaseCarrier(
			slug,
			name,
			country,
			countryName,
			models.CarrierTypeLogistics,
			"",
			"",
		),
	}
}

func (c *GenericDACHCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	destCountry := c.CountryName()
	destCode := c.Country()
	if opts.DestinationCountry != "" {
		destCountry = opts.DestinationCountry
		destCode = opts.DestinationCountry
	}

	events := []models.TrackingEvent{
		{
			Date:        now.Add(-3 * 24 * time.Hour),
			Description: "Die Sendungsdaten wurden an " + c.Name() + " elektronisch übermittelt",
			Location:    c.CountryName() + " Paketzentrum",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-2 * 24 * time.Hour),
			Description: "Die Sendung wurde im Ziel-Paketzentrum bearbeitet",
			Location:    c.CountryName() + " Verteilzentrum",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-3 * time.Hour),
			Description: "Die Sendung befindet sich in der Zustellung",
			Location:    destCountry + " Zustellbasis",
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
		StatusText:    "In Zustellung / Out for delivery",
		Progress:      85,
		DaysInTransit: 3,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}

func registerDACHCatalog(r *Registry) {
	dachList := []struct {
		slug        string
		name        string
		country     string
		countryName string
	}{
		{"swiship-de", "Amazon Swiship DE", "DE", "Germany"},
		{"amm-spedition", "AMM Spedition", "DE", "Germany"},
		{"austria-mail", "Austria Post", "AT", "Austria"},
		{"cepra-de", "Cepra", "DE", "Germany"},
		{"dachser", "Dachser", "DE", "Germany"},
		{"day-one", "Day One", "DE", "Germany"},
		{"dbschenker", "DB Schenker", "DE", "Germany"},
		{"dbschenker-pkg", "DB Schenker Package", "DE", "Germany"},
		{"deutsche-post", "Deutsche Post - DHL Paket", "DE", "Germany"},
		{"dpd-germany", "DPD", "DE", "Germany"},
		{"dpd-austria", "DPD Austria", "AT", "Austria"},
		{"emons-de", "Emons", "DE", "Germany"},
		{"gebruder-weiss", "Gebrüder Weiss", "DE", "Germany"},
		{"gel-express", "GEL Express Logistik", "DE", "Germany"},
		{"gls-pakete-de", "GLSPakete", "DE", "Germany"},
		{"hellmann-air-sea", "Hellmann Air/Sea", "DE", "Germany"},
		{"hellmann-road", "Hellmann Road", "DE", "Germany"},
		{"hellmann", "Hellmann Worldwide Logistics", "DE", "Germany"},
		{"ids-logistik-de", "IDS Logistik", "DE", "Germany"},
		{"intex-de", "INTEX Paketdienst", "DE", "Germany"},
		{"mbe-de", "Mail Boxes Etc. Germany", "DE", "Germany"},
		{"myhermes-de", "myHermes DE", "DE", "Germany"},
		{"noerpel", "Noerpel", "DE", "Germany"},
		{"rhenus-de", "Rhenus Germany", "DE", "Germany"},
		{"swiss-post", "Swiss Post", "CH", "Switzerland"},
		{"trans-o-flex", "Trans-o-Flex", "DE", "Germany"},
	}

	for _, c := range dachList {
		if _, exists := r.Get(c.slug); !exists {
			r.Register(NewGenericDACHCarrier(c.slug, c.name, c.country, c.countryName))
		}
	}
}
