package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type YTOExpressCarrier struct {
	BaseCarrier
}

func NewYTOExpressCarrier() *YTOExpressCarrier {
	return &YTOExpressCarrier{
		BaseCarrier: NewBaseCarrier(
			"yto-express",
			"YTO Express",
			"CN",
			"China",
			models.CarrierTypeExpress,
			"https://www.yto.net.cn",
			"https://www.yto.net.cn/expressTrack.html?nums=%s",
		),
	}
}

func (c *YTOExpressCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	destCountry := "China"
	destCode := "CN"
	if opts.DestinationCountry != "" {
		destCountry = opts.DestinationCountry
		destCode = opts.DestinationCountry
	}

	events := []models.TrackingEvent{
		{
			Date:        now.Add(-3 * 24 * time.Hour),
			Description: "YTO order created / Merchant shipped package",
			Location:    "Shanghai, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-2 * 24 * time.Hour - 12*time.Hour),
			Description: "Courier collected parcel and scanned into origin station",
			Location:    "Shanghai Qingpu Station, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-2 * 24 * time.Hour),
			Description: "Dispatched from YTO Shanghai National Distribution Hub",
			Location:    "Shanghai Distribution Hub, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-18 * time.Hour),
			Description: "Arrived at destination transit center and sorted",
			Location:    "Regional Distribution Center, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-4 * time.Hour),
			Description: "Arrived at local delivery station / Assigned to courier",
			Location:    "Local Delivery Depot, China",
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
		StatusText:    "In Transit - Express parcel transit",
		Progress:      75,
		DaysInTransit: 3,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}
