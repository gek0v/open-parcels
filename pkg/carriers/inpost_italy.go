package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

// InPostItalyCarrier represents InPost.it locker and delivery network.
type InPostItalyCarrier struct {
	BaseCarrier
}

func NewInPostItalyCarrier() *InPostItalyCarrier {
	return &InPostItalyCarrier{
		BaseCarrier: NewBaseCarrier(
			"inpost-it",
			"InPost.it",
			"IT",
			"Italy",
			models.CarrierTypeLogistics,
			"https://inpost.it",
			"https://inpost.it/trova-il-tuo-pacco?number=%s",
		),
	}
}

func (c *InPostItalyCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-30 * time.Hour),
			Description: "Spedizione registrata dal mittente per il punto di ritiro / locker",
			Location:    "Punto InPost Milano, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-18 * time.Hour),
			Description: "Pacco ritirato e in transito verso il centro logistico",
			Location:    "Hub InPost Bologna, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-3 * time.Hour),
			Description: "In consegna presso il locker / Punto InPost di destinazione",
			Location:    "Locker InPost Roma, Italy",
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
		StatusText:    "In consegna presso Locker / Punto InPost",
		Progress:      85,
		DaysInTransit: 1,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}
