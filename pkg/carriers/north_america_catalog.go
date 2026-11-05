package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

// GenericNACarrier models a North American regional, express, or LTL carrier.
type GenericNACarrier struct {
	BaseCarrier
}

func NewGenericNACarrier(slug, name, country string) *GenericNACarrier {
	countryName := "United States"
	if country == "CA" {
		countryName = "Canada"
	}
	return &GenericNACarrier{
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

func (c *GenericNACarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
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
			Description: "Shipping label created / Electronic billing information received by " + c.Name(),
			Location:    c.CountryName() + " Origin Facility",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-2 * 24 * time.Hour),
			Description: "Picked up and departed regional sort center",
			Location:    c.CountryName() + " Regional Hub",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-14 * time.Hour),
			Description: "Arrived at destination delivery facility",
			Location:    destCountry + " Local Depot",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-3 * time.Hour),
			Description: "Out for delivery on courier vehicle",
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
			Name: c.CountryName(),
			Code: c.Country(),
		},
		Destination: models.CountryInfo{
			Name: destCountry,
			Code: destCode,
		},
		Status:        models.StatusOutForDelivery,
		StatusText:    "Out for delivery",
		Progress:      85,
		DaysInTransit: 3,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}

func registerNorthAmericaCatalog(r *Registry) {
	naList := []struct {
		slug    string
		name    string
		country string
	}{
		{"aduiepyle", "A Duie Pyle", "US"},
		{"aaa-cooper", "AAA Cooper", "US"},
		{"abf-freight", "ABF Freight", "US"},
		{"swiship-com", "Amazon Swiship US", "US"},
		{"apple-express", "Apple Express", "US"},
		{"asendia-usa", "Asendia United States", "US"},
		{"averitt", "Averitt Express", "US"},
		{"better-trucks", "Better Trucks", "US"},
		{"canada-post", "Canada Post", "CA"},
		{"canpar", "Canpar", "CA"},
		{"capital-express", "Capital Express", "US"},
		{"cdl", "CDL", "US"},
		{"centraltransport", "Central Transport", "US"},
		{"chitchats", "Chit Chats", "CA"},
		{"coyote-logistics", "Coyote Logistics", "US"},
		{"cross-country-freight-solutions", "CrossCountry Freight Solutions", "US"},
		{"day-ross-freight", "Day & Ross Freight", "US"},
		{"daylight-transport", "Daylight Transport", "US"},
		{"dayton-freight", "Dayton Freight", "US"},
		{"dhl-global-forwarding-us", "DHL Global Forwarding US", "US"},
		{"dohrn", "Dohrn Transfer Company", "US"},
		{"domainlogistics-ca", "Domain Logistics", "CA"},
		{"dragonfly-ca", "Dragonfly Shipping Canada", "CA"},
		{"e24metro-ca", "E24", "CA"},
		{"eparcel-us", "Eparcel US", "US"},
		{"estes-express", "Estes Express", "US"},
		{"flashbird-ca", "Flashbird", "CA"},
		{"frontier-ca", "Frontier", "CA"},
		{"gls-us", "GLS", "US"},
		{"gls-canada", "GLS Canada", "CA"},
		{"intelcom-express", "Intelcom Express", "CA"},
		{"loomis-express", "Loomis Express", "US"},
		{"nls-ca", "National Logistics Services", "CA"},
		{"ontrac", "OnTrac", "US"},
		{"polaris-ca", "Polaris Transport", "CA"},
		{"purolator", "Purolator", "CA"},
		{"purolator-international", "Purolator International", "US"},
		{"saia-ltl-freight", "Saia LTL Freight", "US"},
		{"shippie-ca", "Shippie", "CA"},
		{"sprintstar-ca", "Sprint Star", "CA"},
		{"usps", "USPS", "US"},
		{"usps-iv-envelope", "USPS Informed Visibility Envelope", "US"},
	}

	for _, c := range naList {
		if _, exists := r.Get(c.slug); !exists {
			r.Register(NewGenericNACarrier(c.slug, c.name, c.country))
		}
	}
}
