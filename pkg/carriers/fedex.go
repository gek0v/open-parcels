package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type FedExCarrier struct {
	BaseCarrier
}

func NewFedExCarrier() *FedExCarrier {
	return &FedExCarrier{
		BaseCarrier: NewBaseCarrier(
			"fedex",
			"FedEx",
			"US",
			"United States",
			models.CarrierTypeExpress,
			"https://www.fedex.com",
			"https://www.fedex.com/fedextrack/?trknbr=%s",
		),
	}
}

func (c *FedExCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-36 * time.Hour),
			Description: "Shipment information sent to FedEx",
			Location:    "Memphis, TN, United States",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-24 * time.Hour),
			Description: "Picked up and left FedEx origin facility",
			Location:    "Memphis World Hub, TN, United States",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-3 * time.Hour),
			Description: "On FedEx vehicle for delivery",
			Location:    "Destination Destination Facility",
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
		StatusText:    "On FedEx vehicle for delivery",
		Progress:      85,
		DaysInTransit: 1,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}
