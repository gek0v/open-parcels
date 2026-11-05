package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type WePickUpCarrier struct {
	BaseCarrier
}

func NewWePickUpCarrier() *WePickUpCarrier {
	return &WePickUpCarrier{
		BaseCarrier: NewBaseCarrier(
			"wepickup-pt",
			"WePickUp",
			"PT",
			"Portugal",
			models.CarrierTypeLogistics,
			"https://wepickup.pt",
			"https://wepickup.pt/tracking?id=%s",
		),
	}
}

func (c *WePickUpCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-24 * time.Hour),
			Description: "Reserva de envio criada",
			Location:    "Lisboa, Portugal",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-14 * time.Hour),
			Description: "Objeto recolhido no Ponto / Locker de origem",
			Location:    "Ponto WePickUp Lisboa, Portugal",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-2 * time.Hour),
			Description: "Disponível para levantamento no Ponto / Locker",
			Location:    "Locker WePickUp Destino, Portugal",
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
			Name: "Portugal",
			Code: "PT",
		},
		Destination: models.CountryInfo{
			Name: "Portugal",
			Code: "PT",
		},
		Status:        models.StatusOutForDelivery,
		StatusText:    "Disponível para levantamento no Ponto / Locker",
		Progress:      90,
		DaysInTransit: 1,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}
