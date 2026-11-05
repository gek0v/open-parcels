package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type SendingCarrier struct {
	BaseCarrier
}

func NewSendingCarrier() *SendingCarrier {
	return &SendingCarrier{
		BaseCarrier: NewBaseCarrier(
			"sending-es",
			"Sending",
			"ES",
			"Spain",
			models.CarrierTypeExpress,
			"https://sending.es",
			"https://sending.es/seguimiento?envio=%s",
		),
	}
}

func (c *SendingCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-22 * time.Hour),
			Description: "Datos del envío grabados / Electronic dispatch advice received",
			Location:    "Madrid, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-12 * time.Hour),
			Description: "En tránsito - Llegada a plataforma de distribución / Processed at hub",
			Location:    "Coslada Hub Sending, Madrid, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-2 * time.Hour),
			Description: "En reparto - Envío asignado a ruta / Out for delivery",
			Location:    "Delegación Destino Sending, Spain",
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
