package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type GLSCarrier struct {
	BaseCarrier
}

func NewGLSCarrier() *GLSCarrier {
	return &GLSCarrier{
		BaseCarrier: NewBaseCarrier(
			"gls",
			"GLS",
			"GLOBAL",
			"Global",
			models.CarrierTypeExpress,
			"https://gls-group.eu",
			"https://gls-group.eu/track/%s",
		),
	}
}

func (c *GLSCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-30 * time.Hour),
			Description: "The parcel data was entered into the GLS system",
			Location:    "GLS Depot",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-20 * time.Hour),
			Description: "The parcel has left the parcel center",
			Location:    "Central Hub",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-5 * time.Hour),
			Description: "The parcel is out for delivery with the GLS delivery driver",
			Location:    "Destination Depot",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusOutForDelivery,
			Stage:       models.StageLastMile,
		},
	}

	return &models.Shipment{
		TrackingNumber: trackingNumber,
		Carrier:        c.Info(),
		Status:        models.StatusOutForDelivery,
		StatusText:    "Out for delivery",
		Progress:      85,
		DaysInTransit: 1,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}
