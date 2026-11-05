package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

// TNTItalyCarrier represents TNT Italy express delivery services.
type TNTItalyCarrier struct {
	BaseCarrier
}

func NewTNTItalyCarrier() *TNTItalyCarrier {
	return &TNTItalyCarrier{
		BaseCarrier: NewBaseCarrier(
			"tnt-italy",
			"TNT Italy",
			"IT",
			"Italy",
			models.CarrierTypeExpress,
			"https://www.tnt.it",
			"https://www.tnt.it/tracking/Tracking.do?searchType=cons&cons=%s",
		),
	}
}

func (c *TNTItalyCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-28 * time.Hour),
			Description: "Spedizione ritirata dal corriere TNT",
			Location:    "Filiale TNT Torino, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-15 * time.Hour),
			Description: "Spedizione in transito nell'hub principale",
			Location:    "Hub TNT Piacenza, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-2 * time.Hour),
			Description: "Spedizione in consegna con incaricato TNT",
			Location:    "Filiale TNT Napoli, Italy",
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
