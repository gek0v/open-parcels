package carriers

import (
	"context"
	"errors"
	"net/http"
	"os"
	"testing"

	"open-parcels/pkg/models"
)

func TestDHLExpressDeliveredFixture(t *testing.T) {
	fixtureData, err := LoadFixture("dhl_express_delivered.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	teardown := SetDHLHTTPClientForTest(NewFixtureClient(http.StatusOK, fixtureData, "application/json"))
	defer teardown()

	os.Setenv("DHL_API_KEY", "test-dhl-key")
	defer os.Unsetenv("DHL_API_KEY")

	c := NewDHLExpressCarrier()
	shipment, err := c.Track(context.Background(), "1234567890", models.TrackOptions{
		Language: "en",
	})
	if err != nil {
		t.Fatalf("unexpected error tracking DHL Express fixture: %v", err)
	}

	if shipment.Status != models.StatusDelivered {
		t.Errorf("expected StatusDelivered, got %s", shipment.Status)
	}
	if shipment.Progress != 100 {
		t.Errorf("expected Progress 100, got %d", shipment.Progress)
	}
	if len(shipment.Events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(shipment.Events))
	}
	if shipment.Origin.Code != "DE" {
		t.Errorf("expected origin DE, got %s", shipment.Origin.Code)
	}
	if shipment.Destination.Code != "ES" {
		t.Errorf("expected destination ES, got %s", shipment.Destination.Code)
	}
}

func TestDHLParcelOutForDeliveryFixture(t *testing.T) {
	fixtureData, err := LoadFixture("dhl_parcel_out_for_delivery.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	teardown := SetDHLHTTPClientForTest(NewFixtureClient(http.StatusOK, fixtureData, "application/json"))
	defer teardown()

	os.Setenv("OPEN_PARCELS_DHL_KEY", "test-dhl-parcel-key")
	defer os.Unsetenv("OPEN_PARCELS_DHL_KEY")

	c := NewDHLParcelCarrier()
	shipment, err := c.Track(context.Background(), "JJD0000123456789012", models.TrackOptions{
		Language: "de",
	})
	if err != nil {
		t.Fatalf("unexpected error tracking DHL Parcel fixture: %v", err)
	}

	if shipment.Status != models.StatusOutForDelivery {
		t.Errorf("expected StatusOutForDelivery, got %s", shipment.Status)
	}
	if shipment.Progress != 85 {
		t.Errorf("expected Progress 85, got %d", shipment.Progress)
	}
	if len(shipment.Events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(shipment.Events))
	}
}

func TestDHLMissingAuth(t *testing.T) {
	os.Unsetenv("DHL_API_KEY")
	os.Unsetenv("OPEN_PARCELS_DHL_KEY")

	c := NewDHLExpressCarrier()
	_, err := c.Track(context.Background(), "1234567890", models.TrackOptions{
		AllowSimulated: false,
	})
	if err == nil {
		t.Fatalf("expected error when DHL API key is missing")
	}
	if !errors.Is(err, models.ErrCarrierNotImplemented) {
		t.Errorf("expected ErrCarrierNotImplemented, got: %v", err)
	}
}

func TestDHLNotFoundFixture(t *testing.T) {
	fixtureData, err := LoadFixture("dhl_not_found.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	teardown := SetDHLHTTPClientForTest(NewFixtureClient(http.StatusNotFound, fixtureData, "application/json"))
	defer teardown()

	os.Setenv("DHL_API_KEY", "test-dhl-key")
	defer os.Unsetenv("DHL_API_KEY")

	c := NewDHLExpressCarrier()
	_, err = c.Track(context.Background(), "0000000000", models.TrackOptions{})
	if err == nil {
		t.Fatalf("expected ErrShipmentNotFound, got nil")
	}
	if !errors.Is(err, models.ErrShipmentNotFound) {
		t.Errorf("expected ErrShipmentNotFound, got: %v", err)
	}
}

func TestDHLRateLimitAndAuthError(t *testing.T) {
	os.Setenv("DHL_API_KEY", "test-dhl-key")
	defer os.Unsetenv("DHL_API_KEY")

	c := NewDHLExpressCarrier()

	// 1. Rate Limit HTTP 429
	teardown429 := SetDHLHTTPClientForTest(NewFixtureClient(http.StatusTooManyRequests, []byte("Rate limit"), "text/plain"))
	_, err := c.Track(context.Background(), "1234567890", models.TrackOptions{})
	teardown429()
	if !errors.Is(err, models.ErrCarrierRateLimited) {
		t.Errorf("expected ErrCarrierRateLimited, got: %v", err)
	}

	// 2. Auth error HTTP 401
	teardown401 := SetDHLHTTPClientForTest(NewFixtureClient(http.StatusUnauthorized, []byte("Unauthorized"), "text/plain"))
	_, err = c.Track(context.Background(), "1234567890", models.TrackOptions{})
	teardown401()
	if !errors.Is(err, models.ErrCarrierNotImplemented) {
		t.Errorf("expected ErrCarrierNotImplemented for HTTP 401, got: %v", err)
	}
}
