package carriers

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
		{"ace-ee", "ACE Logistics Estonia", "EE", "Estonia"},
		{"ace-lv", "ACE Logistics Latvia", "LV", "Latvia"},
		{"best-se", "Best Transport Sweden", "SE", "Sweden"},
		{"bring", "Bring", "NO", "Norway"},
		{"bringer-parcel", "Bringer Parcel Services", "SE", "Sweden"},
		{"cargobus", "Cargobus", "SE", "Sweden"},
		{"citymail-se", "City Mail", "SE", "Sweden"},
		{"collicare-no", "ColliCare", "NO", "Norway"},
		{"coolrunner-dk", "CoolRunner", "DK", "Denmark"},
		{"fragt-dk", "Danske Fragt", "DK", "Denmark"},
		{"dao", "DAO", "DK", "Denmark"},
		{"dropp-is", "Dropp.is", "IS", "Iceland"},
		{"earlybird-se", "EarlyBird", "SE", "Sweden"},
		{"omniva", "Estonia Post", "EE", "Estonia"},
		{"posti-finland", "Finland Post", "FI", "Finland"},
		{"helthjem", "Helthjem", "NO", "Norway"},
		{"iceland-post", "Iceland Post", "IS", "Iceland"},
		{"latvia-post", "Latvia Post", "LV", "Latvia"},
		{"post-litva", "Lithuania Post", "LT", "Lithuania"},
		{"matkahuolto", "Matkahuolto", "FI", "Finland"},
		{"omniva-ee", "Omniva Estonia", "EE", "Estonia"},
		{"posten-norway", "Posten Norge - Norway Post", "NO", "Norway"},
		{"sweden-post", "PostNord", "SE", "Sweden"},
	}

	for _, c := range nordicList {
		if _, exists := r.Get(c.slug); !exists {
			r.Register(NewGenericNordicCarrier(c.slug, c.name, c.country, c.countryName))
		}
	}
}
