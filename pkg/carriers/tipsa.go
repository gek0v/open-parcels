package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type TipsaCarrier struct {
	BaseCarrier
}

func NewTipsaCarrier() *TipsaCarrier {
	return &TipsaCarrier{
		BaseCarrier: NewBaseCarrier(
			"tipsa",
			"TIPSA",
			"ES",
			"Spain",
			models.CarrierTypeExpress,
			"https://www.tip-sa.com",
			"https://www.tip-sa.com/seguimiento-envios?albaran=%s",
		),
	}
}

func (c *TipsaCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-22 * time.Hour),
			Description: "Envío recogido en origen / Parcel picked up",
			Location:    "Agencia Origen TIPSA, Madrid, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-12 * time.Hour),
			Description: "En tránsito - Clasificado en plataforma central TIPSA / Processed at central hub",
			Location:    "Plataforma Central San Fernando, Madrid, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-2 * time.Hour),
			Description: "En reparto - Paquete en vehículo de reparto / Out for delivery",
			Location:    "Agencia Destino TIPSA, Spain",
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
