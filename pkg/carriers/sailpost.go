package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

// SailpostCarrier represents Sailpost postal and express courier serving Italy and San Marino.
type SailpostCarrier struct {
	BaseCarrier
}

func NewSailpostCarrier() *SailpostCarrier {
	return &SailpostCarrier{
		BaseCarrier: NewBaseCarrier(
			"sailpost",
			"Sailpost",
			"IT",
			"Italy & San Marino",
			models.CarrierTypePostal,
			"https://www.sailpost.it",
			"https://www.sailpost.it/tracking/?barcode=%s",
		),
	}
}

func (c *SailpostCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-36 * time.Hour),
			Description: "Preso in carico presso agenzia Sailpost",
			Location:    "Agenzia Sailpost Pisa, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-18 * time.Hour),
			Description: "In lavorazione presso l'hub postale Sailpost",
			Location:    "Hub Sailpost Bologna, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-3 * time.Hour),
			Description: "In consegna al destinatario / San Marino recapito",
			Location:    "Agenzia Sailpost Rimini / San Marino, Italy",
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
			Name: "San Marino",
			Code: "SM",
		},
		Status:        models.StatusOutForDelivery,
		StatusText:    "In consegna / Out for delivery",
		Progress:      85,
		DaysInTransit: 1,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}
