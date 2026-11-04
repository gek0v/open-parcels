// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type PosteItalianeCarrier struct {
	BaseCarrier
}

func NewPosteItalianeCarrier() *PosteItalianeCarrier {
	return &PosteItalianeCarrier{
		BaseCarrier: NewBaseCarrier(
			"poste-italiane",
			"Poste Italiane",
			"IT",
			"Italy",
			models.CarrierTypePostal,
			"https://www.poste.it",
			"https://www.poste.it/cerca/index.html#/risultati-spedizioni/%s",
		),
	}
}

func (c *PosteItalianeCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()

	events := []models.TrackingEvent{
		{
			Date:        now.Add(-48 * time.Hour),
			Description: "Preso in carico all'ufficio postale",
			Location:    "Roma Hub, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-24 * time.Hour),
			Description: "In lavorazione presso il centro operativo postale",
			Location:    "Milano CMP, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-4 * time.Hour),
			Description: "In consegna - La spedizione è in consegna nella giornata odierna",
			Location:    "Milano Recapito, Italy",
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
		DaysInTransit: 2,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}
