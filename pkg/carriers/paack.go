package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type PaackCarrier struct {
	BaseCarrier
}

func NewPaackCarrier() *PaackCarrier {
	return &PaackCarrier{
		BaseCarrier: NewBaseCarrier(
			"paack",
			"Paack Logistics",
			"ES",
			"Spain",
			models.CarrierTypeExpress,
			"https://paack.co",
			"https://paack.co/tracking/%s",
		),
	}
}

func (c *PaackCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-18 * time.Hour),
			Description: "Order details received by Paack",
			Location:    "Paack Fulfillment Hub, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-10 * time.Hour),
			Description: "Parcel arrived at local distribution warehouse",
			Location:    "Local Delivery Station, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-2 * time.Hour),
			Description: "Out for delivery with Paack courier driver",
			Location:    "En ruta hacia destino, Spain",
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
			Name: "Spain",
			Code: "ES",
		},
		Destination: models.CountryInfo{
			Name: "Spain",
			Code: "ES",
		},
		Status:        models.StatusOutForDelivery,
		StatusText:    "Out for delivery scheduled today",
		Progress:      85,
		DaysInTransit: 1,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}
