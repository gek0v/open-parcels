package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type AsendiaSpainCarrier struct {
	BaseCarrier
}

func NewAsendiaSpainCarrier() *AsendiaSpainCarrier {
	return &AsendiaSpainCarrier{
		BaseCarrier: NewBaseCarrier(
			"asendia-es",
			"Asendia Spain",
			"ES",
			"Spain",
			models.CarrierTypeCrossBorder,
			"https://www.asendia.es",
			"https://tracking.asendia.com/tracking/%s",
		),
	}
}

func (c *AsendiaSpainCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-28 * time.Hour),
			Description: "Datos del envío registrados telemáticamente / Electronic shipment info received",
			Location:    "Madrid, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-16 * time.Hour),
			Description: "Envío procesado en centro logístico internacional / Processed at international hub",
			Location:    "Madrid Barajas Logistics Hub, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-4 * time.Hour),
			Description: "En tránsito - Transferido a red de reparto nacional / In transit to delivery facility",
			Location:    "Madrid, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageLastMile,
		},
	}

	return &models.Shipment{
		TrackingNumber: trackingNumber,
		Carrier:        c.Info(),
		Origin: models.CountryInfo{
			Name: "Spain",
			Code: "ES",
		},
		Destination: models.CountryInfo{
			Name: "Spain",
			Code: "ES",
		},
		Status:        models.StatusInTransit,
		StatusText:    "En tránsito / In transit",
		Progress:      65,
		DaysInTransit: 1,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}
