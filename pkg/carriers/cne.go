package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type CNECarrier struct {
	BaseCarrier
}

func NewCNECarrier() *CNECarrier {
	return &CNECarrier{
		BaseCarrier: NewBaseCarrier(
			"cne-express",
			"CNE Express",
			"CN",
			"China",
			models.CarrierTypeCrossBorder,
			"https://www.cne.com",
			"https://www.cne.com/English/detail?code=%s",
		),
	}
}

func (c *CNECarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
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
			Description: "Electronic shipment data registered at CNE system",
			Location:    "Shanghai, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-5 * 24 * time.Hour),
			Description: "Package processed at CNE Express outbound sorting facility",
			Location:    "CNE Shanghai International Hub, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-4 * 24 * time.Hour),
			Description: "Customs declaration completed for international export",
			Location:    "Shanghai Pudong Airport, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageCustoms,
		},
		{
			Date:        now.Add(-3 * 24 * time.Hour),
			Description: "Flight departed origin airport for destination country",
			Location:    "Shanghai Pudong Airport, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-36 * time.Hour),
			Description: "Arrived at destination country port of entry and customs cleared",
			Location:    destCountry + " International Hub",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageCustoms,
		},
		{
			Date:        now.Add(-12 * time.Hour),
			Description: "Handover to local destination postal provider for last-mile delivery",
			Location:    destCountry + " Postal Hub",
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
		StatusText:    "In Transit - International dispatch",
		Progress:      65,
		DaysInTransit: 6,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}
