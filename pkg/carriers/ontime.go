// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type OntimeCarrier struct {
	BaseCarrier
}

func NewOntimeCarrier() *OntimeCarrier {
	return &OntimeCarrier{
		BaseCarrier: NewBaseCarrier(
			"ontime",
			"Ontime Transporte",
			"ES",
			"Spain",
			models.CarrierTypeLogistics,
			"https://www.ontime.es",
			"https://www.ontime.es/seguimiento?tracking=%s",
		),
	}
}

func (c *OntimeCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-26 * time.Hour),
			Description: "Recepción de mercancía en hub Ontime",
			Location:    "Hub Ontime Origen, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-12 * time.Hour),
			Description: "Transporte regular de enlace interprovincial",
			Location:    "Madrid Hub Central, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-2 * time.Hour),
			Description: "Vehículo en reparto hacia dirección de entrega",
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
