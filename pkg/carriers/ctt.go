package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type CTTExpressCarrier struct {
	BaseCarrier
}

func NewCTTExpressCarrier() *CTTExpressCarrier {
	return &CTTExpressCarrier{
		BaseCarrier: NewBaseCarrier(
			"ctt-express",
			"CTT Express",
			"ES",
			"Spain",
			models.CarrierTypeExpress,
			"https://www.cttexpress.com",
			"https://www.cttexpress.com/localizador-de-envios/?sc=%s",
		),
	}
}

func (c *CTTExpressCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-24 * time.Hour),
			Description: "Envío pre-registrado telemáticamente",
			Location:    "Madrid Hub, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-14 * time.Hour),
			Description: "En tránsito - Recepcionado en plataforma de distribución",
			Location:    "Plataforma Central, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-3 * time.Hour),
			Description: "En reparto - El envío se encuentra en ruta de entrega",
			Location:    "Centro de Distribución Local, Spain",
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
