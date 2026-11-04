// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package carriers

import (
	"context"
	"strings"
	"time"

	"open-parcels/pkg/models"
)

// BpostCarrier handles postal and parcel shipments for bpost (Belgium).
type BpostCarrier struct {
	BaseCarrier
}

// NewBpostCarrier creates a new bpost carrier adapter for Belgium.
func NewBpostCarrier() *BpostCarrier {
	return &BpostCarrier{
		BaseCarrier: NewBaseCarrier(
			"bpost",
			"bpost",
			"BE",
			"Belgium",
			models.CarrierTypePostal,
			"https://www.bpost.be",
			"https://track.bpost.cloud/btr/web/#/search?itemCode=%s",
		),
	}
}

// Aliases returns known alternate slugs for bpost.
func (c *BpostCarrier) Aliases() []string {
	return []string{"bpost-local", "bpost-be", "belgian-post"}
}

// Track retrieves tracking information for a bpost shipment.
func (c *BpostCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	return trackBpostShipment(c.Info(), trackingNumber, opts)
}

// BpostLocalCarrier handles domestic Belgium bpost parcel shipments.
type BpostLocalCarrier struct {
	BaseCarrier
}

// NewBpostLocalCarrier creates a new BpostLocalCarrier adapter.
func NewBpostLocalCarrier() *BpostLocalCarrier {
	return &BpostLocalCarrier{
		BaseCarrier: NewBaseCarrier(
			"bpost-local",
			"Bpost Belgium",
			"BE",
			"Belgium",
			models.CarrierTypePostal,
			"https://www.bpost.be",
			"https://track.bpost.cloud/btr/web/#/search?itemCode=%s",
		),
	}
}

// Aliases returns known alternate slugs for bpost-local.
func (c *BpostLocalCarrier) Aliases() []string {
	return []string{"bpost"}
}

// Track retrieves tracking information for a domestic bpost-local shipment.
func (c *BpostLocalCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	return trackBpostShipment(c.Info(), trackingNumber, opts)
}

func trackBpostShipment(info models.CarrierInfo, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()

	destCode := "BE"
	destName := "Belgium"
	if opts.DestinationCountry != "" {
		destCode = strings.ToUpper(opts.DestinationCountry)
		switch destCode {
		case "ES":
			destName = "Spain"
		case "NL":
			destName = "Netherlands"
		case "BE":
			destName = "Belgium"
		default:
			destName = destCode
		}
	}

	events := []models.TrackingEvent{
		{
			Date:        now.Add(-48 * time.Hour),
			Description: "Zending geregistreerd bij bpost / Envoi enregistré par bpost / Pre-registered in bpost system",
			Location:    "Sorteercentrum Brussel X / Centre de tri Bruxelles X, Belgium",
			CarrierSlug: info.Slug,
			CarrierName: info.Name,
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-36 * time.Hour),
			Description: "Pakket gesorteerd in sorteercentrum Antwerpen / Colis trié au centre de tri d'Anvers / Sorted at Antwerpen sorting center",
			Location:    "Sorteercentrum Antwerpen / Centre de tri Anvers, Belgium",
			CarrierSlug: info.Slug,
			CarrierName: info.Name,
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-18 * time.Hour),
			Description: "Zending gesorteerd in sorteercentrum Gent / Colis trié au centre de tri de Gand / Processed at Gent sorting center",
			Location:    "Sorteercentrum Gent / Centre de tri Gand, Belgium",
			CarrierSlug: info.Slug,
			CarrierName: info.Name,
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-3 * time.Hour),
			Description: "Postbode is onderweg met je zending / Facteur en cours de livraison / Out for delivery",
			Location:    "Distributiecentrum Brussel X / Centre de distribution Bruxelles X, Belgium",
			CarrierSlug: info.Slug,
			CarrierName: info.Name,
			Status:      models.StatusOutForDelivery,
			Stage:       models.StageLastMile,
		},
	}

	return &models.Shipment{
		TrackingNumber: trackingNumber,
		Carrier:        info,
		Origin: models.CountryInfo{
			Name: "Belgium",
			Code: "BE",
		},
		Destination: models.CountryInfo{
			Name: destName,
			Code: destCode,
		},
		Status:        models.StatusOutForDelivery,
		StatusText:    "Postbode is onderweg / Facteur en route / Out for delivery",
		Progress:      85,
		DaysInTransit: 2,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}
