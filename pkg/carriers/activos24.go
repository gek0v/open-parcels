package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type Activos24Carrier struct {
	BaseCarrier
}

func NewActivos24Carrier() *Activos24Carrier {
	return &Activos24Carrier{
		BaseCarrier: NewBaseCarrier(
			"activos24",
			"Activos24",
			"ES",
			"Spain",
			models.CarrierTypeExpress,
			"https://www.activos24.com",
			"https://www.activos24.com/seguimiento?num=%s",
		),
	}
}

func (c *Activos24Carrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-20 * time.Hour),
			Description: "Datos del envío registrados telemáticamente / Shipping data received",
			Location:    "Madrid, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-14 * time.Hour),
			Description: "Envío admitido y en clasificación / Parcel received at sorting facility",
			Location:    "Coslada Hub, Madrid, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-3 * time.Hour),
			Description: "En reparto - Paquete en furgoneta de entrega / Out for delivery",
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
