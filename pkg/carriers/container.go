package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type OceanContainerCarrier struct {
	BaseCarrier
}

func NewOceanContainerCarrier() *OceanContainerCarrier {
	return &OceanContainerCarrier{
		BaseCarrier: NewBaseCarrier(
			"ocean-container",
			"ISO Ocean Container",
			"GLOBAL",
			"International",
			models.CarrierTypeFreight,
			"https://www.bic-code.org",
			"",
		),
	}
}

func (c *OceanContainerCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()

	events := []models.TrackingEvent{
		{
			Date:        now.Add(-10 * 24 * time.Hour),
			Description: "Gate in full at origin port terminal",
			Location:    "Port of Shanghai, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-8 * 24 * time.Hour),
			Description: "Loaded on board vessel",
			Location:    "Port of Shanghai, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-2 * 24 * time.Hour),
			Description: "Vessel in transit on high seas",
			Location:    "Mediterranean Sea",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
	}

	return &models.Shipment{
		TrackingNumber: trackingNumber,
		Carrier:        c.Info(),
		Origin: models.CountryInfo{
			Name: "Shanghai, China",
			Code: "CNSHA",
		},
		Destination: models.CountryInfo{
			Name: "Valencia, Spain",
			Code: "ESVLC",
		},
		Status:        models.StatusInTransit,
		StatusText:    "In Transit - At Sea",
		Progress:      50,
		DaysInTransit: 10,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}
