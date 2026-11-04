// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type PosteSanMarinoCarrier struct {
	BaseCarrier
}

func NewPosteSanMarinoCarrier() *PosteSanMarinoCarrier {
	return &PosteSanMarinoCarrier{
		BaseCarrier: NewBaseCarrier(
			"poste-san-marino",
			"Poste San Marino",
			"SM",
			"San Marino",
			models.CarrierTypePostal,
			"https://www.poste.sm",
			"https://www.poste.sm/tracking?code=%s",
		),
	}
}

func (c *PosteSanMarinoCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()

	events := []models.TrackingEvent{
		{
			Date:        now.Add(-36 * time.Hour),
			Description: "Accettato presso l'Ufficio Postale di San Marino",
			Location:    "Ufficio Postale San Marino Città, San Marino",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-20 * time.Hour),
			Description: "In lavorazione presso il CMP di San Marino",
			Location:    "Centro Postale Operativo, San Marino",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-4 * time.Hour),
			Description: "In consegna - Spedizione in consegna al destinatario",
			Location:    "Borgo Maggiore, San Marino",
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
			Name: "San Marino",
			Code: "SM",
		},
		Destination: models.CountryInfo{
			Name: "San Marino",
			Code: "SM",
		},
		Status:        models.StatusOutForDelivery,
		StatusText:    "In consegna / Out for delivery",
		Progress:      85,
		DaysInTransit: 2,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}
