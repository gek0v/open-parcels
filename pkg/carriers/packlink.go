package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type PacklinkCarrier struct {
	BaseCarrier
}

func NewPacklinkCarrier() *PacklinkCarrier {
	return &PacklinkCarrier{
		BaseCarrier: NewBaseCarrier(
			"packlink",
			"Packlink",
			"ES",
			"Spain",
			models.CarrierTypeLogistics,
			"https://www.packlink.es",
			"https://packlink.es/seguimiento?tracking=%s",
		),
	}
}

func (c *PacklinkCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-26 * time.Hour),
			Description: "Etiqueta generada y pedido confirmado en Packlink / Label created",
			Location:    "Madrid, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-14 * time.Hour),
			Description: "Envío recogido por el transportista seleccionado / Picked up by partner carrier",
			Location:    "Centro Logístico, Madrid, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-3 * time.Hour),
			Description: "En reparto hacia dirección del destinatario / Out for delivery",
			Location:    "Delegación Local, Spain",
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
