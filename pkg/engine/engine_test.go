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

func TestEngineDetectorAndRegistry(t *testing.T) {
	eng := New()
	if eng.Detector() == nil {
		t.Errorf("expected non-nil detector")
	}
	if eng.Registry() == nil {
		t.Errorf("expected non-nil registry")
	}
}

func TestEngineTrackErrorCases(t *testing.T) {
	eng := New()
	ctx := context.Background()

	// Empty tracking number
	_, err := eng.Track(ctx, "", models.TrackOptions{})
	if err == nil {
		t.Errorf("expected error for empty tracking number")
	}

	// Unsupported carrier hint
	_, err = eng.Track(ctx, "123456789", models.TrackOptions{
		CarrierHint: "non-existent-carrier-slug-xyz",
	})
	if err == nil {
		t.Errorf("expected error for unsupported carrier hint")
	}

	// Unrecognized tracking number without hint
	_, err = eng.Track(ctx, "INVALID_NUMBER_XYZ_99999", models.TrackOptions{})
	if err == nil {
		t.Errorf("expected error for undetectable tracking number")
	}
}

func TestEngineTrackItalianCarrierWithHint(t *testing.T) {
	eng := New()
	ctx := context.Background()

	// Track Poste Italiane with hint and Italian language
	shipment, err := eng.Track(ctx, "386341123456", models.TrackOptions{
		CarrierHint: "poste-italiane",
		Language:    "it",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if shipment.Carrier.Slug != "poste-italiane" {
		t.Errorf("expected carrier poste-italiane, got %s", shipment.Carrier.Slug)
	}
	if shipment.StatusText != "In consegna" {
		t.Errorf("expected Italian status text 'In consegna', got %q", shipment.StatusText)
	}
}

func TestMergeEventsDeduplicationAndNormalization(t *testing.T) {
	now := time.Now()

	primary := []models.TrackingEvent{
		{
			Date:        now,
			Description: "Pacco consegnato al destinatario",
			CarrierSlug: "poste-italiane",
			// Status and Stage empty to test auto-normalization
		},
		{
			Date:        now.Add(-10 * time.Minute),
			Description: "Pacco consegnato al destinatario",
			CarrierSlug: "poste-italiane",
		},
		{
			Date:        now.Add(-1 * time.Hour),
			Description: "Spedizione in consegna nella giornata odierna",
			CarrierSlug: "poste-italiane",
			Status:      models.StatusOutForDelivery,
			Stage:       models.StageLastMile,
		},
	}

	secondary := []models.TrackingEvent{
		{
			Date:        now.Add(-3 * time.Hour),
			Description: "Preso in carico all'ufficio postale",
			CarrierSlug: "poste-italiane",
		},
	}

	merged := MergeEvents(primary, secondary)

	// Duplicate delivered event within 10 minutes should be deduplicated
	if len(merged) != 3 {
		t.Errorf("expected 3 events after deduplication, got %d", len(merged))
	}

	// First event should be delivered with auto-normalized status and stage
	if merged[0].Status != models.StatusDelivered || merged[0].Stage != models.StageDelivery {
		t.Errorf("expected first event to be delivered/delivery, got %s/%s", merged[0].Status, merged[0].Stage)
	}
}

func TestDeriveOverallStatus(t *testing.T) {
	// Empty events -> StatusPending
	if got := DeriveOverallStatus(nil); got != models.StatusPending {
		t.Errorf("expected StatusPending for empty events, got %v", got)
	}

	// Events containing delivered anywhere -> StatusDelivered
	eventsDelivered := []models.TrackingEvent{
		{Status: models.StatusInTransit},
		{Status: models.StatusDelivered},
	}
	if got := DeriveOverallStatus(eventsDelivered); got != models.StatusDelivered {
		t.Errorf("expected StatusDelivered, got %v", got)
	}

	// Latest OutForDelivery
	eventsOFD := []models.TrackingEvent{
		{Status: models.StatusOutForDelivery},
		{Status: models.StatusInTransit},
	}
	if got := DeriveOverallStatus(eventsOFD); got != models.StatusOutForDelivery {
		t.Errorf("expected StatusOutForDelivery, got %v", got)
	}

	// Latest FailedAttempt
	eventsFA := []models.TrackingEvent{
		{Status: models.StatusFailedAttempt},
		{Status: models.StatusInTransit},
	}
	if got := DeriveOverallStatus(eventsFA); got != models.StatusFailedAttempt {
		t.Errorf("expected StatusFailedAttempt, got %v", got)
	}

	// Latest Exception
	eventsExc := []models.TrackingEvent{
		{Status: models.StatusException},
		{Status: models.StatusInTransit},
	}
	if got := DeriveOverallStatus(eventsExc); got != models.StatusException {
		t.Errorf("expected StatusException, got %v", got)
	}

	// Latest InTransit
	eventsTransit := []models.TrackingEvent{
		{Status: models.StatusInTransit},
	}
	if got := DeriveOverallStatus(eventsTransit); got != models.StatusInTransit {
		t.Errorf("expected StatusInTransit, got %v", got)
	}

	// Empty latest status falls back to InTransit
	eventsEmptyStatus := []models.TrackingEvent{
		{Status: ""},
	}
	if got := DeriveOverallStatus(eventsEmptyStatus); got != models.StatusInTransit {
		t.Errorf("expected StatusInTransit fallback, got %v", got)
	}
}

func TestCalculateProgress(t *testing.T) {
	if got := CalculateProgress(models.StatusDelivered, nil); got != 100 {
		t.Errorf("expected 100 for delivered, got %d", got)
	}
	if got := CalculateProgress(models.StatusOutForDelivery, nil); got != 85 {
		t.Errorf("expected 85 for out for delivery, got %d", got)
	}
	if got := CalculateProgress(models.StatusFailedAttempt, nil); got != 80 {
		t.Errorf("expected 80 for failed attempt, got %d", got)
	}
	if got := CalculateProgress(models.StatusInfoReceived, nil); got != 15 {
		t.Errorf("expected 15 for info received, got %d", got)
	}
	if got := CalculateProgress(models.StatusPending, nil); got != 5 {
		t.Errorf("expected 5 for pending, got %d", got)
	}
	if got := CalculateProgress(models.StatusException, nil); got != 60 {
		t.Errorf("expected 60 for exception, got %d", got)
	}

	// In transit stages
	handoverEvents := []models.TrackingEvent{{Stage: models.StageHandover}}
	if got := CalculateProgress(models.StatusInTransit, handoverEvents); got != 75 {
		t.Errorf("expected 75 for handover, got %d", got)
	}

	customsEvents := []models.TrackingEvent{{Stage: models.StageCustoms}}
	if got := CalculateProgress(models.StatusInTransit, customsEvents); got != 55 {
		t.Errorf("expected 55 for customs, got %d", got)
	}

	transitEvents := []models.TrackingEvent{{Stage: models.StageTransit}}
	if got := CalculateProgress(models.StatusInTransit, transitEvents); got != 35 {
		t.Errorf("expected 35 for transit, got %d", got)
	}

	defaultTransitEvents := []models.TrackingEvent{}
	if got := CalculateProgress(models.StatusInTransit, defaultTransitEvents); got != 25 {
		t.Errorf("expected 25 for empty in transit, got %d", got)
	}

	if got := CalculateProgress(models.StatusExpired, nil); got != 30 {
		t.Errorf("expected 30 for default, got %d", got)
	}
}

func TestCalculateDaysInTransit(t *testing.T) {
	// Empty events -> 0
	if got := CalculateDaysInTransit(nil, nil); got != 0 {
		t.Errorf("expected 0 for empty events, got %d", got)
	}

	now := time.Now()
	// Single event recent (<1 day) -> 1
	recentEvents := []models.TrackingEvent{
		{Date: now.Add(-2 * time.Hour)},
	}
	if got := CalculateDaysInTransit(recentEvents, nil); got != 1 {
		t.Errorf("expected 1 for recent event, got %d", got)
	}

	// Delivered event 5 days later
	deliveredAt := now.Add(-24 * time.Hour)
	pastEvents := []models.TrackingEvent{
		{Date: now.Add(-6 * 24 * time.Hour)},
		{Date: now.Add(-3 * 24 * time.Hour)},
	}
	if got := CalculateDaysInTransit(pastEvents, &deliveredAt); got != 5 {
		t.Errorf("expected 5 days, got %d", got)
	}
}
