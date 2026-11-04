package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type CorreosExpressCarrier struct {
	BaseCarrier
}

func NewCorreosExpressCarrier() *CorreosExpressCarrier {
	return &CorreosExpressCarrier{
		BaseCarrier: NewBaseCarrier(
			"correos-express",
			"Correos Express",
			"ES",
			"Spain",
			models.CarrierTypeExpress,
			"https://www.correosexpress.com",
			"https://www.correosexpress.com/web/correosexpress/consultanos?shippingNumber=%s",
		),
	}
}

func (c *CorreosExpressCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()

	// Typical Spanish Correos Express timeline
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-18 * time.Hour),
			Description: "Sin recibir - Envío informado electrónicamente",
			Location:    "Madrid, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-12 * time.Hour),
			Description: "En tránsito - Envío clasificado en nave central",
			Location:    "Coslada (Madrid), Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-6 * time.Hour),
			Description: "En tránsito - Llegada a delegación de destino",
			Location:    "Barcelona Delegación, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageLastMile,
		},
		{
			Date:        now.Add(-2 * time.Hour),
			Description: "En reparto - Envío asignado al repartidor para entrega",
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
