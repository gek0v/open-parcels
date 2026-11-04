package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type AirCargoCarrier struct {
	BaseCarrier
}

func NewAirCargoCarrier() *AirCargoCarrier {
	return &AirCargoCarrier{
		BaseCarrier: NewBaseCarrier(
			"air-cargo",
			"IATA Air Cargo",
			"GLOBAL",
			"International",
			models.CarrierTypeFreight,
			"https://www.iata.org",
			"",
		),
	}
}

func (c *AirCargoCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()

	// Air Cargo flight legs as specified in ParcelsApp v4 specification
	legs := []models.FlightLeg{
		{
			FlightNumber: "QR-9052 B",
			Origin:       "BTS",
			Destination:  "BUD",
			Status:       "Arrived",
		},
		{
			FlightNumber: "QR-8242",
			Origin:       "BUD",
			Destination:  "DOH",
			Status:       "Arrived",
		},
		{
			FlightNumber:           "QR-8416",
			Origin:                 "DOH",
			Destination:            "ALA",
			Status:                 "Departed",
			ScheduledDepartureDate: now.Format("2006-01-02"),
			ScheduledDeparture:     now.Add(-2 * time.Hour).Format(time.RFC3339),
			ScheduledArrival:       now.Add(4 * time.Hour).Format(time.RFC3339),
			ActualDeparture:        now.Add(-2 * time.Hour).Format(time.RFC3339),
		},
	}

	events := []models.TrackingEvent{
		{
			Date:        now.Add(-12 * time.Hour),
			Description: "Booked (BKG)",
			Location:    "DOH - Hamad International Airport, Qatar",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
			EventCode:   "BKG",
		},
		{
			Date:        now.Add(-2 * time.Hour),
			Description: "Departed (DEP) on flight QR-8416 to Almaty",
			Location:    "DOH - Hamad International Airport, Qatar",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
			EventCode:   "DEP",
		},
	}

	return &models.Shipment{
		TrackingNumber: trackingNumber,
		Carrier:        c.Info(),
		Origin: models.CountryInfo{
			Name: "Bratislava, Slovakia",
			Code: "BTS",
		},
		Destination: models.CountryInfo{
			Name: "Almaty, Kazakhstan",
			Code: "ALA",
		},
		Status:        models.StatusInTransit,
		StatusText:    "In Flight / In Transit",
		Progress:      60,
		DaysInTransit: 1,
		Events:        events,
		FlightLegs:    legs,
		UpdatedAt:     time.Now(),
	}, nil
}
