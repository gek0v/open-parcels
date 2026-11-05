package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type InPostSpainCarrier struct {
	BaseCarrier
}

func NewInPostSpainCarrier() *InPostSpainCarrier {
	return &InPostSpainCarrier{
		BaseCarrier: NewBaseCarrier(
			"inpost-es",
			"InPost Spain",
			"ES",
			"Spain",
			models.CarrierTypeLogistics,
			"https://inpost.es",
			"https://inpost.es/seguimiento-del-envio/?number=%s",
		),
	}
}

func (c *InPostSpainCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-28 * time.Hour),
			Description: "Envío registrado por remitente para Punto Pack o Locker / Parcel registered for locker",
			Location:    "Madrid, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-16 * time.Hour),
			Description: "Paquete recogido y en tránsito a nave de clasificación / In transit to depot",
			Location:    "Centro de Clasificación, Madrid, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-3 * time.Hour),
			Description: "Paquete en reparto hacia Punto Pack / Locker de destino / Out for delivery to locker",
			Location:    "Madrid Locker, Spain",
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
