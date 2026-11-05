package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type DelnextCarrier struct {
	BaseCarrier
}

func NewDelnextCarrier() *DelnextCarrier {
	return &DelnextCarrier{
		BaseCarrier: NewBaseCarrier(
			"delnext",
			"Delnext",
			"PT",
			"Portugal",
			models.CarrierTypeExpress,
			"https://www.delnext.com",
			"https://www.delnext.com/tracking/?tracking=%s",
		),
	}
}

func (c *DelnextCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-30 * time.Hour),
			Description: "Envio registado na plataforma Delnext",
			Location:    "Lisboa, Portugal",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-18 * time.Hour),
			Description: "Objeto recolhido e encaminhado para o armazém central",
			Location:    "Armazém Delnext Lisboa, Portugal",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-5 * time.Hour),
			Description: "Em rota de entrega ao domicílio",
			Location:    "Delegação Regional, Portugal",
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
		StatusText:    "Em rota de entrega",
		Progress:      85,
		DaysInTransit: 1,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}
