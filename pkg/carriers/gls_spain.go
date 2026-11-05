package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type GLSSpainCarrier struct {
	BaseCarrier
}

func NewGLSSpainCarrier() *GLSSpainCarrier {
	return &GLSSpainCarrier{
		BaseCarrier: NewBaseCarrier(
			"asm-gls-spain",
			"GLS Spain (ASM)",
			"ES",
			"Spain",
			models.CarrierTypeExpress,
			"https://www.gls-spain.es",
			"https://www.gls-spain.es/es/seguimiento-envios/?match=%s",
		),
	}
}

func (c *GLSSpainCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-24 * time.Hour),
			Description: "Datos del paquete registrados en el sistema / Parcel registered in system",
			Location:    "Madrid Delegación ASM/GLS, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-14 * time.Hour),
			Description: "El paquete ha salido del centro logístico / Parcel left sorting center",
			Location:    "San Fernando Hub, Madrid, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-3 * time.Hour),
			Description: "En reparto con el repartidor de GLS / Out for delivery with courier",
			Location:    "Madrid, Spain",
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
