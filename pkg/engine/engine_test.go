package engine

import (
	"context"
	"testing"

	"open-parcels/pkg/models"
)

func TestEngineTrackChaining(t *testing.T) {
	eng := New()

	// Track Cainiao number with handover to Correos Express
	shipment, err := eng.Track(context.Background(), "LP00561234567890", models.TrackOptions{
		DestinationCountry: "ES",
		Language:           "es",
	})
	if err != nil {
		t.Fatalf("expected successful tracking, got err: %v", err)
	}

	if shipment.Carrier.Slug != "cainiao" {
		t.Errorf("expected primary carrier cainiao, got %s", shipment.Carrier.Slug)
	}

	if len(shipment.AdditionalCarriers) == 0 {
		t.Errorf("expected additional carrier (Correos Express) chained for last mile")
	}

	if len(shipment.Events) < 5 {
		t.Errorf("expected merged timeline with events, got %d", len(shipment.Events))
	}

	// Verify chronological ordering: newest event first
	for i := 0; i < len(shipment.Events)-1; i++ {
		if shipment.Events[i].Date.Before(shipment.Events[i+1].Date) {
			t.Errorf("events not sorted in descending chronological order at index %d", i)
		}
	}

	// Check status normalization and progress
	if shipment.Progress <= 0 || shipment.Progress > 100 {
		t.Errorf("progress should be between 1 and 100, got %d", shipment.Progress)
	}

	if shipment.StatusText == "" {
		t.Errorf("expected localized status text, got empty")
	}

	// Test ParcelsApp v4 response conversion
	pResp := eng.ToParcelsAppResponse(shipment, "test-req-123")
	if pResp == nil || len(pResp.Results) == 0 {
		t.Fatalf("expected ParcelsApp response with results")
	}
	if pResp.Results[0].Shipment.TrackingID != "LP00561234567890" {
		t.Errorf("expected tracking ID in ParcelsApp shipment, got %s", pResp.Results[0].Shipment.TrackingID)
	}
	if len(pResp.Results[0].Shipment.Services) < 2 {
		t.Errorf("expected multiple services listed in ParcelsApp response for chained journey")
	}
}

func TestEngineAirCargo(t *testing.T) {
	eng := New()
	shipment, err := eng.Track(context.Background(), "157-52471764", models.TrackOptions{})
	if err != nil {
		t.Fatalf("unexpected error tracking AWB: %v", err)
	}

	if shipment.Carrier.Slug != "air-cargo" {
		t.Errorf("expected air-cargo carrier, got %s", shipment.Carrier.Slug)
	}

	if len(shipment.FlightLegs) == 0 {
		t.Errorf("expected flight legs for air cargo tracking")
	}
}

func TestEngineItalianTrackingAndLocalization(t *testing.T) {
	eng := New()

	shipment, err := eng.Track(context.Background(), "RA123456785IT", models.TrackOptions{
		DestinationCountry: "IT",
		Language:           "it",
	})
	if err != nil {
		t.Fatalf("unexpected error tracking Italian shipment: %v", err)
	}

	if shipment.Carrier.Slug != "poste-italiane" {
		t.Errorf("expected carrier poste-italiane, got %s", shipment.Carrier.Slug)
	}

	if shipment.StatusText != "In consegna" {
		t.Errorf("expected Italian localized status 'In consegna', got %s", shipment.StatusText)
	}

	// Verify normalizer for Italian events
	status, stage := NormalizeEventStatus("Pacco consegnato al destinatario")
	if status != models.StatusDelivered || stage != models.StageDelivery {
		t.Errorf("expected Delivered/Delivery, got %s/%s", status, stage)
	}

	status, stage = NormalizeEventStatus("Spedizione in giacenza")
	if status != models.StatusException || stage != models.StageException {
		t.Errorf("expected Exception/Exception, got %s/%s", status, stage)
	}
}

func TestEngineTracking323CainiaoAndCorreosExpress(t *testing.T) {
	eng := New()

	// 1. Auto-detection tracking: 32300035210642701471403 with ES language
	shipment, err := eng.Track(context.Background(), "32300035210642701471403", models.TrackOptions{
		Language: "es",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if shipment.Status != models.StatusDelivered {
		t.Errorf("expected status Delivered, got %s", shipment.Status)
	}
	if shipment.Progress != 100 {
		t.Errorf("expected progress 100, got %d", shipment.Progress)
	}
	if shipment.Carrier.Slug != "cainiao" {
		t.Errorf("expected primary carrier cainiao, got %s", shipment.Carrier.Slug)
	}

	// Verify Correos Express is in AdditionalCarriers
	hasCE := false
	for _, ac := range shipment.AdditionalCarriers {
		if ac.Slug == "correosexpress" {
			hasCE = true
			break
		}
	}
	if !hasCE {
		t.Errorf("expected Correos Express in AdditionalCarriers")
	}

	// Verify events show Correos Express for local delivery
	hasCorreosEvent := false
	for _, ev := range shipment.Events {
		if ev.CarrierSlug == "correosexpress" {
			hasCorreosEvent = true
			break
		}
	}
	if !hasCorreosEvent {
		t.Errorf("expected at least one event attributed to Correos Express")
	}

	// 2. Direct query with correosexpress hint
	ceShipment, err := eng.Track(context.Background(), "32300035210642701471403", models.TrackOptions{
		CarrierHint: "correosexpress",
		Language:    "es",
	})
	if err != nil {
		t.Fatalf("unexpected error with correosexpress hint: %v", err)
	}
	if ceShipment.Status != models.StatusDelivered {
		t.Errorf("expected status Delivered with correosexpress hint, got %s", ceShipment.Status)
	}
	if ceShipment.Progress != 100 {
		t.Errorf("expected progress 100 with correosexpress hint, got %d", ceShipment.Progress)
	}
	// Verify Cainiao is present
	hasCainiao := false
	if ceShipment.Carrier.Slug == "cainiao" {
		hasCainiao = true
	}
	for _, ac := range ceShipment.AdditionalCarriers {
		if ac.Slug == "cainiao" {
			hasCainiao = true
			break
		}
	}
	if !hasCainiao {
		t.Errorf("expected Cainiao Global represented when tracked with correosexpress hint")
	}
}


