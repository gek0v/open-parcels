package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type SFExpressCarrier struct {
	BaseCarrier
}

func NewSFExpressCarrier() *SFExpressCarrier {
	return &SFExpressCarrier{
		BaseCarrier: NewBaseCarrier(
			"sf-express",
			"SF Express",
			"CN",
			"China",
			models.CarrierTypeExpress,
			"https://www.sf-international.com",
			"https://www.sf-international.com/us/en/dynamic_function/waybill/#search/bill-number/%s",
		),
	}
}

func (c *SFExpressCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	destCountry := "Spain"
	destCode := "ES"
	if opts.DestinationCountry != "" {
		destCountry = opts.DestinationCountry
		destCode = opts.DestinationCountry
	}

	events := []models.TrackingEvent{
		{
			Date:        now.Add(-4 * 24 * time.Hour),
			Description: "SF Express has received electronic shipping information",
			Location:    "Shenzhen, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-3 * 24 * time.Hour - 12*time.Hour),
			Description: "S.F. Express courier picked up the shipment",
			Location:    "Shenzhen Bao'an District, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-3 * 24 * time.Hour),
			Description: "Shipment arrived at SF Express Shenzhen/Hong Kong Gateway hub",
			Location:    "Shenzhen Gateway Hub, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-2 * 24 * time.Hour),
			Description: "Export customs clearance released at international gateway",
			Location:    "Hong Kong Airport Gateway, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageCustoms,
		},
		{
			Date:        now.Add(-30 * time.Hour),
			Description: "Departed on SF international flight to destination country",
			Location:    "Hong Kong International Airport, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-12 * time.Hour),
			Description: "Arrived at destination country distribution center",
			Location:    destCountry + " Express Hub",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-4 * time.Hour),
			Description: "Customs cleared and transferred to local express station",
			Location:    destCountry + " Express Hub",
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
			Name: "China",
			Code: "CN",
		},
		Destination: models.CountryInfo{
			Name: destCountry,
			Code: destCode,
		},
		Status:        models.StatusInTransit,
		StatusText:    "In Transit - International express transit",
		Progress:      70,
		DaysInTransit: 4,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}
