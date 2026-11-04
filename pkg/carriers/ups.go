package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type UPSCarrier struct {
	BaseCarrier
}

func NewUPSCarrier() *UPSCarrier {
	return &UPSCarrier{
		BaseCarrier: NewBaseCarrier(
			"ups",
			"UPS",
			"US",
			"United States",
			models.CarrierTypeExpress,
			"https://www.ups.com",
			"https://www.ups.com/track?tracknum=%s",
		),
	}
}

func (c *UPSCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-48 * time.Hour),
			Description: "Shipper created a label, UPS has not received the package yet",
			Location:    "Louisville, KY, United States",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-36 * time.Hour),
			Description: "Origin Scan / Picked up by UPS",
			Location:    "Worldport Hub, Louisville, KY, United States",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-18 * time.Hour),
			Description: "Departed from Facility",
			Location:    "Philadelphia Hub, PA, United States",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-3 * time.Hour),
			Description: "Out For Delivery - Today by 7:00 PM",
			Location:    "Local Delivery Facility, United States",
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
