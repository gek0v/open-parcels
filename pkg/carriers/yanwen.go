package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type YanwenCarrier struct {
	BaseCarrier
}

func NewYanwenCarrier() *YanwenCarrier {
	return &YanwenCarrier{
		BaseCarrier: NewBaseCarrier(
			"yanwen",
			"Yanwen Logistics",
			"CN",
			"China",
			models.CarrierTypeCrossBorder,
			"https://www.yanwenlogistics.cn",
			"https://www.yanwenlogistics.cn/track/?nums=%s",
		),
	}
}

func (c *YanwenCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
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
			Description: "Yanwen Pickup scan / Shipment information received",
			Location:    "Yiwu Distribution Hub, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-5 * 24 * time.Hour),
			Description: "Process completed at Yanwen International Express Sorting Center",
			Location:    "Shenzhen Outbound Hub, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-4 * 24 * time.Hour),
			Description: "Export customs declaration released",
			Location:    "Shenzhen Bao'an Airport, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageCustoms,
		},
		{
			Date:        now.Add(-3 * 24 * time.Hour),
			Description: "Uplifted by air flight / Port of departure",
			Location:    "Shenzhen / Hong Kong Airport, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-36 * time.Hour),
			Description: "Port of destination arrival / Arrival at destination sorting facility",
			Location:    destCountry + " International Hub",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-20 * time.Hour),
			Description: "Import customs cleared in destination country",
			Location:    destCountry + " Customs Office",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageCustoms,
		},
		{
			Date:        now.Add(-8 * time.Hour),
			Description: "Handed over to local carrier for final mile delivery",
			Location:    destCountry + " Delivery Depot",
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
		StatusText:    "In Transit - International cross-border delivery",
		Progress:      65,
		DaysInTransit: 6,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}
