package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

// GenericBeneluxCarrier models a Dutch, Belgian, or Luxembourgish delivery service.
type GenericBeneluxCarrier struct {
	BaseCarrier
}

func NewGenericBeneluxCarrier(slug, name, country, countryName string) *GenericBeneluxCarrier {
	return &GenericBeneluxCarrier{
		BaseCarrier: NewBaseCarrier(
			slug,
			name,
			country,
			countryName,
			models.CarrierTypeExpress,
			"",
			"",
		),
	}
}

func (c *GenericBeneluxCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
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
			Description: "Zending is aangemeld / Shipment registered by " + c.Name(),
			Location:    c.CountryName() + " Origin Sorting Center",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-2 * 24 * time.Hour),
			Description: "Zending gesorteerd in sorteercentrum / In transit to distribution hub",
			Location:    c.CountryName() + " Central Distribution Depot",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-3 * time.Hour),
			Description: "Chauffeur is onderweg / Out for delivery with courier",
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
		StatusText:    "Onderweg / Out for delivery",
		Progress:      85,
		DaysInTransit: 3,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}

func registerBeneluxCatalog(r *Registry) {
	beneluxList := []struct {
		slug        string
		name        string
		country     string
		countryName string
	}{
		{"bpost-local", "Bpost Belgium", "BE", "Belgium"},
		{"bringly-nl", "Bringly.nl", "NL", "Netherlands"},
		{"budbee", "Budbee", "NL", "Netherlands"},
		{"cirro-nl", "Cirro Parcel Netherlands", "NL", "Netherlands"},
		{"dhl-nl-private", "DHL Parcel Netherlands", "NL", "Netherlands"},
		{"dragonfly-nl", "Dragonfly Netherlands", "NL", "Netherlands"},
		{"dynalogic", "Dynalogic", "NL", "Netherlands"},
		{"gls-netherlands", "GLS Netherlands", "NL", "Netherlands"},
		{"intrapost-nl", "Intrapost", "NL", "Netherlands"},
		{"luxembourg-post", "Luxembourg Post", "LU", "Luxembourg"},
		{"packs-nl", "Packs.nl", "NL", "Netherlands"},
		{"dpost-parcel-nl", "Parcel.nl", "NL", "Netherlands"},
		{"peddler-nl", "Peddler Express", "NL", "Netherlands"},
		{"postnl", "PostNL", "NL", "Netherlands"},
		{"postnl-international", "PostNL - Netherlands Post", "NL", "Netherlands"},
		{"transmission-nl", "Transmission", "NL", "Netherlands"},
		{"trunkrs-nl", "Trunkrs", "NL", "Netherlands"},
		{"vindmijnzending-nl", "Vind Mijn Zending", "NL", "Netherlands"},
	}

	for _, c := range beneluxList {
		if _, exists := r.Get(c.slug); !exists {
			r.Register(NewGenericBeneluxCarrier(c.slug, c.name, c.country, c.countryName))
		}
	}
}
