package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type ZelerisCarrier struct {
	BaseCarrier
}

func NewZelerisCarrier() *ZelerisCarrier {
	return &ZelerisCarrier{
		BaseCarrier: NewBaseCarrier(
			"zeleris",
			"Zeleris",
			"ES",
			"Spain",
			models.CarrierTypeExpress,
			"https://www.zeleris.com",
			"https://www.zeleris.com/seguimiento-envio.do?tracking=%s",
		),
	}
}

func (c *ZelerisCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-22 * time.Hour),
			Description: "Envío pre-alertado telemáticamente / Electronic pre-advice received",
			Location:    "Madrid, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-12 * time.Hour),
			Description: "En tránsito - Clasificado en plataforma logística / Processed at hub",
			Location:    "Hub Central Zeleris, Madrid, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-2 * time.Hour),
			Description: "En reparto - Paquete en vehículo de distribución / Out for delivery",
			Location:    "Centro de Reparto Zeleris, Spain",
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
