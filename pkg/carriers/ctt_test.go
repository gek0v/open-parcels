package carriers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"open-parcels/pkg/models"
)

func TestCTTExpressLiveTrackingFlag(t *testing.T) {
	c := NewCTTExpressCarrier()
	if !c.HasLiveTracking() {
		t.Fatalf("expected CTT Express HasLiveTracking() to be true")
	}
	if !HasLiveTracking(c) {
		t.Fatalf("expected carriers.HasLiveTracking(ctt-express) to be true")
	}
}

func TestCTTExpressDeliveredWithHTTPTestServer(t *testing.T) {
	fixture, err := LoadFixture("ctt_delivered.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	teardownURL := SetCTTExpressBaseURLForTest(srv.URL)
	defer teardownURL()
	teardownClient := SetCTTExpressHTTPClientForTest(srv.Client())
	defer teardownClient()

	c := NewCTTExpressCarrier()
	shipment, err := c.Track(context.Background(), "0001234567890123456789", models.TrackOptions{
		Language:       "es",
		AllowSimulated: false,
	})
	if err != nil {
		t.Fatalf("unexpected error tracking CTT Express delivered: %v", err)
	}

	if shipment.Status != models.StatusDelivered {
		t.Errorf("expected StatusDelivered, got %s", shipment.Status)
	}
	if shipment.Progress != 100 {
		t.Errorf("expected Progress 100, got %d", shipment.Progress)
	}
	if shipment.DeliveredAt == nil {
		t.Fatalf("expected DeliveredAt populated")
	}
	expectedDelivered := time.Date(2024, 11, 15, 16, 20, 0, 0, time.UTC)
	if !shipment.DeliveredAt.Equal(expectedDelivered) {
		t.Errorf("expected DeliveredAt %v, got %v", expectedDelivered, shipment.DeliveredAt)
	}
	if shipment.Origin.Name != "Madrid" {
		t.Errorf("expected Origin 'Madrid', got %q", shipment.Origin.Name)
	}
	if shipment.Destination.Name != "Sevilla" {
		t.Errorf("expected Destination 'Sevilla', got %q", shipment.Destination.Name)
	}
	if len(shipment.Events) != 4 {
		t.Fatalf("expected 4 events, got %d", len(shipment.Events))
	}
}

func TestCTTExpressOutForDeliveryWithHTTPTestServer(t *testing.T) {
	fixture, err := LoadFixture("ctt_out_for_delivery.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	teardownURL := SetCTTExpressBaseURLForTest(srv.URL)
	defer teardownURL()
	teardownClient := SetCTTExpressHTTPClientForTest(srv.Client())
	defer teardownClient()

	c := NewCTTExpressCarrier()
	shipment, err := c.Track(context.Background(), "0009876543210987654321", models.TrackOptions{
		AllowSimulated: false,
	})
	if err != nil {
		t.Fatalf("unexpected error tracking CTT Express out for delivery: %v", err)
	}

	if shipment.Status != models.StatusOutForDelivery {
		t.Errorf("expected StatusOutForDelivery, got %s", shipment.Status)
	}
	if shipment.Progress != 85 {
		t.Errorf("expected Progress 85, got %d", shipment.Progress)
	}
	if shipment.DeliveredAt != nil {
		t.Errorf("expected DeliveredAt nil")
	}
}

func TestCTTExpressExceptionHTMLWithHTTPTestServer(t *testing.T) {
	fixture, err := LoadFixture("ctt_exception.html")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	teardownURL := SetCTTExpressBaseURLForTest(srv.URL)
	defer teardownURL()
	teardownClient := SetCTTExpressHTTPClientForTest(srv.Client())
	defer teardownClient()

	c := NewCTTExpressCarrier()
	shipment, err := c.Track(context.Background(), "EXP1234567890", models.TrackOptions{
		AllowSimulated: false,
	})
	if err != nil {
		t.Fatalf("unexpected error tracking CTT Express HTML exception: %v", err)
	}

	if shipment.Status != models.StatusFailedAttempt {
		t.Errorf("expected StatusFailedAttempt, got %s", shipment.Status)
	}
	if shipment.Origin.Name != "M?laga" {
		t.Errorf("expected Origin 'M?laga', got %q", shipment.Origin.Name)
	}
	if shipment.Destination.Name != "Madrid" {
		t.Errorf("expected Destination 'Madrid', got %q", shipment.Destination.Name)
	}
}

func TestCTTExpressErrorsAndNoSilentMock(t *testing.T) {
	c := NewCTTExpressCarrier()

	t.Run("not_found", func(t *testing.T) {
		fixture, _ := LoadFixture("ctt_not_found.json")
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(fixture)
		}))
		defer srv.Close()

		teardownURL := SetCTTExpressBaseURLForTest(srv.URL)
		defer teardownURL()
		teardownClient := SetCTTExpressHTTPClientForTest(srv.Client())
		defer teardownClient()

		_, err := c.Track(context.Background(), "0000000000000000000000", models.TrackOptions{AllowSimulated: false})
		if !errors.Is(err, models.ErrShipmentNotFound) {
			t.Errorf("expected ErrShipmentNotFound, got %v", err)
		}
	})

	t.Run("blocked_waf", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("WAF block"))
		}))
		defer srv.Close()

		teardownURL := SetCTTExpressBaseURLForTest(srv.URL)
		defer teardownURL()
		teardownClient := SetCTTExpressHTTPClientForTest(srv.Client())
		defer teardownClient()

		_, err := c.Track(context.Background(), "0001234567890123456789", models.TrackOptions{AllowSimulated: false})
		if !errors.Is(err, models.ErrCarrierBlocked) {
			t.Errorf("expected ErrCarrierBlocked, got %v", err)
		}
	})

	t.Run("schema_changed", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"unexpectedField": true}`))
		}))
		defer srv.Close()

		teardownURL := SetCTTExpressBaseURLForTest(srv.URL)
		defer teardownURL()
		teardownClient := SetCTTExpressHTTPClientForTest(srv.Client())
		defer teardownClient()

		_, err := c.Track(context.Background(), "0001234567890123456789", models.TrackOptions{AllowSimulated: false})
		if !errors.Is(err, models.ErrScraperSchemaChanged) {
			t.Errorf("expected ErrScraperSchemaChanged, got %v", err)
		}
	})
}
