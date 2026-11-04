// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type EnvialiaCarrier struct {
	BaseCarrier
}

func NewEnvialiaCarrier() *EnvialiaCarrier {
	return &EnvialiaCarrier{
		BaseCarrier: NewBaseCarrier(
			"envialia",
			"Envialia",
			"ES",
			"Spain",
			models.CarrierTypeExpress,
			"https://www.envialia.com",
			"https://www.envialia.com/seguimiento/?num=%s",
		),
	}
}

func (c *EnvialiaCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-20 * time.Hour),
			Description: "Recogido en centro de origen Envialia",
			Location:    "Centro Origen, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-10 * time.Hour),
			Description: "En tránsito por plataforma de enlace",
			Location:    "Madrid Hub Central, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-2 * time.Hour),
			Description: "En reparto al destinatario",
			Location:    "Centro Destino, Spain",
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
