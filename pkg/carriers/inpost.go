package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type InPostCarrier struct {
	BaseCarrier
}

func NewInPostCarrier() *InPostCarrier {
	return &InPostCarrier{
		BaseCarrier: NewBaseCarrier(
			"inpost",
			"InPost / Mondial Relay",
			"GLOBAL",
			"Global",
			models.CarrierTypeLogistics,
			"https://inpost.es",
			"https://inpost.es/seguimiento-del-envio/?number=%s",
		),
	}
}

func (c *InPostCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-30 * time.Hour),
			Description: "Parcel registered by sender for locker drop-off",
			Location:    "Point Relais / Locker, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-18 * time.Hour),
			Description: "Parcel collected from locker and in transit to sorting depot",
			Location:    "Regional Depot, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-4 * time.Hour),
			Description: "Parcel is being delivered to destination locker / Punto Pack",
			Location:    "Destination Locker, Spain",
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
		StatusText:    "Out for delivery to Locker / Pickup Point",
		Progress:      85,
		DaysInTransit: 1,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}
