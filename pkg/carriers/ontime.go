package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type OntimeCarrier struct {
	BaseCarrier
}

func NewOntimeCarrier() *OntimeCarrier {
	return &OntimeCarrier{
		BaseCarrier: NewBaseCarrier(
			"ontime",
			"Ontime",
			"ES",
			"Spain",
			models.CarrierTypeExpress,
			"https://ontime.es",
			"https://ontime.es/seguimiento/?num=%s",
		),
	}
}

func (c *OntimeCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-20 * time.Hour),
			Description: "Datos telemáticos del envío recibidos / Shipping order created",
			Location:    "Madrid, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-12 * time.Hour),
			Description: "Envío recibido en nave central Ontime / Arrived at central depot",
			Location:    "Ontime Hub Madrid, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-3 * time.Hour),
			Description: "En reparto - En ruta hacia el destinatario / Out for delivery",
			Location:    "Delegación Destino Ontime, Spain",
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
