package carriers

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"open-parcels/pkg/models"
)

func TestCorreosDeliveredFixture(t *testing.T) {
	fixtureData, err := LoadFixture("correos_delivered.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	teardown := SetCorreosHTTPClientForTest(NewFixtureClient(http.StatusOK, fixtureData, "application/json"))
	defer teardown()

	c := NewCorreosSpainCarrier()
	shipment, err := c.Track(context.Background(), "PQ4F6P07000000000000000", models.TrackOptions{
		Language: "es",
	})
	if err != nil {
		t.Fatalf("unexpected error tracking delivered fixture: %v", err)
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
	if len(shipment.Events) != 5 {
		t.Fatalf("expected 5 events, got %d", len(shipment.Events))
	}
	if shipment.Origin.Name != "España" {
		t.Errorf("expected Origin España, got %s", shipment.Origin.Name)
	}
	if shipment.Destination.Name != "España" {
		t.Errorf("expected Destination España, got %s", shipment.Destination.Name)
	}
}

func TestCorreosPickupOfficeFixture(t *testing.T) {
	fixtureData, err := LoadFixture("correos_pickup_office.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	teardown := SetCorreosHTTPClientForTest(NewFixtureClient(http.StatusOK, fixtureData, "application/json"))
	defer teardown()

	c := NewCorreosSpainCarrier()
	shipment, err := c.Track(context.Background(), "CD001234567ES", models.TrackOptions{
		Language: "es",
	})
	if err != nil {
		t.Fatalf("unexpected error tracking pickup office fixture: %v", err)
	}

	if !strings.Contains(shipment.StatusText, "OFICINA PRINCIPAL MADRID") {
		t.Errorf("expected status text to contain pickup office, got: %s", shipment.StatusText)
	}
	if !strings.Contains(shipment.StatusText, "2024-11-30") {
		t.Errorf("expected status text to contain deadline date 2024-11-30, got: %s", shipment.StatusText)
	}
}

func TestCorreosNotFoundFixture(t *testing.T) {
	fixtureData, err := LoadFixture("correos_not_found.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	teardown := SetCorreosHTTPClientForTest(NewFixtureClient(http.StatusOK, fixtureData, "application/json"))
	defer teardown()

	c := NewCorreosSpainCarrier()
	_, err = c.Track(context.Background(), "PQ000000000000000000000", models.TrackOptions{})
	if err == nil {
		t.Fatalf("expected ErrShipmentNotFound, got nil")
	}
	if !errors.Is(err, models.ErrShipmentNotFound) {
		t.Errorf("expected ErrShipmentNotFound, got: %v", err)
	}
}

func TestCorreosRateLimitAndUnavailable(t *testing.T) {
	c := NewCorreosSpainCarrier()

	// 1. Rate Limit HTTP 429
	teardown := SetCorreosHTTPClientForTest(NewFixtureClient(http.StatusTooManyRequests, []byte("Rate limit"), "text/plain"))
	_, err := c.Track(context.Background(), "PQ4F6P07000000000000000", models.TrackOptions{})
	teardown()
	if !errors.Is(err, models.ErrCarrierRateLimited) {
		t.Errorf("expected ErrCarrierRateLimited, got: %v", err)
	}

	// 2. HTTP 500
	teardown500 := SetCorreosHTTPClientForTest(NewFixtureClient(http.StatusInternalServerError, []byte("Server Error"), "text/plain"))
	_, err = c.Track(context.Background(), "PQ4F6P07000000000000000", models.TrackOptions{})
	teardown500()
	if !errors.Is(err, models.ErrCarrierTemporarilyUnavailable) {
		t.Errorf("expected ErrCarrierTemporarilyUnavailable, got: %v", err)
	}
}

func TestCorreosWAFBlockedAndBrokenSchema(t *testing.T) {
	c := NewCorreosSpainCarrier()

	// 1. HTTP 403 WAF Blocked
	teardown403 := SetCorreosHTTPClientForTest(NewFixtureClient(http.StatusForbidden, []byte("Forbidden - Akamai WAF Blocked"), "text/html"))
	_, err := c.Track(context.Background(), "PQ4F6P07000000000000000", models.TrackOptions{})
	teardown403()
	if !errors.Is(err, models.ErrCarrierBlocked) {
		t.Errorf("expected ErrCarrierBlocked for HTTP 403, got: %v", err)
	}

	// 2. Broken schema fixture
	brokenFixture, err := LoadFixture("correos_broken_schema.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	teardownSchema := SetCorreosHTTPClientForTest(NewFixtureClient(http.StatusOK, brokenFixture, "application/json"))
	_, err = c.Track(context.Background(), "PQ4F6P07000000000000000", models.TrackOptions{})
	teardownSchema()
	if !errors.Is(err, models.ErrScraperSchemaChanged) {
		t.Errorf("expected ErrScraperSchemaChanged for broken schema fixture, got: %v", err)
	}
}

