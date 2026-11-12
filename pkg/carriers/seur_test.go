package carriers

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"open-parcels/pkg/models"
)

func TestSEURDeliveredFixture(t *testing.T) {
	fixtureData, err := LoadFixture("seur_delivered.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	teardown := SetSEURHTTPClientForTest(NewFixtureClient(http.StatusOK, fixtureData, "application/json"))
	defer teardown()

	c := NewSEURCarrier()
	shipment, err := c.Track(context.Background(), "36150127630406", models.TrackOptions{
		PostalCode: "08001",
		Language:   "es",
	})
	if err != nil {
		t.Fatalf("unexpected error tracking SEUR delivered: %v", err)
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
	if len(shipment.Events) != 4 {
		t.Fatalf("expected 4 events, got %d", len(shipment.Events))
	}
	if shipment.Origin.Name != "MADRID" {
		t.Errorf("expected Origin MADRID, got %s", shipment.Origin.Name)
	}
	if shipment.Destination.Name != "BARCELONA" {
		t.Errorf("expected Destination BARCELONA, got %s", shipment.Destination.Name)
	}
}

func TestSEURPostalCodeRequired(t *testing.T) {
	c := NewSEURCarrier()
	_, err := c.Track(context.Background(), "36150127630406", models.TrackOptions{
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

func TestSEURNotFoundFixture(t *testing.T) {
	fixtureData, err := LoadFixture("seur_not_found.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	teardown := SetSEURHTTPClientForTest(NewFixtureClient(http.StatusOK, fixtureData, "application/json"))
	defer teardown()

	c := NewSEURCarrier()
	_, err = c.Track(context.Background(), "36150000000000", models.TrackOptions{
		PostalCode: "28001",
	})
	if err == nil {
		t.Fatalf("expected ErrShipmentNotFound, got nil")
	}
	if !errors.Is(err, models.ErrShipmentNotFound) {
		t.Errorf("expected ErrShipmentNotFound, got: %v", err)
	}
}

func TestSEURRateLimitAndUnavailable(t *testing.T) {
	c := NewSEURCarrier()

	// 1. Rate Limit HTTP 429
	teardown := SetSEURHTTPClientForTest(NewFixtureClient(http.StatusTooManyRequests, []byte("Rate limit"), "text/plain"))
	_, err := c.Track(context.Background(), "36150127630406", models.TrackOptions{PostalCode: "28001"})
	teardown()
	if !errors.Is(err, models.ErrCarrierRateLimited) {
		t.Errorf("expected ErrCarrierRateLimited, got: %v", err)
	}

	// 2. HTTP 500
	teardown500 := SetSEURHTTPClientForTest(NewFixtureClient(http.StatusInternalServerError, []byte("Server Error"), "text/plain"))
	_, err = c.Track(context.Background(), "36150127630406", models.TrackOptions{PostalCode: "28001"})
	teardown500()
	if !errors.Is(err, models.ErrCarrierTemporarilyUnavailable) {
		t.Errorf("expected ErrCarrierTemporarilyUnavailable, got: %v", err)
	}
}

func TestSEURWAFBlockedAndBrokenSchema(t *testing.T) {
	c := NewSEURCarrier()

	// 1. HTTP 403 WAF Blocked
	teardown403 := SetSEURHTTPClientForTest(NewFixtureClient(http.StatusForbidden, []byte("Access Denied - Firewall"), "text/html"))
	_, err := c.Track(context.Background(), "36150127630406", models.TrackOptions{PostalCode: "08001"})
	teardown403()
	if !errors.Is(err, models.ErrCarrierBlocked) {
		t.Errorf("expected ErrCarrierBlocked for HTTP 403, got: %v", err)
	}

	// 2. Broken schema fixture
	brokenFixture, err := LoadFixture("seur_broken_schema.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	teardownSchema := SetSEURHTTPClientForTest(NewFixtureClient(http.StatusOK, brokenFixture, "application/json"))
	_, err = c.Track(context.Background(), "36150127630406", models.TrackOptions{PostalCode: "08001"})
	teardownSchema()
	if !errors.Is(err, models.ErrScraperSchemaChanged) {
		t.Errorf("expected ErrScraperSchemaChanged for broken schema fixture, got: %v", err)
	}
}

