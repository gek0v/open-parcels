package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type FourPXCarrier struct {
	BaseCarrier
}

func NewFourPXCarrier() *FourPXCarrier {
	return &FourPXCarrier{
		BaseCarrier: NewBaseCarrier(
			"four-px",
			"4PX Express",
			"CN",
			"China",
			models.CarrierTypeCrossBorder,
			"https://track.4px.com",
			"https://track.4px.com/#/result/1/%s",
		),
	}
}

func (c *FourPXCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-5 * 24 * time.Hour),
			Description: "Shipment information received",
			Location:    "Shenzhen, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-4 * 24 * time.Hour),
			Description: "Parcel arrived at 4PX Shenzhen outbound sorting facility",
			Location:    "Shenzhen Hub, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-2 * 24 * time.Hour),
			Description: "Departed transit country / Export customs released",
			Location:    "Hong Kong Airport, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageCustoms,
		},
		{
			Date:        now.Add(-18 * time.Hour),
			Description: "Arrived at destination country international airport",
			Location:    "Destination Port of Entry",
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
			Name: "China",
			Code: "CN",
		},
		Destination: models.CountryInfo{
			Name: "Destination",
			Code: opts.DestinationCountry,
		},
		Status:        models.StatusInTransit,
		StatusText:    "In Transit - International transit",
		Progress:      60,
		DaysInTransit: 5,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}
