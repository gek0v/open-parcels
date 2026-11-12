package carriers

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"open-parcels/pkg/models"
)

func TestGLSSpainDeliveredFixture(t *testing.T) {
	fixtureData, err := LoadFixture("gls_spain_delivered.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	teardown := SetGLSSpainHTTPClientForTest(NewFixtureClient(http.StatusOK, fixtureData, "application/json"))
	defer teardown()

	c := NewGLSSpainCarrier()
	shipment, err := c.Track(context.Background(), "ASM123456789", models.TrackOptions{
		PostalCode: "41001",
		Language:   "es",
	})
	if err != nil {
		t.Fatalf("unexpected error tracking GLS Spain delivered: %v", err)
	}

	if shipment.Status != models.StatusDelivered {
		t.Errorf("expected StatusDelivered, got %s", shipment.Status)
	}
	if shipment.Progress != 100 {
		t.Errorf("expected Progress 100, got %d", shipment.Progress)
	}
	if shipment.DeliveredAt == nil {
		t.Errorf("expected DeliveredAt populated")
	}
	if len(shipment.Events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(shipment.Events))
	}
	if shipment.Origin.Name != "MADRID" {
		t.Errorf("expected Origin MADRID, got %s", shipment.Origin.Name)
	}
	if shipment.Destination.Name != "SEVILLA" {
		t.Errorf("expected Destination SEVILLA, got %s", shipment.Destination.Name)
	}
}

func TestGLSSpainPostalCodeRequired(t *testing.T) {
	c := NewGLSSpainCarrier()
	_, err := c.Track(context.Background(), "ASM123456789", models.TrackOptions{
		PostalCode:     "",
		AllowSimulated: false,
	})
	if err == nil {
		t.Fatalf("expected ErrPostalCodeRequired when postal code is missing")
	}
	if !errors.Is(err, models.ErrPostalCodeRequired) {
		t.Errorf("expected ErrPostalCodeRequired, got: %v", err)
	}
}

func TestGLSSpainDeduplication(t *testing.T) {
	fixtureData, err := LoadFixture("gls_spain_failed_retry.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	teardown := SetGLSSpainHTTPClientForTest(NewFixtureClient(http.StatusOK, fixtureData, "application/json"))
	defer teardown()

	c := NewGLSSpainCarrier()
	shipment, err := c.Track(context.Background(), "ASM987654321", models.TrackOptions{
		PostalCode: "28010",
		Language:   "es",
	})
	if err != nil {
		t.Fatalf("unexpected error tracking retry fixture: %v", err)
	}

	// Raw fixture has 2 duplicate failed attempts at 15:30 and 15:28 + 1 out for delivery
	// With deduplication, only 1 failed attempt should remain (total 2 events)
	if len(shipment.Events) != 2 {
		t.Errorf("expected deduplicated 2 events, got %d", len(shipment.Events))
	}
	if shipment.Status != models.StatusFailedAttempt {
		t.Errorf("expected StatusFailedAttempt, got %s", shipment.Status)
	}
}

func TestGLSSpainNotFound(t *testing.T) {
	fixtureData, err := LoadFixture("gls_spain_not_found.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	teardown := SetGLSSpainHTTPClientForTest(NewFixtureClient(http.StatusOK, fixtureData, "application/json"))
	defer teardown()

	c := NewGLSSpainCarrier()
	_, err = c.Track(context.Background(), "ASM000000000", models.TrackOptions{
		PostalCode: "28001",
	})
	if err == nil {
		t.Fatalf("expected ErrShipmentNotFound, got nil")
	}
	if !errors.Is(err, models.ErrShipmentNotFound) {
		t.Errorf("expected ErrShipmentNotFound, got: %v", err)
	}
}

func TestGLSSpainRateLimitAndUnavailable(t *testing.T) {
	c := NewGLSSpainCarrier()

	// 1. Rate Limit HTTP 429
	teardown := SetGLSSpainHTTPClientForTest(NewFixtureClient(http.StatusTooManyRequests, []byte("Rate limit"), "text/plain"))
	_, err := c.Track(context.Background(), "ASM123456789", models.TrackOptions{PostalCode: "41001"})
	teardown()
	if !errors.Is(err, models.ErrCarrierRateLimited) {
		t.Errorf("expected ErrCarrierRateLimited, got: %v", err)
	}

	// 2. HTTP 500
	teardown500 := SetGLSSpainHTTPClientForTest(NewFixtureClient(http.StatusInternalServerError, []byte("Server Error"), "text/plain"))
	_, err = c.Track(context.Background(), "ASM123456789", models.TrackOptions{PostalCode: "41001"})
	teardown500()
	if !errors.Is(err, models.ErrCarrierTemporarilyUnavailable) {
		t.Errorf("expected ErrCarrierTemporarilyUnavailable, got: %v", err)
	}
}

func TestGLSSpainWAFBlockedAndBrokenSchema(t *testing.T) {
	c := NewGLSSpainCarrier()

	// 1. HTTP 403 WAF Blocked
	teardown403 := SetGLSSpainHTTPClientForTest(NewFixtureClient(http.StatusForbidden, []byte("Forbidden - Incapsula/DataDome"), "text/html"))
	_, err := c.Track(context.Background(), "ASM123456789", models.TrackOptions{PostalCode: "41001"})
	teardown403()
	if !errors.Is(err, models.ErrCarrierBlocked) {
		t.Errorf("expected ErrCarrierBlocked for HTTP 403, got: %v", err)
	}

	// 2. Broken schema fixture
	brokenFixture, err := LoadFixture("gls_spain_broken_schema.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	teardownSchema := SetGLSSpainHTTPClientForTest(NewFixtureClient(http.StatusOK, brokenFixture, "application/json"))
	_, err = c.Track(context.Background(), "ASM123456789", models.TrackOptions{PostalCode: "41001"})
	teardownSchema()
	if !errors.Is(err, models.ErrScraperSchemaChanged) {
		t.Errorf("expected ErrScraperSchemaChanged for broken schema fixture, got: %v", err)
	}
}

