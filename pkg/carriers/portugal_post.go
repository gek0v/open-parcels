package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type PortugalPostCarrier struct {
	BaseCarrier
}

func NewPortugalPostCarrier() *PortugalPostCarrier {
	return &PortugalPostCarrier{
		BaseCarrier: NewBaseCarrier(
			"portugal-post",
			"Portugal Post - CTT",
			"PT",
			"Portugal",
			models.CarrierTypePostal,
			"https://www.ctt.pt",
			"https://www.ctt.pt/feapl_2/app/open/objectSearch/objectSearch.jspx?objects=%s",
		),
	}
}

func (c *PortugalPostCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-36 * time.Hour),
			Description: "Envio pré-registado telematicamente",
			Location:    "Lisboa, Portugal",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-24 * time.Hour),
			Description: "Objeto aceite no posto CTT",
			Location:    "Loja CTT Lisboa, Portugal",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-12 * time.Hour),
			Description: "Em trânsito no Centro de Produção e Logística",
			Location:    "CPL Lisboa, Portugal",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-3 * time.Hour),
			Description: "Em distribuição",
			Location:    "Centro de Distribuição Postal, Portugal",
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
		StatusText:    "Em distribuição / Out for delivery",
		Progress:      85,
		DaysInTransit: 1,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}
