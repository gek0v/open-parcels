package engine

import (
	"testing"
	"time"

	"open-parcels/pkg/models"
)

func TestMergeShipmentsTimezoneOrderingMixedNoOffset(t *testing.T) {
	// Primary shipment: Cainiao (China, Asia/Shanghai, UTC+8)
	// Timestamps in raw string without timezone offset.
	primary := &models.Shipment{
		TrackingNumber: "LP001234567890CN",
		Carrier: models.CarrierInfo{
			Slug:    "cainiao",
			Name:    "Cainiao Global",
			Country: "CN",
		},
		Origin: models.CountryInfo{
			Name: "China",
			Code: "CN",
		},
		Events: []models.TrackingEvent{
			{
				RawDate:     "2024-03-15 10:00:00",
				Date:        time.Date(2024, 3, 15, 10, 0, 0, 0, time.UTC), // naive parse without offset
				Description: "Departed sorting hub",
				Location:    "Shanghai, China",
				CarrierSlug: "cainiao",
				CarrierName: "Cainiao Global",
				Status:      models.StatusInTransit,
				Stage:       models.StageTransit,
			},
			{
				RawDate:     "2024-03-14 16:30:00",
				Date:        time.Date(2024, 3, 14, 16, 30, 0, 0, time.UTC), // naive parse without offset
				Description: "Parcel collected from merchant",
				Location:    "Shenzhen, China",
				CarrierSlug: "cainiao",
				CarrierName: "Cainiao Global",
				Status:      models.StatusInTransit,
				Stage:       models.StagePickedUp,
			},
		},
	}

	// Secondary shipment: Correos Express (Spain, Europe/Madrid, CET = UTC+1 in March)
	// Timestamps in raw string without timezone offset.
	secondary := &models.Shipment{
		TrackingNumber: "6900123456789012",
		Carrier: models.CarrierInfo{
			Slug:    "correosexpress",
			Name:    "Correos Express",
			Country: "ES",
		},
		Destination: models.CountryInfo{
			Name: "Madrid",
			Code: "ES",
		},
		Events: []models.TrackingEvent{
			{
				RawDate:     "2024-03-15 11:30:00",
				Date:        time.Date(2024, 3, 15, 11, 30, 0, 0, time.UTC), // naive parse without offset
				Description: "Entregado en destino",
				Location:    "Madrid, Spain",
				CarrierSlug: "correosexpress",
				CarrierName: "Correos Express",
				Status:      models.StatusDelivered,
				Stage:       models.StageDelivery,
			},
			{
				RawDate:     "2024-03-15 09:00:00",
				Date:        time.Date(2024, 3, 15, 9, 0, 0, 0, time.UTC), // naive parse without offset
				Description: "En reparto con mensajero",
				Location:    "Coslada, Madrid, Spain",
				CarrierSlug: "correosexpress",
				CarrierName: "Correos Express",
				Status:      models.StatusOutForDelivery,
				Stage:       models.StageLastMile,
			},
		},
	}

	// Merge shipments
	merged := MergeShipments(primary, secondary)
	if merged == nil {
		t.Fatalf("expected non-nil merged shipment")
	}

	if len(merged.Events) != 4 {
		t.Fatalf("expected 4 merged events, got %d", len(merged.Events))
	}

	// Calculate expected UTC instants:
	// Event A: Madrid 11:30:00 CET (UTC+1) -> 10:30:00 UTC
	// Event B: Madrid 09:00:00 CET (UTC+1) -> 08:00:00 UTC
	// Event C: Shanghai 10:00:00 CST (UTC+8) -> 02:00:00 UTC
	// Event D: Shenzhen 16:30:00 CST (UTC+8) -> 08:30:00 UTC on 2024-03-14
	expectedTimes := []time.Time{
		time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC),
		time.Date(2024, 3, 15, 8, 0, 0, 0, time.UTC),
		time.Date(2024, 3, 15, 2, 0, 0, 0, time.UTC),
		time.Date(2024, 3, 14, 8, 30, 0, 0, time.UTC),
	}

	for i, expected := range expectedTimes {
		actual := merged.Events[i].Date
		if !actual.Equal(expected) {
			t.Errorf("event %d: expected UTC time %v, got %v (desc: %s)",
				i, expected, actual, merged.Events[i].Description)
		}
	}

	// Critical check: Ensure Madrid 09:00:00 CET (08:00 UTC) is sorted BEFORE Shanghai 10:00:00 CST (02:00 UTC)
	// despite Shanghai's naive wall-clock hour (10) being numerically greater than Madrid's (9).
	madridOutForDelivery := merged.Events[1]
	shanghaiDeparted := merged.Events[2]

	if !madridOutForDelivery.Date.After(shanghaiDeparted.Date) {
		t.Fatalf("CRITICAL: Madrid event at 09:00 CET (%v UTC) must be AFTER Shanghai event at 10:00 CST (%v UTC)",
			madridOutForDelivery.Date, shanghaiDeparted.Date)
	}

	// Verify descending order
	for i := 0; i < len(merged.Events)-1; i++ {
		if !merged.Events[i].Date.After(merged.Events[i+1].Date) {
			t.Errorf("events at index %d and %d are not in strict descending order: %v vs %v",
				i, i+1, merged.Events[i].Date, merged.Events[i+1].Date)
		}
	}

	// Verify overall status and progress
	if merged.Status != models.StatusDelivered {
		t.Errorf("expected merged status Delivered, got %s", merged.Status)
	}
	if merged.Progress != 100 {
		t.Errorf("expected progress 100, got %d", merged.Progress)
	}
	if merged.DeliveredAt == nil || !merged.DeliveredAt.Equal(expectedTimes[0]) {
		t.Errorf("expected DeliveredAt %v, got %v", expectedTimes[0], merged.DeliveredAt)
	}

	// Verify AdditionalCarriers
	hasCorreosExpress := false
	for _, ac := range merged.AdditionalCarriers {
		if ac.Slug == "correosexpress" {
			hasCorreosExpress = true
			break
		}
	}
	if !hasCorreosExpress {
		t.Errorf("expected Correos Express in AdditionalCarriers")
	}
}

func TestMergeShipmentsSummerTimeCEST(t *testing.T) {
	// July test: Europe/Madrid is CEST (UTC+2), Asia/Shanghai is CST (UTC+8)
	primary := &models.Shipment{
		TrackingNumber: "LP009876543210CN",
		Carrier: models.CarrierInfo{
			Slug:    "cainiao",
			Country: "CN",
		},
		Events: []models.TrackingEvent{
			{
				RawDate:     "2024-07-20 12:00:00",
				Date:        time.Date(2024, 7, 20, 12, 0, 0, 0, time.UTC),
				Description: "Dispatched from customs warehouse",
				Location:    "Shanghai, China",
				CarrierSlug: "cainiao",
			},
		},
	}

	secondary := &models.Shipment{
		TrackingNumber: "6900999988887777",
		Carrier: models.CarrierInfo{
			Slug:    "correosexpress",
			Country: "ES",
		},
		Events: []models.TrackingEvent{
			{
				RawDate:     "2024-07-20 10:00:00",
				Date:        time.Date(2024, 7, 20, 10, 0, 0, 0, time.UTC),
				Description: "Llegada a nave de destino",
				Location:    "Madrid, Spain",
				CarrierSlug: "correosexpress",
			},
		},
	}

	merged := MergeShipments(primary, secondary)
	if len(merged.Events) != 2 {
		t.Fatalf("expected 2 merged events, got %d", len(merged.Events))
	}

	// Madrid 10:00 CEST (UTC+2) = 08:00 UTC
	// Shanghai 12:00 CST (UTC+8) = 04:00 UTC
	// Therefore Madrid event (08:00 UTC) must be index 0, Shanghai event (04:00 UTC) must be index 1.
	if !merged.Events[0].Date.After(merged.Events[1].Date) {
		t.Fatalf("expected Madrid summer event (%v UTC) to be after Shanghai event (%v UTC)",
			merged.Events[0].Date, merged.Events[1].Date)
	}

	expectedMadridUTC := time.Date(2024, 7, 20, 8, 0, 0, 0, time.UTC)
	expectedShanghaiUTC := time.Date(2024, 7, 20, 4, 0, 0, 0, time.UTC)

	if !merged.Events[0].Date.Equal(expectedMadridUTC) {
		t.Errorf("expected Madrid event at %v, got %v", expectedMadridUTC, merged.Events[0].Date)
	}
	if !merged.Events[1].Date.Equal(expectedShanghaiUTC) {
		t.Errorf("expected Shanghai event at %v, got %v", expectedShanghaiUTC, merged.Events[1].Date)
	}
}

func TestMergeShipmentsNilCases(t *testing.T) {
	if res := MergeShipments(nil, nil); res != nil {
		t.Errorf("expected nil when both shipments are nil")
	}

	shp := &models.Shipment{
		TrackingNumber: "TEST123",
		Carrier:        models.CarrierInfo{Slug: "cainiao", Country: "CN"},
		Events: []models.TrackingEvent{
			{
				RawDate:     "2024-03-15 10:00:00",
				Description: "Test event",
				Location:    "Shanghai, China",
			},
		},
	}

	resPrim := MergeShipments(shp, nil)
	if resPrim == nil || len(resPrim.Events) != 1 {
		t.Fatalf("expected single merged event from primary alone")
	}
	expectedUTC := time.Date(2024, 3, 15, 2, 0, 0, 0, time.UTC)
	if !resPrim.Events[0].Date.Equal(expectedUTC) {
		t.Errorf("expected normalized UTC date %v, got %v", expectedUTC, resPrim.Events[0].Date)
	}

	resSec := MergeShipments(nil, shp)
	if resSec == nil || len(resSec.Events) != 1 {
		t.Fatalf("expected single merged event from secondary alone")
	}
	if !resSec.Events[0].Date.Equal(expectedUTC) {
		t.Errorf("expected normalized UTC date %v, got %v", expectedUTC, resSec.Events[0].Date)
	}
}

func TestMergeShipmentsWithoutRawDate(t *testing.T) {
	// Events without RawDate, passing naive time.Date in UTC
	primary := &models.Shipment{
		TrackingNumber: "LP0011223344",
		Carrier:        models.CarrierInfo{Slug: "cainiao", Country: "CN"},
		Events: []models.TrackingEvent{
			{
				Date:        time.Date(2024, 3, 15, 10, 0, 0, 0, time.UTC),
				Description: "Departed China",
				CarrierSlug: "cainiao",
			},
		},
	}
	secondary := &models.Shipment{
		TrackingNumber: "690011223344",
		Carrier:        models.CarrierInfo{Slug: "correosexpress", Country: "ES"},
		Events: []models.TrackingEvent{
			{
				Date:        time.Date(2024, 3, 15, 9, 0, 0, 0, time.UTC),
				Description: "En reparto",
				CarrierSlug: "correosexpress",
			},
		},
	}

	merged := MergeShipments(primary, secondary)
	if len(merged.Events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(merged.Events))
	}

	// Spain 09:00 CET (08:00 UTC) must come before China 10:00 CST (02:00 UTC)
	if !merged.Events[0].Date.Equal(time.Date(2024, 3, 15, 8, 0, 0, 0, time.UTC)) {
		t.Errorf("expected Madrid event at 08:00 UTC first, got %v", merged.Events[0].Date)
	}
	if !merged.Events[1].Date.Equal(time.Date(2024, 3, 15, 2, 0, 0, 0, time.UTC)) {
		t.Errorf("expected Shanghai event at 02:00 UTC second, got %v", merged.Events[1].Date)
	}
}

func TestMergeShipmentsNoDoubleShift(t *testing.T) {
	// Events already parsed via ParseLocalTimeWithZone (already UTC) with RawDate preserved
	cnTime, err := ParseLocalTimeWithZone("2006-01-02 15:04:05", "2024-03-15 10:00:00", "Asia/Shanghai")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	esTime, err := ParseLocalTimeWithZone("2006-01-02 15:04:05", "2024-03-15 09:00:00", "Europe/Madrid")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	primary := &models.Shipment{
		TrackingNumber: "LP0011223344",
		Carrier:        models.CarrierInfo{Slug: "cainiao", Country: "CN"},
		Events: []models.TrackingEvent{
			{
				Date:        cnTime,
				RawDate:     "2024-03-15 10:00:00",
				Description: "Departed China",
				CarrierSlug: "cainiao",
			},
		},
	}
	secondary := &models.Shipment{
		TrackingNumber: "690011223344",
		Carrier:        models.CarrierInfo{Slug: "correosexpress", Country: "ES"},
		Events: []models.TrackingEvent{
			{
				Date:        esTime,
				RawDate:     "2024-03-15 09:00:00",
				Description: "En reparto",
				CarrierSlug: "correosexpress",
			},
		},
	}

	merged := MergeShipments(primary, secondary)
	if len(merged.Events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(merged.Events))
	}

	if !merged.Events[0].Date.Equal(esTime) {
		t.Errorf("expected Madrid event unchanged at %v, got %v", esTime, merged.Events[0].Date)
	}
	if !merged.Events[1].Date.Equal(cnTime) {
		t.Errorf("expected Shanghai event unchanged at %v, got %v", cnTime, merged.Events[1].Date)
	}
}
