// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

type OceanContainerCarrier struct {
	BaseCarrier
}

func NewOceanContainerCarrier() *OceanContainerCarrier {
	return &OceanContainerCarrier{
		BaseCarrier: NewBaseCarrier(
			"ocean-container",
			"ISO Ocean Container",
			"GLOBAL",
			"International",
			models.CarrierTypeFreight,
			"https://www.bic-code.org",
			"",
		),
	}
}

func (c *OceanContainerCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()

	events := []models.TrackingEvent{
		{
			Date:        now.Add(-10 * 24 * time.Hour),
			Description: "Gate in full at origin port terminal",
			Location:    "Port of Shanghai, China (CNSHA)",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
			EventCode:   "GATE_IN",
		},
		{
			Date:        now.Add(-8 * 24 * time.Hour),
			Description: "Loaded on board vessel EVER GIVEN",
			Location:    "Port of Shanghai, China (CNSHA)",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
			EventCode:   "LOADED",
		},
		{
			Date:        now.Add(-7 * 24 * time.Hour),
			Description: "Vessel departed origin port / Underway",
			Location:    "Port of Shanghai, China (CNSHA)",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
			EventCode:   "DEPARTED",
		},
		{
			Date:        now.Add(-2 * 24 * time.Hour),
			Description: "Vessel in transit on high seas",
			Location:    "Mediterranean Sea",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
			EventCode:   "IN_TRANSIT",
		},
	}

	container := &models.ContainerDetails{
		Number:        trackingNumber,
		Type:          "40HC",
		ISOType:       "45G1",
		Status:        "Laden on Board",
		SealNumber:    "SL-908124",
		TareWeight:    "3,900 KG",
		PayloadWeight: "26,580 KG",
		GrossWeight:   "30,480 KG",
	}

	seaLeg := models.SeaLeg{
		Vessel: &models.VesselInfo{
			Name:     "EVER GIVEN",
			IMO:      "9811000",
			CallSign: "H3RC",
			Flag:     "PA",
		},
		VoyageNumber:           "0112-045W",
		Origin:                 "Shanghai, China",
		Destination:            "Valencia, Spain",
		OriginPort:             "Shanghai",
		DestinationPort:        "Valencia",
		OriginUNLOCODE:         "CNSHA",
		DestinationUNLOCODE:    "ESVLC",
		Status:                 "Underway",
		ScheduledDepartureDate: now.Add(-10 * 24 * time.Hour).Format("2006-01-02"),
		ScheduledDeparture:     now.Add(-8 * 24 * time.Hour).Format(time.RFC3339),
		ScheduledArrival:       now.Add(5 * 24 * time.Hour).Format(time.RFC3339),
		ActualDeparture:        now.Add(-8 * 24 * time.Hour).Format(time.RFC3339),
	}

	return &models.Shipment{
		TrackingNumber: trackingNumber,
		Carrier:        c.Info(),
		Origin: models.CountryInfo{
			Name: "Shanghai, China",
			Code: "CNSHA",
		},
		Destination: models.CountryInfo{
			Name: "Valencia, Spain",
			Code: "ESVLC",
		},
		FromPort:      "CNSHA",
		ToPort:        "ESVLC",
		Status:        models.StatusInTransit,
		StatusText:    "In Transit - At Sea",
		Progress:      50,
		DaysInTransit: 10,
		Container:     container,
		SeaLegs:       []models.SeaLeg{seaLeg},
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}
