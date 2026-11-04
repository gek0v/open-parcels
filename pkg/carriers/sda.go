// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type SDACarrier struct {
	BaseCarrier
}

func NewSDACarrier() *SDACarrier {
	return &SDACarrier{
		BaseCarrier: NewBaseCarrier(
			"sda",
			"SDA Express Courier",
			"IT",
			"Italy",
			models.CarrierTypeExpress,
			"https://www.sda.it",
			"https://www.sda.it/wps/portal/Servizi_online/dettaglio_spedizione?id_spedizione=%s",
		),
	}
}

func (c *SDACarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()

	events := []models.TrackingEvent{
		{
			Date:        now.Add(-32 * time.Hour),
			Description: "La spedizione è stata presa in carico da SDA",
			Location:    "Centro Operativo SDA Roma, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-18 * time.Hour),
			Description: "In lavorazione presso l'Hub di smistamento",
			Location:    "Hub Centrale Bologna, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-4 * time.Hour),
			Description: "In consegna - Spedizione in consegna con l'autista",
			Location:    "Filiale di Consegna Napoli, Italy",
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
			Name: "Italy",
			Code: "IT",
		},
		Destination: models.CountryInfo{
			Name: "Italy",
			Code: "IT",
		},
		Status:        models.StatusOutForDelivery,
		StatusText:    "In consegna / Out for delivery",
		Progress:      85,
		DaysInTransit: 1,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}
