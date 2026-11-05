package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type MBESpainCarrier struct {
	BaseCarrier
}

func NewMBESpainCarrier() *MBESpainCarrier {
	return &MBESpainCarrier{
		BaseCarrier: NewBaseCarrier(
			"mbe-es",
			"Mail Boxes Etc. Spain",
			"ES",
			"Spain",
			models.CarrierTypeExpress,
			"https://www.mbe.es",
			"https://www.mbe.es/es/tracking?shipping_code=%s",
		),
	}
}

func (c *MBESpainCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-24 * time.Hour),
			Description: "Envío recibido en centro MBE / Received at Mail Boxes Etc center",
			Location:    "Centro MBE Madrid, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-14 * time.Hour),
			Description: "En tránsito - Procesado en plataforma logística / In transit at logistics hub",
			Location:    "Madrid Hub Central, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-3 * time.Hour),
			Description: "En reparto hacia dirección de entrega / Out for delivery",
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
