package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type DHLExpressCarrier struct {
	BaseCarrier
}

func NewDHLExpressCarrier() *DHLExpressCarrier {
	return &DHLExpressCarrier{
		BaseCarrier: NewBaseCarrier(
			"dhl-express",
			"DHL Express",
			"GLOBAL",
			"Global",
			models.CarrierTypeExpress,
			"https://www.dhl.com",
			"https://www.dhl.com/en/express/tracking.html?AWB=%s",
		),
	}
}

func (c *DHLExpressCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-24 * time.Hour),
			Description: "Shipment picked up",
			Location:    "Frankfurt, Germany",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-14 * time.Hour),
			Description: "Processed at Leipzig Hub, Germany",
			Location:    "Leipzig Hub, Germany",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-4 * time.Hour),
			Description: "With delivery courier - Madrid, Spain",
			Location:    "Madrid Delivery Station, Spain",
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
			Name: "Germany",
			Code: "DE",
		},
		Destination: models.CountryInfo{
			Name: "Spain",
			Code: "ES",
		},
		Status:        models.StatusOutForDelivery,
		StatusText:    "With delivery courier",
		Progress:      85,
		DaysInTransit: 1,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}

type DHLParcelCarrier struct {
	BaseCarrier
}

func NewDHLParcelCarrier() *DHLParcelCarrier {
	return &DHLParcelCarrier{
		BaseCarrier: NewBaseCarrier(
			"dhl-parcel",
			"DHL Parcel",
			"DE",
			"Germany",
			models.CarrierTypeLogistics,
			"https://www.dhl.de",
			"https://www.dhl.de/en/privatkunden/pakete-empfangen/verfolgen.html?piececode=%s",
		),
	}
}

func (c *DHLParcelCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-30 * time.Hour),
			Description: "The instruction data for this shipment have been provided by the sender to DHL electronically",
			Location:    "Bonn, Germany",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-18 * time.Hour),
			Description: "The shipment has been loaded onto the delivery vehicle",
			Location:    "Köln Paketzentrum, Germany",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusOutForDelivery,
			Stage:       models.StageLastMile,
		},
	}

	return &models.Shipment{
		TrackingNumber: trackingNumber,
		Carrier:        c.Info(),
		Status:        models.StatusOutForDelivery,
		StatusText:    "Loaded onto delivery vehicle",
		Progress:      85,
		DaysInTransit: 1,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}
