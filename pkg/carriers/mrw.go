package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type MRWCarrier struct {
	BaseCarrier
}

func NewMRWCarrier() *MRWCarrier {
	return &MRWCarrier{
		BaseCarrier: NewBaseCarrier(
			"mrw",
			"MRW",
			"ES",
			"Spain",
			models.CarrierTypeExpress,
			"https://www.mrw.es",
			"https://www.mrw.es/seguimiento_envios/mrw_resultados_consulta.asp?modo=nacional&envio=%s",
		),
	}
}

func (c *MRWCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-24 * time.Hour),
			Description: "Envío recogido en origen por franquicia MRW",
			Location:    "Franquicia Origen, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-12 * time.Hour),
			Description: "En tránsito interurbano - Plataforma logística central",
			Location:    "Coslada Hub, Madrid, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-3 * time.Hour),
			Description: "En reparto - Envío en furgoneta de reparto para entrega hoy",
			Location:    "Franquicia Destino, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusOutForDelivery,
			Stage:       models.StageLastMile,
		},
	}

	return &models.Shipment{
		TrackingNumber: trackingNumber,
		Carrier:        c.Info(),
		Status:        models.StatusOutForDelivery,
		StatusText:    "En reparto / Out for delivery",
		Progress:      85,
		DaysInTransit: 1,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}
