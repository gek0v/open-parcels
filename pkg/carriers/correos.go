package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type CorreosSpainCarrier struct {
	BaseCarrier
}

func NewCorreosSpainCarrier() *CorreosSpainCarrier {
	return &CorreosSpainCarrier{
		BaseCarrier: NewBaseCarrier(
			"correos-spain",
			"Correos España",
			"ES",
			"Spain",
			models.CarrierTypePostal,
			"https://www.correos.es",
			"https://www.correos.es/es/es/herramientas/localizador/envios/detalle?tracking-number=%s",
		),
	}
}

func (c *CorreosSpainCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()

	events := []models.TrackingEvent{
		{
			Date:        now.Add(-48 * time.Hour),
			Description: "Pre-registrado en los sistemas de Correos",
			Location:    "Centro de Admisión, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-36 * time.Hour),
			Description: "Admitido en origen",
			Location:    "Madrid CTA, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-18 * time.Hour),
			Description: "En tránsito - En camino al centro de distribución local",
			Location:    "Valencia CTA, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-4 * time.Hour),
			Description: "En reparto - El cartero ha iniciado el reparto del envío",
			Location:    "Valencia Unidad de Reparto, Spain",
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
		DaysInTransit: 2,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}
