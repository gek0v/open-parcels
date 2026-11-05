package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type CeleritasCarrier struct {
	BaseCarrier
}

func NewCeleritasCarrier() *CeleritasCarrier {
	return &CeleritasCarrier{
		BaseCarrier: NewBaseCarrier(
			"celeritas",
			"Celeritas Transporte",
			"ES",
			"Spain",
			models.CarrierTypeLogistics,
			"https://celeritastransporte.com",
			"https://celeritastransporte.com/seguimiento/?tracking=%s",
		),
	}
}

func (c *CeleritasCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-24 * time.Hour),
			Description: "Datos del envío registrados telemáticamente / Shipping data created",
			Location:    "Madrid, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-14 * time.Hour),
			Description: "Envío recibido en nave de distribución / Parcel processed at hub",
			Location:    "San Fernando de Henares, Madrid, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-3 * time.Hour),
			Description: "En reparto hacia Punto Celeritas / Domicilio / Out for delivery",
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
