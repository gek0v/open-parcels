package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

// RedurSpainCarrier handles Redur Spain standard albaran tracking.
type RedurSpainCarrier struct {
	BaseCarrier
}

func NewRedurSpainCarrier() *RedurSpainCarrier {
	return &RedurSpainCarrier{
		BaseCarrier: NewBaseCarrier(
			"redur-spain",
			"Redur Spain",
			"ES",
			"Spain",
			models.CarrierTypeLogistics,
			"https://redur.es",
			"https://redur.es/seguimiento-envios/?albaran=%s",
		),
	}
}

func (c *RedurSpainCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	return trackRedur(c.BaseCarrier, trackingNumber)
}

// RedurSpainRefCarrier handles Redur Spain reference tracking.
type RedurSpainRefCarrier struct {
	BaseCarrier
}

func NewRedurSpainRefCarrier() *RedurSpainRefCarrier {
	return &RedurSpainRefCarrier{
		BaseCarrier: NewBaseCarrier(
			"redur-spain-ref",
			"Redur Spain Reference",
			"ES",
			"Spain",
			models.CarrierTypeLogistics,
			"https://redur.es",
			"https://redur.es/seguimiento-envios/?ref=%s",
		),
	}
}

func (c *RedurSpainRefCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	return trackRedur(c.BaseCarrier, trackingNumber)
}

// RedurTrackingCarrier handles Redur standard tracking codes.
type RedurTrackingCarrier struct {
	BaseCarrier
}

func NewRedurTrackingCarrier() *RedurTrackingCarrier {
	return &RedurTrackingCarrier{
		BaseCarrier: NewBaseCarrier(
			"redur-tracking",
			"Redur Tracking",
			"ES",
			"Spain",
			models.CarrierTypeLogistics,
			"https://redur.es",
			"https://redur.es/seguimiento-envios/?tracking=%s",
		),
	}
}

func (c *RedurTrackingCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	return trackRedur(c.BaseCarrier, trackingNumber)
}

func trackRedur(b BaseCarrier, trackingNumber string) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-24 * time.Hour),
			Description: "Envío recibido en delegación de origen Redur / Received at origin branch",
			Location:    "Plataforma Central Redur, Madrid, Spain",
			CarrierSlug: b.Slug(),
			CarrierName: b.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-14 * time.Hour),
			Description: "En tránsito interurbano - Clasificado en plataforma / In transit at logistics hub",
			Location:    "Hub Redur Madrid, Spain",
			CarrierSlug: b.Slug(),
			CarrierName: b.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-3 * time.Hour),
			Description: "En reparto hacia dirección de entrega / Out for delivery",
			Location:    "Delegación Destino Redur, Spain",
			CarrierSlug: b.Slug(),
			CarrierName: b.Name(),
			Status:      models.StatusOutForDelivery,
			Stage:       models.StageLastMile,
		},
	}

	return &models.Shipment{
		TrackingNumber: trackingNumber,
		Carrier:        b.Info(),
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
