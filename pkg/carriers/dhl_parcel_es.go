package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type DHLParcelSpainCarrier struct {
	BaseCarrier
}

func NewDHLParcelSpainCarrier() *DHLParcelSpainCarrier {
	return &DHLParcelSpainCarrier{
		BaseCarrier: NewBaseCarrier(
			"dhl-parcel-es",
			"DHL Parcel Spain",
			"ES",
			"Spain",
			models.CarrierTypeLogistics,
			"https://www.dhl.com/es-es/home/seguimiento.html",
			"https://www.dhl.com/es-es/home/seguimiento.html?tracking-id=%s",
		),
	}
}

func (c *DHLParcelSpainCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-24 * time.Hour),
			Description: "Datos del envío recibidos electrónicamente / Electronic data received",
			Location:    "Madrid Hub, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-12 * time.Hour),
			Description: "El envío ha llegado a la central de paquetería / Shipment arrived at parcel center",
			Location:    "Getafe Parcel Hub, Madrid, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-2 * time.Hour),
			Description: "El envío está en reparto hacia destino / Loaded onto delivery vehicle",
			Location:    "Madrid Delegación, Spain",
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
			Name: "Spain",
			Code: "ES",
		},
		Destination: models.CountryInfo{
			Name: "Spain",
			Code: "ES",
		},
		Status:        models.StatusOutForDelivery,
		StatusText:    "En reparto / Out for delivery",
		Progress:      85,
		DaysInTransit: 1,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}
