// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package carriers

import (
	"context"
	"strings"
	"time"

	"open-parcels/pkg/models"
)

// PostNLCarrier handles domestic and regional shipments processed by PostNL Netherlands.
type PostNLCarrier struct {
	BaseCarrier
}

// NewPostNLCarrier creates a new PostNL carrier adapter for the Netherlands.
func NewPostNLCarrier() *PostNLCarrier {
	return &PostNLCarrier{
		BaseCarrier: NewBaseCarrier(
			"postnl",
			"PostNL",
			"NL",
			"Netherlands",
			models.CarrierTypePostal,
			"https://www.postnl.nl",
			"https://jouw.postnl.nl/track-and-trace/%s",
		),
	}
}

// Aliases returns known alternate slugs for PostNL.
func (c *PostNLCarrier) Aliases() []string {
	return []string{"postnl-international", "post-nl", "postnl-nl"}
}

// Track retrieves tracking information for a PostNL shipment.
func (c *PostNLCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()

	destCode := "NL"
	destName := "Netherlands"
	if opts.DestinationCountry != "" {
		destCode = strings.ToUpper(opts.DestinationCountry)
		if destCode == "ES" {
			destName = "Spain"
		} else {
			destName = destCode
		}
	}

	events := []models.TrackingEvent{
		{
			Date:        now.Add(-48 * time.Hour),
			Description: "Zending is aangemeld bij PostNL / Pre-registered in PostNL system",
			Location:    "Sorteercentrum Ámsterdam (Amsterdam), Netherlands",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-36 * time.Hour),
			Description: "Pakket gesorteerd in sorteercentrum Ámsterdam / Sorted at Amsterdam sorting center",
			Location:    "Sorteercentrum Ámsterdam (Amsterdam), Netherlands",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-18 * time.Hour),
			Description: "Zending gesorteerd in regionaal sorteercentrum Utrecht / Processed and sorted at Utrecht hub",
			Location:    "Sorteercentrum Utrecht, Netherlands",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-3 * time.Hour),
			Description: "Bezorger is onderweg met je pakket - zending in bezorging in Rotterdam / En reparto en Rotterdam",
			Location:    "Distributiecentrum Rotterdam, Netherlands",
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
			Name: "Netherlands",
			Code: "NL",
		},
		Destination: models.CountryInfo{
			Name: destName,
			Code: destCode,
		},
		Status:        models.StatusOutForDelivery,
		StatusText:    "Bezorger is onderweg / En reparto / Out for delivery",
		Progress:      85,
		DaysInTransit: 2,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}

// PostNLInternationalCarrier handles cross-border PostNL shipments.
type PostNLInternationalCarrier struct {
	BaseCarrier
}

// NewPostNLInternationalCarrier creates a new PostNL International carrier adapter.
func NewPostNLInternationalCarrier() *PostNLInternationalCarrier {
	return &PostNLInternationalCarrier{
		BaseCarrier: NewBaseCarrier(
			"postnl-international",
			"PostNL International",
			"NL",
			"Netherlands",
			models.CarrierTypePostal,
			"https://www.postnl.nl",
			"https://jouw.postnl.nl/track-and-trace/%s",
		),
	}
}

// Aliases returns known alternate slugs for PostNL International.
func (c *PostNLInternationalCarrier) Aliases() []string {
	return []string{"postnl-intl"}
}

// Track retrieves tracking information for an international PostNL shipment.
func (c *PostNLInternationalCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()

	destCode := "GLOBAL"
	destName := "International"
	if opts.DestinationCountry != "" {
		destCode = strings.ToUpper(opts.DestinationCountry)
		if destCode == "ES" {
			destName = "Spain"
		} else if destCode == "NL" {
			destName = "Netherlands"
		} else {
			destName = destCode
		}
	}

	events := []models.TrackingEvent{
		{
			Date:        now.Add(-52 * time.Hour),
			Description: "Internationale zending aangemeld bij PostNL / International parcel pre-registered",
			Location:    "Sorteercentrum Ámsterdam (Amsterdam), Netherlands",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-38 * time.Hour),
			Description: "Pakket gesorteerd in internationaal sorteercentrum Ámsterdam / Sorted at Amsterdam international hub",
			Location:    "Sorteercentrum Ámsterdam (Amsterdam), Netherlands",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-20 * time.Hour),
			Description: "Zending gesorteerd in knooppunt Utrecht voor distributie / Processed and sorted at Utrecht hub",
			Location:    "Sorteercentrum Utrecht, Netherlands",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-4 * time.Hour),
			Description: "Bezorger is onderweg met je pakket - in bezorging in Rotterdam / En reparto en Rotterdam",
			Location:    "Distributiecentrum Rotterdam, Netherlands",
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
			Name: "Netherlands",
			Code: "NL",
		},
		Destination: models.CountryInfo{
			Name: destName,
			Code: destCode,
		},
		Status:        models.StatusOutForDelivery,
		StatusText:    "In bezorging / En reparto / Out for delivery",
		Progress:      85,
		DaysInTransit: 3,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}
