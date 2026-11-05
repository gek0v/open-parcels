package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type DepponCarrier struct {
	BaseCarrier
}

func NewDepponCarrier() *DepponCarrier {
	return &DepponCarrier{
		BaseCarrier: NewBaseCarrier(
			"deppon",
			"Deppon",
			"CN",
			"China",
			models.CarrierTypeLogistics,
			"https://www.deppon.com",
			"https://www.deppon.com/track?numbers=%s",
		),
	}
}

func (c *DepponCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	destCountry := "China"
	destCode := "CN"
	if opts.DestinationCountry != "" {
		destCountry = opts.DestinationCountry
		destCode = opts.DestinationCountry
	}

	events := []models.TrackingEvent{
		{
			Date:        now.Add(-4 * 24 * time.Hour),
			Description: "Deppon logistics order confirmed / Waybill printed",
			Location:    "Guangzhou, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-3 * 24 * time.Hour - 12*time.Hour),
			Description: "Deppon logistics driver completed pickup scan",
			Location:    "Guangzhou Baiyun Depot, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-3 * 24 * time.Hour),
			Description: "Package sorted at Deppon South China Transshipment Center",
			Location:    "Guangzhou Transshipment Center, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-2 * 24 * time.Hour),
			Description: "Highway freight vehicle departed on regional trunk route",
			Location:    "Guangzhou Trunk Depot, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-18 * time.Hour),
			Description: "Arrived at destination city transfer depot and offloaded",
			Location:    "Destination Logistics Depot, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-4 * time.Hour),
			Description: "Dispatched to local distribution team for delivery",
			Location:    "Destination Logistics Station, China",
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
		StatusText:    "In Transit - Logistics freight transit",
		Progress:      70,
		DaysInTransit: 4,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}
