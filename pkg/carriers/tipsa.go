// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type TipsaCarrier struct {
	BaseCarrier
}

func NewTipsaCarrier() *TipsaCarrier {
	return &TipsaCarrier{
		BaseCarrier: NewBaseCarrier(
			"tipsa",
			"TIPSA",
			"ES",
			"Spain",
			models.CarrierTypeExpress,
			"https://www.tip-sa.com",
			"https://www.tip-sa.com/cliente/seguimiento-envios?albaran=%s",
		),
	}
}

func (c *TipsaCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-20 * time.Hour),
			Description: "Envío recogido en origen por agencia TIPSA",
			Location:    "Agencia Origen, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-10 * time.Hour),
			Description: "En tránsito - Hub central de distribución",
			Location:    "Bailén Hub, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-2 * time.Hour),
			Description: "En reparto para entrega hoy",
			Location:    "Agencia Destino, Spain",
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
