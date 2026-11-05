package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type EnvialiaCarrier struct {
	BaseCarrier
}

func NewEnvialiaCarrier() *EnvialiaCarrier {
	return &EnvialiaCarrier{
		BaseCarrier: NewBaseCarrier(
			"envialia",
			"Envialia",
			"ES",
			"Spain",
			models.CarrierTypeExpress,
			"https://www.envialia.com",
			"https://www.envialia.com/seguimiento/?envio=%s",
		),
	}
}

func (c *EnvialiaCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-22 * time.Hour),
			Description: "Envío documentado en delegación origen / Documented at origin branch",
			Location:    "Madrid Delegación, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-12 * time.Hour),
			Description: "En tránsito - Clasificado en plataforma central / Processed at central hub",
			Location:    "Coslada Hub, Madrid, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-2 * time.Hour),
			Description: "En reparto - En vehículo de reparto para entrega / Out for delivery",
			Location:    "Madrid Delegación, Spain",
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
