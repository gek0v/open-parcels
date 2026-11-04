package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type USPSCarrier struct {
	BaseCarrier
}

func NewUSPSCarrier() *USPSCarrier {
	return &USPSCarrier{
		BaseCarrier: NewBaseCarrier(
			"usps",
			"USPS",
			"US",
			"United States",
			models.CarrierTypePostal,
			"https://tools.usps.com",
			"https://tools.usps.com/go/TrackConfirmAction?tLabels=%s",
		),
	}
}

func (c *USPSCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-40 * time.Hour),
			Description: "Accepted at USPS Origin Facility",
			Location:    "Portland, OR 97201, United States",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-24 * time.Hour),
			Description: "Arrived at USPS Regional Destination Facility",
			Location:    "Austin TX Distribution Center, United States",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-6 * time.Hour),
			Description: "Arrived at Post Office",
			Location:    "Austin, TX 78701, United States",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-2 * time.Hour),
			Description: "Out for Delivery, Expected Delivery by 4:00pm",
			Location:    "Austin, TX 78701, United States",
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
			Name: "United States",
			Code: "US",
		},
		Destination: models.CountryInfo{
			Name: "United States",
			Code: "US",
		},
		Status:        models.StatusOutForDelivery,
		StatusText:    "Out for Delivery",
		Progress:      85,
		DaysInTransit: 2,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}
