package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type YunExpressCarrier struct {
	BaseCarrier
}

func NewYunExpressCarrier() *YunExpressCarrier {
	return &YunExpressCarrier{
		BaseCarrier: NewBaseCarrier(
			"yunexpress",
			"YunExpress",
			"CN",
			"China",
			models.CarrierTypeCrossBorder,
			"https://www.yunexpress.com",
			"https://www.yuntrack.com/parcelTracking?id=%s",
		),
	}
}

func (c *YunExpressCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	destCountry := "Spain"
	destCode := "ES"
	if opts.DestinationCountry != "" {
		destCountry = opts.DestinationCountry
		destCode = opts.DestinationCountry
	}

	events := []models.TrackingEvent{
		{
			Date:        now.Add(-7 * 24 * time.Hour),
			Description: "YunExpress shipment information received",
			Location:    "Shenzhen, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-6 * 24 * time.Hour),
			Description: "Package arrived at sorting facility",
			Location:    "Dongguan Hub, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-5 * 24 * time.Hour),
			Description: "Customs declaration completed at origin airport",
			Location:    "Hong Kong Airport, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageCustoms,
		},
		{
			Date:        now.Add(-3 * 24 * time.Hour),
			Description: "Departed on international flight",
			Location:    "Hong Kong Airport, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-40 * time.Hour),
			Description: "Arrived at destination country international airport",
			Location:    "Madrid-Barajas Airport, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-24 * time.Hour),
			Description: "Import customs clearance completed",
			Location:    "Madrid Customs Facility, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageCustoms,
		},
		{
			Date:        now.Add(-14 * time.Hour),
			Description: "Transferred to destination carrier for local delivery",
			Location:    "Madrid Logistics Hub, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageHandover,
		},
	}

	lastMileNum := "PK" + trackingNumber[max(0, len(trackingNumber)-10):] + "01"

	return &models.Shipment{
		TrackingNumber:         trackingNumber,
		Carrier:                c.Info(),
		LastMileTrackingNumber: lastMileNum,
		LastMileCarrier: &models.CarrierInfo{
			Slug:        "correos-spain",
			Name:        "Correos España",
			Country:     destCode,
			CountryName: destCountry,
			Type:        models.CarrierTypePostal,
			Website:     "https://www.correos.es",
			TrackingURL: "https://www.correos.es/es/es/herramientas/localizador/envios/detalle?tracking-number=%s",
		},
		Origin: models.CountryInfo{
			Name: "China",
			Code: "CN",
		},
		Destination: models.CountryInfo{
			Name: destCountry,
			Code: destCode,
		},
		Status:        models.StatusInTransit,
		StatusText:    "In Transit - Handed over to local carrier",
		Progress:      72,
		DaysInTransit: 7,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}
