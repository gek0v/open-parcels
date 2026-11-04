package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type RoyalMailCarrier struct {
	BaseCarrier
}

func NewRoyalMailCarrier() *RoyalMailCarrier {
	return &RoyalMailCarrier{
		BaseCarrier: NewBaseCarrier(
			"royal-mail",
			"Royal Mail",
			"GB",
			"United Kingdom",
			models.CarrierTypePostal,
			"https://www.royalmail.com",
			"https://www.royalmail.com/track-your-item#/tracking-results/%s",
		),
	}
}

func (c *RoyalMailCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-40 * time.Hour),
			Description: "Sender preparing item",
			Location:    "London, United Kingdom",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-24 * time.Hour),
			Description: "Item Received at Heathrow Worldwide DC",
			Location:    "Heathrow DC, United Kingdom",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-3 * time.Hour),
			Description: "Due to be delivered today by Royal Mail Postie",
			Location:    "Local Delivery Office, United Kingdom",
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
			Name: "United Kingdom",
			Code: "GB",
		},
		Status:        models.StatusOutForDelivery,
		StatusText:    "Out for delivery today",
		Progress:      85,
		DaysInTransit: 2,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}

type ColissimoCarrier struct {
	BaseCarrier
}

func NewColissimoCarrier() *ColissimoCarrier {
	return &ColissimoCarrier{
		BaseCarrier: NewBaseCarrier(
			"colissimo",
			"Colissimo / La Poste",
			"FR",
			"France",
			models.CarrierTypePostal,
			"https://www.laposte.fr",
			"https://www.laposte.fr/outils/suivre-vos-envois?code=%s",
		),
	}
}

func (c *ColissimoCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-36 * time.Hour),
			Description: "Votre colis est prêt à être expédié, il n'est pas encore pris en charge par La Poste",
			Location:    "Plateforme Paris, France",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-20 * time.Hour),
			Description: "Votre colis est en transit sur notre réseau",
			Location:    "Plateforme Briarde, France",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-3 * time.Hour),
			Description: "Votre colis est en cours de livraison",
			Location:    "Centre de distribution, France",
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
			Name: "France",
			Code: "FR",
		},
		Status:        models.StatusOutForDelivery,
		StatusText:    "En cours de livraison",
		Progress:      85,
		DaysInTransit: 1,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}
