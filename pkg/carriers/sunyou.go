package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type SunYouCarrier struct {
	BaseCarrier
}

func NewSunYouCarrier() *SunYouCarrier {
	return &SunYouCarrier{
		BaseCarrier: NewBaseCarrier(
			"sytrack",
			"SunYou - Sytrack",
			"CN",
			"China",
			models.CarrierTypeCrossBorder,
			"https://www.sypost.net",
			"https://www.sypost.net/search?orderNo=%s",
		),
	}
}

func (c *SunYouCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
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
			Description: "Logistics order generated / Pre-shipment information received",
			Location:    "Shenzhen, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-5 * 24 * time.Hour),
			Description: "SunYou facility acceptance scan",
			Location:    "Dongguan Processing Center, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-4 * 24 * time.Hour),
			Description: "Export customs cleared / Dispatched from SunYou transit hub",
			Location:    "Shenzhen Hub, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageCustoms,
		},
		{
			Date:        now.Add(-3 * 24 * time.Hour),
			Description: "Handed over to airline / Departed origin airport",
			Location:    "Hong Kong International Airport, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-30 * time.Hour),
			Description: "Arrival at destination country exchange facility",
			Location:    destCountry + " Transit Hub",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-14 * time.Hour),
			Description: "Transferred to destination postal courier for distribution",
			Location:    destCountry + " Sorting Hub",
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
			Name: "China",
			Code: "CN",
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
