package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type HongKongPostCarrier struct {
	BaseCarrier
}

func NewHongKongPostCarrier() *HongKongPostCarrier {
	return &HongKongPostCarrier{
		BaseCarrier: NewBaseCarrier(
			"hongkong-post",
			"Hong Kong Post",
			"HK",
			"Hong Kong",
			models.CarrierTypePostal,
			"https://www.hongkongpost.hk",
			"https://www.hongkongpost.hk/en/mail_tracking/index.html?tracking_number=%s",
		),
	}
}

func (c *HongKongPostCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	destCountry := "Spain"
	destCode := "ES"
	if opts.DestinationCountry != "" {
		destCountry = opts.DestinationCountry
		destCode = opts.DestinationCountry
	}

	events := []models.TrackingEvent{
		{
			Date:        now.Add(-6 * 24 * time.Hour),
			Description: "Item posted at post office",
			Location:    "Kowloon Central Post Office, Hong Kong",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-5 * 24 * time.Hour),
			Description: "Arrived at Hong Kong Air Mail Centre sorting facility",
			Location:    "Hong Kong Air Mail Centre, Hong Kong",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-4 * 24 * time.Hour),
			Description: "Processed for international dispatch and export security scan",
			Location:    "Hong Kong Air Mail Centre, Hong Kong",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageCustoms,
		},
		{
			Date:        now.Add(-3 * 24 * time.Hour),
			Description: "Handed over to airline / Departed Hong Kong for destination",
			Location:    "Hong Kong International Airport, Hong Kong",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-36 * time.Hour),
			Description: "Arrived at destination airport and processing at inward mail center",
			Location:    destCountry + " International Air Mail Centre",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-20 * time.Hour),
			Description: "Import customs clearance completed successfully",
			Location:    destCountry + " Postal Customs",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageCustoms,
		},
		{
			Date:        now.Add(-8 * time.Hour),
			Description: "Handed over to destination postal administration for local delivery",
			Location:    destCountry + " Postal Hub",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageHandover,
		},
	}

	return &models.Shipment{
		TrackingNumber: trackingNumber,
		Carrier:        c.Info(),
		Origin: models.CountryInfo{
			Name: "Hong Kong",
			Code: "HK",
		},
		Destination: models.CountryInfo{
			Name: destCountry,
			Code: destCode,
		},
		Status:        models.StatusInTransit,
		StatusText:    "In Transit - International air mail dispatch",
		Progress:      65,
		DaysInTransit: 6,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}
