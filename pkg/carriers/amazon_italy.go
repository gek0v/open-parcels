package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

// AmazonItalyCarrier represents Amazon Shipping Italy.
type AmazonItalyCarrier struct {
	BaseCarrier
}

func NewAmazonItalyCarrier() *AmazonItalyCarrier {
	return &AmazonItalyCarrier{
		BaseCarrier: NewBaseCarrier(
			"amazon-it",
			"Amazon Italy",
			"IT",
			"Italy",
			models.CarrierTypeLogistics,
			"https://track.amazon.it",
			"https://track.amazon.it/tracking/%s",
		),
	}
}

func (c *AmazonItalyCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-24 * time.Hour),
			Description: "Pacco affidato al corriere Amazon",
			Location:    "Centro Distribuzione MXP5, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-10 * time.Hour),
			Description: "In transito verso la stazione di consegna",
			Location:    "Centro di Smistamento FCO1, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-2 * time.Hour),
			Description: "In consegna con l'autista Amazon",
			Location:    "Stazione di Consegna DLO2, Italy",
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
		StatusText:    "In consegna con corriere Amazon",
		Progress:      85,
		DaysInTransit: 1,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}
