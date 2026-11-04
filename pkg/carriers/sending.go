// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type SendingCarrier struct {
	BaseCarrier
}

func NewSendingCarrier() *SendingCarrier {
	return &SendingCarrier{
		BaseCarrier: NewBaseCarrier(
			"sending",
			"Sending Transporte Urgente",
			"ES",
			"Spain",
			models.CarrierTypeExpress,
			"https://www.sending.es",
			"https://www.sending.es/tracking?id=%s",
		),
	}
}

func (c *SendingCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-22 * time.Hour),
			Description: "Envío admitido en delegación Sending",
			Location:    "Delegación Origen, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-11 * time.Hour),
			Description: "En tránsito - Clasificación en centro neurálgico",
			Location:    "Hub Central, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-2 * time.Hour),
			Description: "En reparto - Envío asignado a repartidor",
			Location:    "Delegación Destino, Spain",
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
