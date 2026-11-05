package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

// BRTBartoliniCarrier represents BRT Bartolini standard shipment tracking.
type BRTBartoliniCarrier struct {
	BaseCarrier
}

func NewBRTBartoliniCarrier() *BRTBartoliniCarrier {
	return &BRTBartoliniCarrier{
		BaseCarrier: NewBaseCarrier(
			"brt-bartolini",
			"BRT Bartolini",
			"IT",
			"Italy",
			models.CarrierTypeExpress,
			"https://www.brt.it",
			"https://vas.brt.it/vas/sped_det_show.hsm?nspediz=%s",
		),
	}
}

func (c *BRTBartoliniCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-30 * time.Hour),
			Description: "Preso in carico - Spedizione ritirata dal corriere",
			Location:    "Bologna Hub, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-16 * time.Hour),
			Description: "In lavorazione - Arrivata nella filiale di smistamento",
			Location:    "Firenze Filiale, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-3 * time.Hour),
			Description: "In consegna - Spedizione in consegna nella giornata odierna",
			Location:    "Firenze Filiale, Italy",
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

// BRTItalyParcelIDCarrier represents BRT Bartolini tracking via Parcel ID.
type BRTItalyParcelIDCarrier struct {
	BaseCarrier
}

func NewBRTItalyParcelIDCarrier() *BRTItalyParcelIDCarrier {
	return &BRTItalyParcelIDCarrier{
		BaseCarrier: NewBaseCarrier(
			"brt-italy-parcelid",
			"BRT Bartolini (Parcel ID)",
			"IT",
			"Italy",
			models.CarrierTypeExpress,
			"https://www.brt.it",
			"https://vas.brt.it/vas/sped_det_show.hsm?parcelid=%s",
		),
	}
}

func (c *BRTItalyParcelIDCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-28 * time.Hour),
			Description: "Preso in carico - Registrata spedizione con Parcel ID",
			Location:    "Verona Hub, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-14 * time.Hour),
			Description: "In transito - Spedizione inoltrata alla filiale di arrivo",
			Location:    "Torino Filiale, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-2 * time.Hour),
			Description: "In consegna - In consegna con incaricato BRT",
			Location:    "Torino Filiale, Italy",
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

// MyBRTCarrier represents myBRT.it online tracking portal.
type MyBRTCarrier struct {
	BaseCarrier
}

func NewMyBRTCarrier() *MyBRTCarrier {
	return &MyBRTCarrier{
		BaseCarrier: NewBaseCarrier(
			"mybrt-it",
			"myBRT.it",
			"IT",
			"Italy",
			models.CarrierTypeExpress,
			"https://mybrt.it",
			"https://mybrt.it/it/mybrt/track-and-trace?parcelNumber=%s",
		),
	}
}

func (c *MyBRTCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-24 * time.Hour),
			Description: "Dati spedizione registrati nel portale myBRT",
			Location:    "Milano Hub, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-12 * time.Hour),
			Description: "Spedizione in transito verso il punto di consegna",
			Location:    "Brescia Hub, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-2 * time.Hour),
			Description: "In consegna al destinatario tramite myBRT",
			Location:    "Brescia Centro, Italy",
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
