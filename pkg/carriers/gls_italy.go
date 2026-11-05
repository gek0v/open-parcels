package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

// GLSItalyCarrier represents GLS Italy tracking.
type GLSItalyCarrier struct {
	BaseCarrier
}

func NewGLSItalyCarrier() *GLSItalyCarrier {
	return &GLSItalyCarrier{
		BaseCarrier: NewBaseCarrier(
			"gls-italy",
			"GLS Italy",
			"IT",
			"Italy",
			models.CarrierTypeExpress,
			"https://www.gls-italy.com",
			"https://www.gls-italy.com/it/servizi-per-destinatari/ricerca-spedizione/?match=%s",
		),
	}
}

func (c *GLSItalyCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-30 * time.Hour),
			Description: "Dati spedizione ricevuti nei sistemi GLS Italy",
			Location:    "Sede GLS Milano, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-20 * time.Hour),
			Description: "In transito - La spedizione è partita dalla sede di partenza",
			Location:    "Hub GLS San Giuliano Milanese, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-3 * time.Hour),
			Description: "In consegna - La spedizione è in consegna con l'autista GLS",
			Location:    "Sede GLS Roma Sud, Italy",
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
			Name: "Italy",
			Code: "IT",
		},
		Destination: models.CountryInfo{
			Name: "Italy",
			Code: "IT",
		},
		Status:        models.StatusOutForDelivery,
		StatusText:    "In consegna / Out for delivery",
		Progress:      85,
		DaysInTransit: 1,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}
