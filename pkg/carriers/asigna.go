package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type AsignaCarrier struct {
	BaseCarrier
}

func NewAsignaCarrier() *AsignaCarrier {
	return &AsignaCarrier{
		BaseCarrier: NewBaseCarrier(
			"asigna-es",
			"Asigna",
			"ES",
			"Spain",
			models.CarrierTypeExpress,
			"https://www.asigna.es",
			"https://www.asigna.es/seguimiento?envio=%s",
		),
	}
}

func (c *AsignaCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-22 * time.Hour),
			Description: "Datos del envío registrados telemáticamente / Electronic info registered",
			Location:    "Madrid, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-12 * time.Hour),
			Description: "Envío recibido en plataforma logística / Received at distribution center",
			Location:    "Coslada Hub, Madrid, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-2 * time.Hour),
			Description: "En reparto hacia destino / Out for delivery to destination",
			Location:    "Madrid, Spain",
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
