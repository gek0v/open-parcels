package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type SEURCarrier struct {
	BaseCarrier
}

func NewSEURCarrier() *SEURCarrier {
	return &SEURCarrier{
		BaseCarrier: NewBaseCarrier(
			"seur",
			"SEUR",
			"ES",
			"Spain",
			models.CarrierTypeExpress,
			"https://www.seur.com",
			"https://www.seur.com/livetracking/?segOnlineIdioma=es&segOnlineIdentificador=%s",
		),
	}
}

func (c *SEURCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-28 * time.Hour),
			Description: "Datos del envío registrados por el remitente",
			Location:    "Madrid, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-16 * time.Hour),
			Description: "En tránsito - Envío en plataforma de clasificación",
			Location:    "Madrid Hub Central, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-2 * time.Hour),
			Description: "En reparto - Su paquete está en reparto con el conductor",
			Location:    "Unidad de Reparto Local, Spain",
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
