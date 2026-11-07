package carriers

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"open-parcels/pkg/models"
)

func TestCorreosExpressDeliveredFixture(t *testing.T) {
	fixtureData, err := LoadFixture("correosexpress_delivered.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	teardown := SetCorreosExpressHTTPClientForTest(NewFixtureClient(http.StatusOK, fixtureData, "application/json"))
	defer teardown()

	c := NewCorreosExpressCarrier()
	shipment, err := c.Track(context.Background(), "6900123456789012", models.TrackOptions{
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
	if len(shipment.Events) != 4 {
		t.Fatalf("expected 4 events, got %d", len(shipment.Events))
	}
	if shipment.Events[0].Status != models.StatusDelivered {
		t.Errorf("expected first event StatusDelivered, got %s", shipment.Events[0].Status)
	}
	if shipment.Origin.Name != "MADRID" {
		t.Errorf("expected Origin MADRID, got %s", shipment.Origin.Name)
	}
	if shipment.Destination.Name != "BARCELONA" {
		t.Errorf("expected Destination BARCELONA, got %s", shipment.Destination.Name)
	}
}

func TestCorreosExpressOutForDeliveryFixture(t *testing.T) {
	fixtureData, err := LoadFixture("correosexpress_out_for_delivery.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	teardown := SetCorreosExpressHTTPClientForTest(NewFixtureClient(http.StatusOK, fixtureData, "application/json"))
	defer teardown()

	c := NewCorreosExpressCarrier()
	shipment, err := c.Track(context.Background(), "6900987654321098", models.TrackOptions{
		Language: "es",
	})
	if err != nil {
		t.Fatalf("unexpected error tracking out for delivery fixture: %v", err)
	}

	if shipment.Status != models.StatusOutForDelivery {
		t.Errorf("expected StatusOutForDelivery, got %s", shipment.Status)
	}
	if shipment.Progress != 85 {
		t.Errorf("expected Progress 85, got %d", shipment.Progress)
	}
	if shipment.DeliveredAt != nil {
		t.Errorf("expected DeliveredAt nil for out for delivery shipment")
	}
}

func TestCorreosExpressNotFoundFixture(t *testing.T) {
	fixtureData, err := LoadFixture("correosexpress_not_found.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	teardown := SetCorreosExpressHTTPClientForTest(NewFixtureClient(http.StatusOK, fixtureData, "application/json"))
	defer teardown()

	c := NewCorreosExpressCarrier()
	_, err = c.Track(context.Background(), "6900000000000000", models.TrackOptions{})
	if err == nil {
		t.Fatalf("expected ErrShipmentNotFound, got nil")
	}
	if !errors.Is(err, models.ErrShipmentNotFound) {
		t.Errorf("expected ErrShipmentNotFound, got: %v", err)
	}
}

func TestCorreosExpressRateLimitAndUnavailable(t *testing.T) {
	c := NewCorreosExpressCarrier()

	// 1. Rate Limit HTTP 429
	teardown := SetCorreosExpressHTTPClientForTest(NewFixtureClient(http.StatusTooManyRequests, []byte("Rate limit"), "text/plain"))
	_, err := c.Track(context.Background(), "6900123456789012", models.TrackOptions{})
	teardown()
	if !errors.Is(err, models.ErrCarrierRateLimited) {
		t.Errorf("expected ErrCarrierRateLimited, got: %v", err)
	}

	// 2. HTTP 500
	teardown500 := SetCorreosExpressHTTPClientForTest(NewFixtureClient(http.StatusInternalServerError, []byte("Server Error"), "text/plain"))
	_, err = c.Track(context.Background(), "6900123456789012", models.TrackOptions{})
	teardown500()
	if !errors.Is(err, models.ErrCarrierTemporarilyUnavailable) {
		t.Errorf("expected ErrCarrierTemporarilyUnavailable, got: %v", err)
	}
}

func TestCorreosExpress323ChainingWithCainiao(t *testing.T) {
	// Simulate CE direct returning not found, but Cainiao having the handover record
	ceNotFound, _ := LoadFixture("correosexpress_not_found.json")
	teardownCE := SetCorreosExpressHTTPClientForTest(NewFixtureClient(http.StatusOK, ceNotFound, "application/json"))
	defer teardownCE()

	cainiaoDelivered, _ := LoadFixture("cainiao_delivered.json")
	teardownCainiao := SetCainiaoHTTPClientForTest(NewFixtureClient(http.StatusOK, cainiaoDelivered, "application/json"))
	defer teardownCainiao()

	c := NewCorreosExpressCarrier()
	shipment, err := c.Track(context.Background(), "32300035210642701471403", models.TrackOptions{
		Language: "es",
	})
	if err != nil {
		t.Fatalf("unexpected error tracking 323 shipment with Cainiao chaining: %v", err)
	}

	if shipment.Status != models.StatusDelivered {
		t.Errorf("expected StatusDelivered, got %s", shipment.Status)
	}
	if shipment.Carrier.Slug != "correosexpress" {
		t.Errorf("expected carrier slug correosexpress, got %s", shipment.Carrier.Slug)
	}
}
