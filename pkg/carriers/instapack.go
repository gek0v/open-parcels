package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type InstapackCarrier struct {
	BaseCarrier
}

func NewInstapackCarrier() *InstapackCarrier {
	return &InstapackCarrier{
		BaseCarrier: NewBaseCarrier(
			"instapack-es",
			"Instapack",
			"ES",
			"Spain",
			models.CarrierTypeExpress,
			"https://instapack.es",
			"https://instapack.es/tracking?id=%s",
		),
	}
}

func (c *InstapackCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-18 * time.Hour),
			Description: "Solicitud de envío recibida / Shipment request received",
			Location:    "Barcelona, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-10 * time.Hour),
			Description: "Envío recogido por mensajero / Parcel collected by courier",
			Location:    "Barcelona Hub, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-2 * time.Hour),
			Description: "En reparto exprés hacia destinatario / Out for express delivery",
			Location:    "Barcelona, Spain",
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
		StatusText:    "En reparto / Out for delivery",
		Progress:      85,
		DaysInTransit: 1,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}
