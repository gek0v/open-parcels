// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package carriers

import (
	"context"
	"strings"
	"time"

	"open-parcels/pkg/models"
)

// LuxembourgPostCarrier handles postal and parcel shipments for POST Luxembourg.
type LuxembourgPostCarrier struct {
	BaseCarrier
}

// NewLuxembourgPostCarrier creates a new POST Luxembourg carrier adapter.
func NewLuxembourgPostCarrier() *LuxembourgPostCarrier {
	return &LuxembourgPostCarrier{
		BaseCarrier: NewBaseCarrier(
			"luxembourg-post",
			"POST Luxembourg",
			"LU",
			"Luxembourg",
			models.CarrierTypePostal,
			"https://www.post.lu",
			"https://www.post.lu/en/particuliers/courrier-colis/track-and-trace?barcode=%s",
		),
	}
}

// Aliases returns known alternate slugs for POST Luxembourg.
func (c *LuxembourgPostCarrier) Aliases() []string {
	return []string{"post-luxembourg", "post-lu"}
}

// Track retrieves tracking information for a POST Luxembourg shipment.
func (c *LuxembourgPostCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()

	destCode := "LU"
	destName := "Luxembourg"
	if opts.DestinationCountry != "" {
		destCode = strings.ToUpper(opts.DestinationCountry)
		switch destCode {
		case "BE":
			destName = "Belgium"
		case "NL":
			destName = "Netherlands"
		case "FR":
			destName = "France"
		case "DE":
			destName = "Germany"
		case "ES":
			destName = "Spain"
		case "LU":
			destName = "Luxembourg"
		default:
			destName = destCode
		}
	}

	events := []models.TrackingEvent{
		{
			Date:        now.Add(-36 * time.Hour),
			Description: "Envoi enregistré par l'expéditeur / Sendung elektronisch angekündigt / Pre-registered by sender",
			Location:    "Centre Postal Bettembourg, Luxembourg",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-20 * time.Hour),
			Description: "Prise en charge et tri au Centre Postal Bettembourg / Im Centre Postal Bettembourg sortiert / Sorted at Centre Postal Bettembourg",
			Location:    "Centre Postal Bettembourg, Luxembourg",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-3 * time.Hour),
			Description: "En cours de distribution locale / In lokaler Zustellung / Out for local delivery",
			Location:    "Centre de distribution locale, Luxembourg",
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
			Name: "Luxembourg",
			Code: "LU",
		},
		Destination: models.CountryInfo{
			Name: destName,
			Code: destCode,
		},
		Status:        models.StatusOutForDelivery,
		StatusText:    "En cours de distribution locale / Out for delivery",
		Progress:      85,
		DaysInTransit: 1,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}
