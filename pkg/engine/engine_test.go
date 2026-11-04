// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package engine

import (
	"context"
	"testing"
	"time"

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

	// Test ToParcelsAppResponse with empty request ID
	pRespEmptyReq := eng.ToParcelsAppResponse(shipment, "")
	if pRespEmptyReq.RequestID == "" {
		t.Errorf("expected generated request ID")
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

func TestEngineAccessorsAndErrors(t *testing.T) {
	eng := New()

	if eng.Detector() == nil {
		t.Errorf("expected detector to be non-nil")
	}
	if eng.Registry() == nil {
		t.Errorf("expected registry to be non-nil")
	}

	// Empty tracking number
	_, err := eng.Track(context.Background(), "", models.TrackOptions{})
	if err == nil {
		t.Errorf("expected error for empty tracking number")
	}

	// Unknown carrier hint
	_, err = eng.Track(context.Background(), "123456", models.TrackOptions{CarrierHint: "non-existent-carrier"})
	if err == nil {
		t.Errorf("expected error for unsupported carrier hint")
	}

	// Valid carrier hint
	shipment, err := eng.Track(context.Background(), "123456", models.TrackOptions{CarrierHint: "ups"})
	if err != nil {
		t.Fatalf("expected successful tracking with carrier hint: %v", err)
	}
	if shipment.Carrier.Slug != "ups" {
		t.Errorf("expected carrier ups, got %s", shipment.Carrier.Slug)
	}

	// Undetectable tracking number without hint
	_, err = eng.Track(context.Background(), "???---", models.TrackOptions{})
	if err == nil {
		t.Errorf("expected error for undetectable tracking number")
	}
}

func TestMergerAndOverallStatus(t *testing.T) {
	now := time.Now()

	// Empty events -> StatusPending
	if s := DeriveOverallStatus(nil); s != models.StatusPending {
		t.Errorf("expected pending for empty events, got %v", s)
	}

	// Delivered event anywhere in history
	evDelivered := []models.TrackingEvent{
		{Date: now, Status: models.StatusInTransit, Stage: models.StageTransit},
		{Date: now.Add(-time.Hour), Status: models.StatusDelivered, Stage: models.StageDelivery},
	}
	if s := DeriveOverallStatus(evDelivered); s != models.StatusDelivered {
		t.Errorf("expected delivered status, got %v", s)
	}

	// Out for delivery
	evOFD := []models.TrackingEvent{
		{Date: now, Status: models.StatusOutForDelivery, Stage: models.StageLastMile},
	}
	if s := DeriveOverallStatus(evOFD); s != models.StatusOutForDelivery {
		t.Errorf("expected out_for_delivery status, got %v", s)
	}

	// Failed attempt
	evFailed := []models.TrackingEvent{
		{Date: now, Status: models.StatusFailedAttempt, Stage: models.StageLastMile},
	}
	if s := DeriveOverallStatus(evFailed); s != models.StatusFailedAttempt {
		t.Errorf("expected failed_attempt status, got %v", s)
	}

	// Exception
	evException := []models.TrackingEvent{
		{Date: now, Status: models.StatusException, Stage: models.StageException},
	}
	if s := DeriveOverallStatus(evException); s != models.StatusException {
		t.Errorf("expected exception status, got %v", s)
	}

	// Deduplication test in MergeEvents
	primary := []models.TrackingEvent{
		{Date: now, CarrierSlug: "ctt", Description: "Objeto em distribuição"},
		{Date: now.Add(-30 * time.Minute), CarrierSlug: "ctt", Description: "Objeto em distribuição"},
	}
	secondary := []models.TrackingEvent{
		{Date: now.Add(-time.Hour), CarrierSlug: "dpd", Description: "Encomenda recebida no centro"},
	}
	merged := MergeEvents(primary, secondary)
	if len(merged) != 2 {
		t.Errorf("expected duplicate within 2h to be removed, got %d events", len(merged))
	}
}

func TestProgressCalculations(t *testing.T) {
	now := time.Now()

	if p := CalculateProgress(models.StatusDelivered, nil); p != 100 {
		t.Errorf("expected 100, got %d", p)
	}
	if p := CalculateProgress(models.StatusOutForDelivery, nil); p != 85 {
		t.Errorf("expected 85, got %d", p)
	}
	if p := CalculateProgress(models.StatusFailedAttempt, nil); p != 80 {
		t.Errorf("expected 80, got %d", p)
	}
	if p := CalculateProgress(models.StatusInfoReceived, nil); p != 15 {
		t.Errorf("expected 15, got %d", p)
	}
	if p := CalculateProgress(models.StatusPending, nil); p != 5 {
		t.Errorf("expected 5, got %d", p)
	}
	if p := CalculateProgress(models.StatusException, nil); p != 60 {
		t.Errorf("expected 60, got %d", p)
	}
	if p := CalculateProgress(models.Status("unknown"), nil); p != 30 {
		t.Errorf("expected 30, got %d", p)
	}

	// StatusInTransit with stages
	handoverEv := []models.TrackingEvent{{Stage: models.StageHandover}}
	if p := CalculateProgress(models.StatusInTransit, handoverEv); p != 75 {
		t.Errorf("expected 75, got %d", p)
	}

	customsEv := []models.TrackingEvent{{Stage: models.StageCustoms}}
	if p := CalculateProgress(models.StatusInTransit, customsEv); p != 55 {
		t.Errorf("expected 55, got %d", p)
	}

	transitEv := []models.TrackingEvent{{Stage: models.StageTransit}}
	if p := CalculateProgress(models.StatusInTransit, transitEv); p != 35 {
		t.Errorf("expected 35, got %d", p)
	}

	emptyInTransitEv := []models.TrackingEvent{}
	if p := CalculateProgress(models.StatusInTransit, emptyInTransitEv); p != 25 {
		t.Errorf("expected 25, got %d", p)
	}

	// Days in transit
	if d := CalculateDaysInTransit(nil, nil); d != 0 {
		t.Errorf("expected 0 for empty events, got %d", d)
	}

	deliveredTime := now.Add(48 * time.Hour)
	events := []models.TrackingEvent{
		{Date: now},
		{Date: now.Add(24 * time.Hour)},
	}
	days := CalculateDaysInTransit(events, &deliveredTime)
	if days < 2 {
		t.Errorf("expected at least 2 days, got %d", days)
	}

	// Event today (less than 24 hours)
	shortEvents := []models.TrackingEvent{
		{Date: now.Add(-2 * time.Hour)},
	}
	if d := CalculateDaysInTransit(shortEvents, nil); d != 1 {
		t.Errorf("expected 1 day for recent event, got %d", d)
	}
}
