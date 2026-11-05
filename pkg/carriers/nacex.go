package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type NacexCarrier struct {
	BaseCarrier
}

func NewNacexCarrier() *NacexCarrier {
	return &NacexCarrier{
		BaseCarrier: NewBaseCarrier(
			"nacex-spain",
			"Nacex",
			"ES",
			"Spain",
			models.CarrierTypeExpress,
			"https://www.nacex.com",
			"https://www.nacex.com/seguimientoDetalle.do?ag_rec=&albaran_rec=%s",
		),
	}
}

func (c *NacexCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-22 * time.Hour),
			Description: "Envío recogido en origen / Parcel picked up at origin",
			Location:    "Agencia Origen Nacex, Madrid, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-12 * time.Hour),
			Description: "En tránsito - Clasificado en plataforma central / Processed at central platform",
			Location:    "Coslada Hub Nacex, Madrid, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-2 * time.Hour),
			Description: "En reparto - Paquete en ruta de entrega / Out for delivery",
			Location:    "Agencia Destino Nacex, Spain",
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
