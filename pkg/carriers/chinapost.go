package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type ChinaPostCarrier struct {
	BaseCarrier
}

func NewChinaPostCarrier() *ChinaPostCarrier {
	return &ChinaPostCarrier{
		BaseCarrier: NewBaseCarrier(
			"china-post",
			"China Post",
			"CN",
			"China",
			models.CarrierTypePostal,
			"https://yjcx.ems.com.cn",
			"https://yjcx.ems.com.cn/qps/yjcx?mailNo=%s",
		),
	}
}

func (c *ChinaPostCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	destCountry := "Spain"
	destCode := "ES"
	if opts.DestinationCountry != "" {
		destCountry = opts.DestinationCountry
		destCode = opts.DestinationCountry
	}

	events := []models.TrackingEvent{
		{
			Date:        now.Add(-8 * 24 * time.Hour),
			Description: "Item posted and collected at Post Branch",
			Location:    "Beijing International Post Office, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-7 * 24 * time.Hour),
			Description: "Arrived at outward office of exchange sorting center",
			Location:    "Beijing EMS Sorting Center, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-6 * 24 * time.Hour),
			Description: "Customs declaration completed at outward office of exchange",
			Location:    "Beijing International Hub, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageCustoms,
		},
		{
			Date:        now.Add(-4 * 24 * time.Hour),
			Description: "Despatched to overseas airline / Departed origin country",
			Location:    "Beijing Capital Airport, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-2 * 24 * time.Hour),
			Description: "Arrived at destination inward office of exchange",
			Location:    destCountry + " International Airport",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-24 * time.Hour),
			Description: "Import customs clearance completed",
			Location:    destCountry + " Postal Customs",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageCustoms,
		},
		{
			Date:        now.Add(-8 * time.Hour),
			Description: "Handed over to destination postal network for local delivery",
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
			Name: "China",
			Code: "CN",
		},
		Destination: models.CountryInfo{
			Name: destCountry,
			Code: destCode,
		},
		Status:        models.StatusInTransit,
		StatusText:    "In Transit - International postal transit",
		Progress:      65,
		DaysInTransit: 8,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}
