package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type DPDPortugalCarrier struct {
	BaseCarrier
}

func NewDPDPortugalCarrier() *DPDPortugalCarrier {
	return &DPDPortugalCarrier{
		BaseCarrier: NewBaseCarrier(
			"dpd-portugal",
			"DPD Portugal",
			"PT",
			"Portugal",
			models.CarrierTypeExpress,
			"https://dpd.pt",
			"https://tracking.dpd.pt/status/pt-PT/%s",
		),
	}
}

func (c *DPDPortugalCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-28 * time.Hour),
			Description: "Pedido transmitido eletronicamente ao transportador",
			Location:    "Lisboa, Portugal",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-16 * time.Hour),
			Description: "Recebido no HUB DPD Lisboa",
			Location:    "HUB Lisboa, Portugal",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-4 * time.Hour),
			Description: "Em distribuição na viatura de entrega",
			Location:    "Centro Operacional DPD, Portugal",
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
